package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Storage interface {
	Write(key string, r io.Reader) (int64, error)
	Read(key string) (io.Reader, error)
	Has(key string) bool
	Delete(key string) error
	Clear() error
}

type Path struct {
	Path     string
	Filename string
}

func (f *Path) GetFullPath() string {
	return fmt.Sprintf("%s/%s", f.Path, f.Filename)
}
func (f *Path) WithRoot(root string) string {
	return fmt.Sprintf("%s/%s", root, f.Path)
}
func (f *Path) ParentFolder() string {
	blocks := strings.Split(f.Path, "/")
	if len(blocks) == 0 {
		return ""
	}
	return blocks[0]
}
func TransformFunc(key string) Path {
	hash := sha1.Sum([]byte(key))
	hashEnc := hex.EncodeToString(hash[:])
	blockSize := 5
	path := ""
	for i := 0; i < len(hashEnc); i++ {
		if i != 0 && i%blockSize == 0 {
			path += "/"
		}
		path += string(hashEnc[i])
	}
	return Path{
		Path:     path,
		Filename: hashEnc,
	}
}

type CASOpts struct {
	TransformFunc func(string) Path
	RootPath      string
}

type CAS struct {
	CASOpts
}

func NewCAS(opts CASOpts) *CAS {
	return &CAS{
		CASOpts: opts,
	}
}

func (s *CAS) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}
func (s *CAS) writeStream(key string, r io.Reader) (int64, error) {
	pathname := s.TransformFunc(key)
	// joins the root with the generated hashed path root/*/*/*...
	withRoot := pathname.WithRoot(s.RootPath)
	if err := os.MkdirAll(withRoot, os.ModePerm); err != nil {
		fmt.Println(err)
		return 0, err
	}
	fullPathName := fmt.Sprintf("%s/%s", s.RootPath, pathname.GetFullPath())
	f, err := os.Create(fullPathName)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}
	fmt.Printf("I wrote %d bytes in disk at %s\n", n, fullPathName)
	return n, nil
}

func (s *CAS) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	return buf, err
}

func (s *CAS) readStream(key string) (io.ReadCloser, error) {
	path := s.TransformFunc(key)
	fullpath := fmt.Sprintf("%s/%s", path.WithRoot(s.RootPath), path.Filename)
	return os.Open(fullpath)
}

func (s *CAS) Has(key string) bool {
	path := s.TransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", path.WithRoot(s.RootPath), path.Filename)
	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(os.ErrNotExist, err)
}

func (s *CAS) Delete(key string) error {
	path := s.TransformFunc(key)
	pathToDelete := fmt.Sprintf("%s/%s", s.RootPath, path.ParentFolder())
	return os.RemoveAll(pathToDelete)
}
func (s *CAS) Clear() error {
	return os.RemoveAll(s.RootPath)
}
