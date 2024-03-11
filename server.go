package main

import (
	"bytes"
	"encoding/binary"
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
	EncKey         []byte
}

type FileServer struct {
	FileServerOpts
	storage Storage

	quitchan chan struct{}

	peers     map[string]p2plib.Peer
	peersLock sync.RWMutex
}

func NewFileServer(listenAddr string, opts FileServerOpts) *FileServer {
	casOpts := CASOpts{
		TransformFunc: TransformFunc,
		RootPath:      listenAddr + "_storage",
	}
	if len(opts.EncKey) == 0 {
		opts.EncKey = generateKey()
	}
	return &FileServer{
		storage:        NewCAS(casOpts),
		FileServerOpts: opts,
		quitchan:       make(chan struct{}),
		peers:          make(map[string]p2plib.Peer),
	}
}

func (s *FileServer) OnPeer(p p2plib.Peer) error {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	fmt.Printf("connected with %s\n", p.RemoteAddr().String())
	return nil
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(s.BootstrapNodes) > 0 {
		for _, addr := range s.BootstrapNodes {
			if err := s.Transport.Dial(addr); err != nil {
				return err
			}
		}
	}

	go s.HandleMessages()
	return nil
}

func (s *FileServer) HandleMessages() error {
	defer func() {
		s.Transport.Close()
	}()
	for {
		select {
		case rpc := <-s.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				return err
			}
			switch v := msg.Payload.(type) {
			case MessageStoreData:
				s.handleMessageStoreData(v, rpc.From)
			case MessageReadData:
				s.handleMessageReadData(v, rpc.From)
			}
		case <-s.quitchan:
			return nil
		}
	}
}
func (s *FileServer) handleMessageReadData(msg MessageReadData, from string) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not connected")
	}
	peer.Send([]byte{p2plib.IncomingStream})

	if !s.storage.Has(msg.Key) {
		peer.Send([]byte{p2plib.DontHaveFile})
		return fmt.Errorf("[%s] dont have file (%s)", s.Transport.Addr(), msg.Key)
	}

	size, file, err := s.storage.Read(msg.Key)
	if err != nil {
		return err
	}
	peer.Send([]byte{p2plib.HaveFile})
	binary.Write(peer, binary.LittleEndian, size)
	io.Copy(peer, file)
	fmt.Printf("[%s] sent file (%s) to %s\n", s.Transport.Addr(), msg.Key, peer.RemoteAddr())
	return nil
}
func (s *FileServer) handleMessageStoreData(msg MessageStoreData, from string) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not connected")
	}
	r := io.LimitReader(peer, msg.Size)
	n, err := s.storage.Write(msg.Key, r)

	if err != nil {
		fmt.Println("Error:", err)
		return err
	}

	fmt.Printf("received message from %s and wrote %d on disk\n", from, n)
	peer.CloseStream()
	return nil
}
func (s *FileServer) Close() {
	close(s.quitchan)
}

type Message struct {
	Payload any
}
type MessageStoreData struct {
	Key  string
	Size int64
}
type MessageReadData struct {
	Key string
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {
		peer.Send([]byte{p2plib.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileServer) Store(key string, r io.Reader) error {
	var (
		buf = new(bytes.Buffer)
		tee = io.TeeReader(r, buf)
	)
	n, err := s.storage.Write(key, tee)
	if err != nil {
		return err
	}
	msg := Message{
		Payload: MessageStoreData{
			Key:  encryptFileKey(key),
			Size: n + int64(IV_SIZE),
		},
	}
	if err := s.broadcast(&msg); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	mw.Write([]byte{p2plib.IncomingStream})
	if _, err := writeEncryptedData(s.EncKey, buf, mw); err != nil {
		return err
	}

	return nil
}

func (s *FileServer) Read(key string) (io.Reader, error) {
	if s.storage.Has(key) {
		fmt.Printf("[%s] found file (%s) on disk.\n", s.Transport.Addr(), key)
		_, r, err := s.storage.Read(key)
		return r, err
	}

	msg := Message{
		Payload: MessageReadData{
			Key: encryptFileKey(key),
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return nil, err
	}
	for _, peer := range s.peers {
		peekBuf := make([]byte, 1)
		peer.Read(peekBuf)
		if peekBuf[0] == p2plib.DontHaveFile {
			fmt.Printf("[%s] peer (%s) dont have file\n", s.Transport.Addr(), peer.RemoteAddr())
			continue
		}
		var filesize int64
		binary.Read(peer, binary.LittleEndian, &filesize)
		_, err := s.storage.ReadEncryptedData(s.EncKey, key, io.LimitReader(peer, filesize))
		if err != nil {
			return nil, err
		}
		peer.CloseStream()
		if peekBuf[0] == p2plib.HaveFile {
			fmt.Printf("[%s] peer %s sent file\n", s.Transport.Addr(), peer.RemoteAddr())
			break
		}
	}
	_, r, err := s.storage.Read(key)
	return r, err
}

func init() {
	gob.Register(MessageStoreData{})
	gob.Register(MessageReadData{})
}
