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
	id := generateID()
	key := "testkey"
	data := []byte("test data")
	_, err := s.writeStream(key, id, bytes.NewReader(data))
	assert.Nil(t, err)

	assert.True(t, s.Has(key, id))

	err = s.Delete(key, id)
	assert.Nil(t, err)
	has := s.Has(key, id)
	fmt.Println(has)
	assert.False(t, has)

}

func TestRead(t *testing.T) {
	s := NewCAS(CASOpts{
		RootPath:      "root_path",
		TransformFunc: TransformFunc,
	})
	assert.NotNil(t, s)
	id := generateID()

	key := "testkey"
	data := []byte("test data")
	s.writeStream(key, id, bytes.NewReader(data))

	_, r, err := s.Read(key, id)
	assert.Nil(t, err)
	buf := make([]byte, 1024)
	r.Read(buf)
	fmt.Println("Data:", string(buf))
	assert.Nil(t, s.Delete(key, id))
	assert.Nil(t, s.Clear())
}

func TestFindAll(t *testing.T) {
	s := NewCAS(CASOpts{
		RootPath:      "root_path",
		TransformFunc: TransformFunc,
	})
	assert.NotNil(t, s)
	id := generateID()

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_%d", i)
		data := []byte("test data")
		s.Write(key, id, bytes.NewReader(data))
	}
	// s.writeStream(key, id, bytes.NewReader(data))
	sizes, readers, err := s.FindAll(id)
	assert.Nil(t, err)
	fmt.Println(sizes)
	fmt.Println(readers)
}
