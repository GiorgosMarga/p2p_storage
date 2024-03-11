package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKey(t *testing.T) {
	key := generateKey()
	data := []byte("test data")
	dst := new(bytes.Buffer)
	decrypted := new(bytes.Buffer)
	size, err := writeEncryptedData(key, bytes.NewReader(data), dst)

	assert.Nil(t, err)
	assert.Equal(t, int64(len(data)+IV_SIZE), size)
	size, err = readEncryptedData(key, dst, decrypted)
	assert.Equal(t, int64(len(data)+IV_SIZE), size)
	assert.Nil(t, err)
	assert.Equal(t, data, decrypted.Bytes())
}
