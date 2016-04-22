package main 
import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/apsdehal/go-logger"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var log, _ = logger.New("main", 1, os.Stdout)

type File struct {
	Key      string
	Mtime    time.Time
	Contents string
}

var mimeSwaps = map[string]bool{
	".js":  true,
	".css": true}

var Epoc = time.Time{}

type Store interface {
	Get(key string) (*File, error)
	Set(*File) error
	List(prefix string) ([]*File, error)
	Delete(key string) error
}

type Manager struct {
	KeyIfNotFound string
	stores        []Store
	names         []string
	watchStores   []int
}

func (m *Manager) AddLayer(name string, store Store, watched bool) {
	m.stores = append(m.stores, store)
	m.names = append(m.names, name)
	if watched {
		m.watchStores = append(m.watchStores, len(m.names)-1)
	}
}

func (m *Manager) handlerSingle(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Path[1:]
	if fileName == "" {
		w.WriteHeader(200)
		io.WriteString(w, "No file requested in path.")
		return
	}
	file, err := m.Get(fileName)
	if err != nil {
		w.WriteHeader(500)
		io.WriteString(w, fmt.Sprintf("Ran into err: %v", err))
		return
	} else {
		fileExt := filepath.Ext(fileName)
		_, ok := mimeSwaps[fileExt]
		if ok {
			mimeType := mime.TypeByExtension(fileExt)
			w.Header().Set("Content-Type", mimeType)
		}
		w.WriteHeader(200)
		io.WriteString(w, file.Contents)
		return

	}

}

func (m *Manager) handlerMany(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	files := []string{}
	err := decoder.Decode(&files)
	for i, filename := range files {
		if strings.Contains(filename, "*") {
			prefixFiles, err := m.stores[0].List(strings.Replace(filename, "*", "", 1))
			prefixKeys := []string{}
			for _, prefixFile := range prefixFiles {
				prefixKeys = append(prefixKeys, prefixFile.Key)
			}
			if err != nil {
				w.WriteHeader(500)
				io.WriteString(w, fmt.Sprintf("Ran into err: %v", err))
				return
			}
			files = append(files[:i], files[i+1:]...) // Remove current spot
			files = append(files, prefixKeys...)
		}
	}
	if err != nil {
		w.WriteHeader(500)
		io.WriteString(w, fmt.Sprintf("Ran into err: %v", err))
		return
	}
	buf := new(bytes.Buffer)
	zipW := zip.NewWriter(buf)
	for _, filename := range files {
		f, err := zipW.Create(filename)
		if err != nil {
			log.Fatal(err.Error())
		}
		file, err := m.Get(filename)
		var contents string
		if err != nil {
			contents = fmt.Sprintf("Ran into err: %v", err)
		} else {
			contents = file.Contents
		}
		_, err = f.Write([]byte(contents))
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	err = zipW.Close()
	if err != nil {
		log.Fatal(err.Error())
	}
	buf.WriteTo(w)
	return
}

func (m *Manager) Get(key string) (*File, error) {
	var err error
	var file *File
	for i := len(m.stores) - 1; i >= 0; i-- {
		file, err = m.stores[i].Get(key)
		if err != nil && err != ErrorDNE {
			return nil, err
		}
		if err == ErrorDNE {
			log.Warning(fmt.Sprintf("MISS on store: %v for file %v", m.names[i], key))
			continue
		}
		log.Info(fmt.Sprintf("HIT on store: %v for file %v", m.names[i], key))
		for j := i + 1; j < len(m.stores); j++ {
			go m.stores[j].Set(file)
		}
		break
	}
	if err == ErrorDNE && m.KeyIfNotFound != "" && key != m.KeyIfNotFound {
		return m.Get(m.KeyIfNotFound)
	}
	return file, err
}

func (m *Manager) Delete(key string, watchedOnly bool) error {
	stores := []Store{}
	if watchedOnly {
		for _, idx := range m.watchStores {
			stores = append(stores, m.stores[idx])
		}
	} else {
		stores = m.stores
	}

	for i := len(stores) - 1; i >= 0; i-- {
		err := stores[i].Delete(key)
		if err != ErrorDNE {
			return err
		}
	}
	return nil
}

func (m *Manager) PrefetchPrefixes(prefix string) {
	files, err := m.stores[0].List(prefix)
	if err != nil {
		log.Warning(fmt.Sprintf("Ran into error %v", err))
		return
	}
	for _, file := range files {
		go func(key string) {
			_, err := m.Get(key)
			if err != nil {
				log.Warning(fmt.Sprintf("Ran into err: %v", err))
			}
		}(file.Key)
	}
}

func (m *Manager) Run(listen string, prefetchPrefixes []string, watchDelay int) {
	if len(m.stores) == 0 {
		panic("Please register at least one layer first.")
	}
	for _, prefix := range prefetchPrefixes {
		go m.PrefetchPrefixes(prefix)
	}
	http.HandleFunc("/_many/", m.handlerMany)
	http.HandleFunc("/", m.handlerSingle)
	log.Notice("Serving on " + listen)
	if watchDelay > 0 {
		go m.Watch(watchDelay)
	}
	err := http.ListenAndServe(listen, nil)
	if err != nil {
		panic(err)
	}
}

func (m *Manager) Watch(delay int) {
	log.Info(fmt.Sprintf("Watching every %v seconds.", delay))

	for {
	Watchloop:
		<-time.After(time.Duration(delay) * time.Second)
		log.Debug("Watch Running")
		// Create list of oldest files
		oldestFiles := map[string]*File{}
		for _, idx := range m.watchStores {
			files, err := m.stores[idx].List("")
			if err != nil {
				log.Warning(fmt.Sprintf("Ran into error updating (watch): %v", err))
				goto Watchloop
			}
			for _, file := range files {
				saveFile, ok := oldestFiles[file.Key]
				if !ok {
					oldestFiles[file.Key] = file
				} else {
					if file.Mtime.Before(saveFile.Mtime) {
						oldestFiles[file.Key] = file
					}
				}
			}
		}
		// Group them up by bucket for quick listing
		groups := map[string]bool{}
		for _, file := range oldestFiles {
			groupName := strings.Split(file.Key, "/")[0]
			if _, ok := groups[groupName]; !ok {
				groups[groupName] = true
			}
		}

		rootStoreFiles := map[string]*File{}
		for group, _ := range groups {
			files, err := m.stores[0].List(group)
			if err != nil {
				log.Warning(fmt.Sprintf("Ran into error listing group %v (watch): %v", group, err))
				continue
			}
			for _, f := range files {
				rootStoreFiles[f.Key] = f
			}
		}
		for _, oldFile := range oldestFiles {
			rootFile, ok := rootStoreFiles[oldFile.Key]
			// Delete removed files
			if !ok {
				log.Debug(fmt.Sprintf("Deleting removed file %+v.", oldFile))
				m.Delete(oldFile.Key, true)
				continue
			}
			// Update changed files
			if rootFile.Mtime.After(oldFile.Mtime) {
				log.Debug(fmt.Sprintf("Updating removed file %+v.", oldFile))
				m.Delete(oldFile.Key, true)
				m.Get(oldFile.Key)
				continue
			}
		}
		log.Debug("Watch done.")
	}

}
func NewManager(keyIfNotFound string) *Manager {
	stores := []Store{}
	return &Manager{stores: stores, KeyIfNotFound: keyIfNotFound}
}
