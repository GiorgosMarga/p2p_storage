package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath(t *testing.T) {
	s := NewCAS(CASOpts{
		RootPath:      "root_path",
		TransformFunc: TransformFunc,
	})
	assert.NotNil(t, s)
	key := "testkey"
	path := s.TransformFunc(key)
	transformedPath := "913a7/3b565/c8e2c/8ed94/49758/0f619/39770/9b8b6"
	fileName := "913a73b565c8e2c8ed94497580f619397709b8b6"
	assert.Equal(t, transformedPath, path.Path)
	assert.Equal(t, fileName, path.Filename)
}

func TestWrite(t *testing.T) {
	s := NewCAS(CASOpts{
		RootPath:      "root_path",
		TransformFunc: TransformFunc,
	})
	assert.NotNil(t, s)
	key := "testkey"
	data := []byte("test data")
	err := s.writeStream(key, bytes.NewReader(data))
	assert.Nil(t, err)

	assert.True(t, s.Has(key))
}

func TestRead(t *testing.T) {
	s := NewCAS(CASOpts{
		RootPath:      "root_path",
		TransformFunc: TransformFunc,
	})
	assert.NotNil(t, s)
	key := "testkey"
	data := []byte("test data")
	s.writeStream(key, bytes.NewReader(data))

	r, err := s.Read(key)
	assert.Nil(t, err)
	buf := make([]byte, 1024)
	r.Read(buf)
	fmt.Println("Data:", string(buf))
	assert.Nil(t, s.Delete(key))
	assert.Nil(t, s.Clear())
}
