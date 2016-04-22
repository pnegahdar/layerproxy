package main
 
import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSet(t *testing.T) {
	maxSize := 200
	cache := NewCache(maxSize)
	err := cache.Set(makeFile("A", "DATA-A"))
	assert.Nil(t, err)
	result, err := cache.Get("A")
	assert.Nil(t, err)
	assert.Equal(t, result.Contents, "DATA-A")
	assert.Equal(t, cache.TotalSize, 6)
	err = cache.Set(makeFile("B", "DATA-B"))
	assert.Nil(t, err)
	assert.Equal(t, cache.TotalSize, 12)
}

func TestDelete(t *testing.T) {
	cache := NewCache(200)
	err := cache.Set(makeFile("A", "DATA-A"))
	assert.Nil(t, err)
	_, err = cache.Get("A")
	assert.Nil(t, err)
	t.Log("Size", len(cache.Data), cache.idx, cache.Data)
	err = cache.Delete("A")
	assert.Nil(t, err)

	result, err := cache.Get("A")
	assert.Nil(t, result)
	assert.Equal(t, cache.TotalSize, 0)

	cache.Delete("A")
	assert.Equal(t, err, ErrorDNE)

	err = cache.Set(makeFile("A", "DATA-A"))
	assert.Nil(t, err)
	err = cache.Set(makeFile("B", "DATA-B"))
	assert.Nil(t, err)
	err = cache.Set(makeFile("C", "DATA-C"))
	assert.Nil(t, err)
	assert.Equal(t, cache.TotalSize, 18)

	err = cache.Delete("B")
	assert.Nil(t, err)
	result, err = cache.Get("A")
	assert.Nil(t, err)
	assert.Equal(t, result.Contents, "DATA-A")
	result, err = cache.Get("C")
	assert.Nil(t, err)
	assert.Equal(t, result.Contents, "DATA-C")
	assert.Equal(t, cache.TotalSize, 12)
	assert.Equal(t, len(cache.idx), 2)
	assert.Equal(t, len(cache.Data), 2)
}

func TestEviction(t *testing.T) {
	cache := NewCache(24)
	keys := []string{"A", "B", "C", "D"}
	for _, key := range keys {
		err := cache.Set(makeFile(key, fmt.Sprintf("DATA-%v", key)))
		assert.Nil(t, err)

	}
	result, err := cache.Get("D")
	assert.Nil(t, err)
	assert.Equal(t, result.Contents, "DATA-D")

	cache.Set(makeFile("E", "DATA-E"))
	assert.Equal(t, len(cache.Data), 4)

	// Assure only C gets evicted
	cache.Get("B")
	for _, key := range []string{"A", "B", "E"} {
		_, err = cache.Get(key)
		assert.Nil(t, err)
	}
	cache.Set(makeFile("F", "123"))
	cache.Set(makeFile("G", "123"))
	assert.Equal(t, len(cache.Data), 5)
	t.Log(cache.idx)
	_, err = cache.Get("C")
	assert.Equal(t, err, ErrorDNE)
}
