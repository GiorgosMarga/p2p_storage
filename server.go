package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/GiorgosMarga/p2p_storage/p2plib"
)

type FileServerOpts struct {
	Transport      p2plib.Transport
	OnPeer         func(p2plib.Peer) error
	BootstrapNodes []string
}

type FileServer struct {
	FileServerOpts
	Storage Storage

	quitchan chan struct{}

	peers     map[string]p2plib.Peer
	peersLock sync.RWMutex
}

func NewFileServer(listenAddr string, opts FileServerOpts) *FileServer {
	casOpts := CASOpts{
		TransformFunc: TransformFunc,
		RootPath:      listenAddr + "_storage",
	}

	return &FileServer{
		Storage:        NewCAS(casOpts),
		FileServerOpts: opts,
		quitchan:       make(chan struct{}),
		peers:          make(map[string]p2plib.Peer),
	}
}

func (f *FileServer) OnPeer(p p2plib.Peer) error {
	f.peersLock.Lock()
	defer f.peersLock.Unlock()

	f.peers[p.RemoteAddr().String()] = p
	fmt.Printf("connected with %s\n", p.RemoteAddr().String())
	return nil
}

func (f *FileServer) Start() error {
	if err := f.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(f.BootstrapNodes) > 0 {
		for _, addr := range f.BootstrapNodes {
			if err := f.Transport.Dial(addr); err != nil {
				return err
			}
		}
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
		case rpc := <-f.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				return err
			}
			switch v := msg.Payload.(type) {
			case MessageStoreData:
				f.handleMessageStoreData(v, rpc.From)
			}
		case <-f.quitchan:
			return nil
		}
	}
}

func (f *FileServer) handleMessageStoreData(msg MessageStoreData, from string) error {
	peer, ok := f.peers[from]
	if !ok {
		return fmt.Errorf("peer not connected")
	}
	r := io.LimitReader(peer, msg.Size)
	n, err := f.Storage.Write(msg.Key, r)

	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	fmt.Printf("received message from %s and wrote %d on disk\n", from, n)
	peer.CloseStream()
	return nil
}
func (f *FileServer) Close() {
	close(f.quitchan)
}

type Message struct {
	Payload any
}
type MessageStoreData struct {
	Key  string
	Size int64
}

func (f *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range f.peers {
		peer.Send([]byte{p2plib.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileServer) Write(key string, r io.Reader) error {
	var (
		buf = new(bytes.Buffer)
		tee = io.TeeReader(r, buf)
	)
	n, err := f.Storage.Write(key, tee)
	if err != nil {
		return err
	}
	msg := Message{
		Payload: MessageStoreData{
			Key:  key,
			Size: n,
		},
	}
	if err := f.broadcast(&msg); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	for _, peer := range f.peers {
		peer.Send([]byte{p2plib.IncomingStream})
		if _, err := io.Copy(peer, buf); err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}

func init() {
	gob.Register(MessageStoreData{})
}
