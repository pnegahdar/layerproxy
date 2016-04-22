package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
)

type FsCache struct {
	RootDir string
	sync.RWMutex
}

func (fs *FsCache) Get(key string) (*File, error) {
	fs.RLock()
	defer fs.RUnlock()
	filename := filepath.Clean(fs.RootDir + key)
	stat, err := os.Stat(filename)
	if os.IsNotExist(err) || stat.IsDir() {
		return nil, ErrorDNE
	}
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		return nil, err
	}
	contents, err := ioutil.ReadAll(fp)
	if err != nil {
		return nil, err
	}
	file := &File{Key: key, Contents: string(contents), Mtime: stat.ModTime()}
	return file, nil
}

func (fs *FsCache) Set(file *File) error {
	fs.Lock()
	defer fs.Unlock()
	filename := filepath.Clean(fs.RootDir + file.Key)
	err := os.MkdirAll(filepath.Dir(filename), 0700)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filename, []byte(file.Contents), 0644)
	if file.Mtime != Epoc {
		err := os.Chtimes(filename, file.Mtime, file.Mtime)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (fs *FsCache) List(prefix string) ([]*File, error) {
	fs.RLock()
	defer fs.RUnlock()
	files := []*File{}
	filepath.Walk(fs.RootDir, func(path string, info os.FileInfo, err error) error {
		fnFixed, _ := filepath.Rel(fs.RootDir, path)
		if fnFixed == "." || info.IsDir() {
			return nil
		}
		if strings.HasPrefix(fnFixed, prefix) {
			files = append(files, &File{Key: fnFixed, Mtime: info.ModTime()})
		}
		return nil
	})
	for _, file := range files {
		if len(file.Key) < 3 {
			panic(file.Key)
		}
	}
	return files, nil
}

func (fs *FsCache) Delete(key string) error {
	fs.Lock()
	defer fs.Unlock()
	filename := filepath.Clean(fs.RootDir + key)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return ErrorDNE
	}
	err := os.Remove(filename)
	return err
}

func NewFSCache() *FsCache {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	dirAddon := "/.memprox/"
	return &FsCache{RootDir: user.HomeDir + dirAddon}
}
