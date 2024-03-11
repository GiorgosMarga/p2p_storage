package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

const IV_SIZE int = 16

func generateKey() []byte {
	b := make([]byte, 32)
	io.ReadFull(rand.Reader, b)
	return b
}

func encryptFileKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func writeEncryptedData(key []byte, src io.Reader, dst io.Writer) (int64, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	iv := make([]byte, IV_SIZE)
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return 0, err
	}
	_, err = dst.Write(iv)
	if err != nil {
		return 0, err
	}
	ctr := cipher.NewCTR(block, iv)
	buf := make([]byte, 32*1024)
	totalBytes := IV_SIZE
	for {
		n, err := src.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		totalBytes += n
		outBuf := make([]byte, n)
		ctr.XORKeyStream(outBuf, buf[:n])
		dst.Write(outBuf[:n])
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return int64(totalBytes), nil
}
func readEncryptedData(key []byte, src io.Reader, dst io.Writer) (int64, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}
	iv := make([]byte, IV_SIZE)
	src.Read(iv)
	if err != nil {
		return 0, err
	}
	ctr := cipher.NewCTR(block, iv)
	buf := make([]byte, 32*1024)
	totalBytes := IV_SIZE
	for {
		n, err := src.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		totalBytes += n
		outBuf := make([]byte, n)
		ctr.XORKeyStream(outBuf, buf[:n])
		dst.Write(outBuf[:n])
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return int64(totalBytes), nil
}
