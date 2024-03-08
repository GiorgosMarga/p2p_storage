package main

import (
	"fmt"
	"io"
	"sync"

	"github.com/GiorgosMarga/p2p_storage/p2plib"
)

type FileServerOpts struct {
	Storage     Storage
	Transport   p2plib.Transport
	StorageRoot string
}

type FileServer struct {
	FileServerOpts
	quitchan chan struct{}

	peers     map[string]p2plib.Peer
	peersLock sync.RWMutex
}

func NewFileServer(listenAddr string, opts FileServerOpts) *FileServer {
	return &FileServer{
		FileServerOpts: opts,
		quitchan:       make(chan struct{}),
	}
}

func (f *FileServer) Start() error {
	if err := f.Transport.ListenAndAccept(); err != nil {
		return err
	}

	go f.HandleMessages()
	return nil
}

func (f *FileServer) HandleMessages() error {
	defer func() {
		f.Transport.Close()
	}()
	for {
		select {
		case msg := <-f.Transport.Consume():
			fmt.Printf("msg: %+v\n", msg)
		case <-f.quitchan:
			return nil
		}
	}
}

func (f *FileServer) Write(key string, r io.Reader) error {
	if err := f.Storage.Write(key, r); err != nil {
		return err
	}
	return nil
}
