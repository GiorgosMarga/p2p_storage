package p2plib

import (
	"crypto/aes"
	"crypto/rand"
	"fmt"
)

const (
	SecretKeyLen = 32
)

type Key struct {
	SecretKey []byte
}

func GenerateNewKey() Key {
	b := make([]byte, SecretKeyLen)
	rand.Read(b)
	return Key{
		SecretKey: b,
	}
}

func GenerateNewKeyFromValue(val []byte) Key {
	fmt.Println(len(val))
	if len(val) != SecretKeyLen {
		panic("Bad key length")
	}
	return Key{
		SecretKey: val,
	}
}

func (k *Key) EncryptBytes(b []byte) ([]byte, error) {
	block, err := aes.NewCipher(k.SecretKey)
	if err != nil {
		return nil, err
	}
	ciphertext := make([]byte, len(b))
	block.Encrypt(ciphertext, b)
	return ciphertext, nil
}
