package p2plib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKey(t *testing.T) {
	// key := GenerateNewKeyFromValue("f70af9fadf6b76484ece6f1423b2c8b6b585a58ab28b0bfd888e68da3a4130e1")
	key := GenerateNewKey()
	fmt.Println(key.SecretKey)
	assert.Equal(t, 32, len(key.SecretKey))
	data := "123456789123456789"
	b, err := key.EncryptBytes([]byte(data))
	assert.Nil(t, err)
	fmt.Println(b)
}
