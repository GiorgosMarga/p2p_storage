package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Storage interface {
	Write(string, string, io.Reader) (int64, error)
	WriteEncryptedData([]byte, string, io.Reader, io.Writer) (int64, error)
	Read(string, string) (int64, io.Reader, error)
	ReadEncryptedData([]byte, string, string, io.Reader) (int64, error)
	FindAll(string) ([]int64, []io.Reader, error)
	Has(string, string) bool
	Delete(string, string) error
	Clear() error
}

type Path struct {
	Path     string
	Filename string
}

func (f *Path) GetFullPath() string {
	return fmt.Sprintf("%s/%s", f.Path, f.Filename)
}
func (f *Path) WithRoot(root, id string) string {
	return fmt.Sprintf("%s/%s/%s", root, id, f.Path)
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
	path := generatePath(hashEnc, blockSize)
	return Path{
		Path:     path,
		Filename: hashEnc,
	}
}
func generatePath(hashEnc string, blockSize int) string {
	path := ""
	for i := 0; i < len(hashEnc); i++ {
		if i != 0 && i%blockSize == 0 {
			path += "/"
		}
		path += string(hashEnc[i])
	}
	return path
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

func (s *CAS) Write(key, ID string, r io.Reader) (int64, error) {
	return s.writeStream(key, ID, r)
}

func (s *CAS) WriteEncryptedData(encKey []byte, key string, r io.Reader, dst io.Writer) (int64, error) {
	return writeEncryptedData(encKey, r, dst)
}
func (s *CAS) writeStream(key, ID string, r io.Reader) (int64, error) {
	pathname := s.TransformFunc(key)
	// joins the root with the generated hashed path root/*/*/*...
	withRoot := pathname.WithRoot(s.RootPath, ID)
	if err := os.MkdirAll(withRoot, os.ModePerm); err != nil {
		fmt.Println(err)
		return 0, err
	}
	fullPathName := fmt.Sprintf("%s/%s/%s", s.RootPath, ID, pathname.GetFullPath())
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

func (s *CAS) Read(key, ID string) (int64, io.Reader, error) {
	f, err := s.readStream(key, ID)
	if err != nil {
		return 0, nil, err
	}
	stat, err := f.Stat()
	return stat.Size(), f, err
}
func (s *CAS) ReadEncryptedData(encKey []byte, key, ID string, r io.Reader) (int64, error) {
	f, err := s.openFileToWrite(key, ID)
	if err != nil {
		return 0, err
	}
	return readEncryptedData(encKey, r, f)
}

func (s *CAS) openFileToWrite(key, ID string) (*os.File, error) {
	pathname := s.TransformFunc(key)
	// joins the root with the generated hashed path root/*/*/*...
	withRoot := pathname.WithRoot(s.RootPath, ID)
	if err := os.MkdirAll(withRoot, os.ModePerm); err != nil {
		fmt.Println(err)
		return nil, err
	}
	fullPathName := fmt.Sprintf("%s/%s/%s", s.RootPath, ID, pathname.GetFullPath())

	return os.Create(fullPathName)
}
func (s *CAS) readStream(key, ID string) (*os.File, error) {
	path := s.TransformFunc(key)
	fullpath := fmt.Sprintf("%s/%s", path.WithRoot(s.RootPath, ID), path.Filename)
	return os.Open(fullpath)
}
func (s *CAS) Has(key, ID string) bool {
	path := s.TransformFunc(key)
	fullPathWithRoot := fmt.Sprintf("%s/%s", path.WithRoot(s.RootPath, ID), path.Filename)
	_, err := os.Stat(fullPathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *CAS) Delete(key, ID string) error {
	path := s.TransformFunc(key)
	pathToDelete := fmt.Sprintf("%s/%s/%s", s.RootPath, ID, path.ParentFolder())
	err := os.RemoveAll(pathToDelete)
	if err != nil {
		return err
	}
	fmt.Printf("File (%s) was deleted\n", key)
	return nil
}
func (s *CAS) Clear() error {
	return os.RemoveAll(s.RootPath)
}

func (s *CAS) FindAll(ID string) ([]int64, []io.Reader, error) {
	files := []io.Reader{}
	sizes := make([]int64, 0)
	fileNames := make([]string, 0)
	path := fmt.Sprintf("%s/%s/", s.RootPath, ID)
	filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			fileNames = append(fileNames, info.Name())
			return filepath.SkipDir
		}
		return nil
	})

	for _, filename := range fileNames {
		n, r, err := s.findFile(filename, ID)
		if err != nil {
			fmt.Println(err)
			return nil, nil, err
		}
		sizes = append(sizes, n)
		files = append(files, r)
	}
	return sizes, files, nil
}

func (s *CAS) findFile(hashedKey string, ID string) (int64, io.Reader, error) {
	path := generatePath(hashedKey, 5)
	fullPath := fmt.Sprintf("%s/%s/%s/%s", s.RootPath, ID, path, hashedKey)
	f, err := os.Open(fullPath)
	if err != nil {
		return 0, nil, err
	}
	stat, err := f.Stat()
	return stat.Size(), f, err
}
