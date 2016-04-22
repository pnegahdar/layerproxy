package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

var ErrorDNE = errors.New("Key selected does not exists")
var ErrorTooLarge = errors.New("Key size larger than cache size")

type CacheItem struct {
	File        *File
	Size        int
	accessCount uint64
}

type Cache struct {
	Data      []*CacheItem
	idx       map[string]int
	TotalSize int
	MaxSize   int
	sync.RWMutex
}

func (c *Cache) Len() int {
	return len(c.Data)
}

func (c *Cache) Swap(i, j int) {
	iKey, jKey := c.Data[i].File.Key, c.Data[j].File.Key
	c.Data[i], c.Data[j] = c.Data[j], c.Data[i]
	c.idx[iKey], c.idx[jKey] = j, i
}

func (c *Cache) Less(i, j int) bool {
	return c.Data[i].accessCount < c.Data[j].accessCount
}

func (c *Cache) Set(file *File) error {
	c.Lock()
	c.TotalSize += len(file.Contents)
	c.Unlock()
	c.free()
	c.Lock()
	defer c.Unlock()
	idx, ok := c.idx[file.Key]
	dataLen := len(file.Contents)
	if dataLen > c.MaxSize {
		log.Warning("Key is larger than cache.")
		return nil
	}
	var cacheItem *CacheItem
	if !ok {
		cacheItem = &CacheItem{accessCount: 0, File: file}
		c.Data = append(c.Data, cacheItem)
		idx = len(c.Data) - 1
		c.idx[file.Key] = idx

	} else {
		cacheItem = c.Data[idx]
	}
	cacheItem.Size = dataLen
	return nil
}

func (c *Cache) free() {
	c.Lock()
	defer c.Unlock()
	toRemove := c.TotalSize - c.MaxSize
	if toRemove <= 0 {
		return
	}
	removed := 0
	for i := len(c.Data) - 1; i >= 0; i-- {
		if removed >= toRemove {
			break
		} else {
			sort.Sort(sort.Reverse(c))
			item := c.Data[i]
			c.Unlock()
			err := c.Delete(item.File.Key)
			c.Lock()
			if err != nil && err != ErrorDNE {
				log.Warning(fmt.Sprintf("Ran into error clearing cache %v", err))
				panic(err)
			}
			removed += item.Size
		}
	}
}

func (c *Cache) Get(key string) (*File, error) {
	c.RLock()
	defer c.RUnlock()
	idx, ok := c.idx[key]
	if !ok {
		return nil, ErrorDNE
	}
	cacheItem := c.Data[idx]
	atomic.AddUint64(&cacheItem.accessCount, 1)
	return cacheItem.File, nil
}

func (c *Cache) List(prefix string) ([]*File, error) {
	c.RLock()
	defer c.RUnlock()
	files := []*File{}
	for _, cacheItem := range c.Data {
		if strings.HasPrefix(cacheItem.File.Key, prefix) {
			files = append(files, cacheItem.File)
		}
	}
	return files, nil
}

func (c *Cache) Delete(key string) error {
	c.Lock()
	defer c.Unlock()
	idxToRemove, ok := c.idx[key]
	if !ok {
		return ErrorDNE
	}
	// SWAP if needed
	if len(c.Data) > 1 && idxToRemove != len(c.Data)-1 {
		lastIdx := len(c.Data) - 1
		lastKey := c.Data[lastIdx].File.Key
		c.Data[lastIdx], c.Data[idxToRemove] = c.Data[idxToRemove], c.Data[lastIdx]
		c.idx[lastKey] = idxToRemove
	}
	c.TotalSize -= c.Data[idxToRemove].Size
	delete(c.idx, key)
	c.Data = c.Data[0 : len(c.Data)-1]
	return nil
}

func NewCache(maxSize int) *Cache {
	store := []*CacheItem{}
	idx := map[string]int{}
	return &Cache{Data: store, TotalSize: 0, MaxSize: maxSize, idx: idx}
}
