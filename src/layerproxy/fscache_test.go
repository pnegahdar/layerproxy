package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func makeFile(key, value string) *File {
	return &File{Key: key, Contents: value}
}

func TestFsCache(t *testing.T) {
	cache := NewFSCache()
	err := cache.Set(makeFile("test/b", "Test"))
	assert.Nil(t, err)

	file, err := cache.Get("test/dne")
	assert.Nil(t, file)
	assert.Equal(t, err, ErrorDNE)

	file, err = cache.Get("test/b")
	assert.Nil(t, err)
	assert.Equal(t, file.Contents, "Test")

	file = makeFile("test/b", "Blah")
	err = cache.Set(file)
	assert.Nil(t, err)
	file, err = cache.Get("test/b")
	assert.Nil(t, err)
	assert.Equal(t, file.Contents, "Blah")
}
