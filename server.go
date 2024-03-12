package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/GiorgosMarga/p2p_storage/p2plib"
)

type FileServerOpts struct {
	Transport      p2plib.Transport
	OnPeer         func(p2plib.Peer) error
	BootstrapNodes []string
	EncKey         []byte
	ID             string
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
	if len(opts.ID) == 0 {
		opts.ID = generateID()
	}
	return &FileServer{
		storage:        NewCAS(casOpts),
		FileServerOpts: opts,
		quitchan:       make(chan struct{}),
		peers:          make(map[string]p2plib.Peer),
	}
}

func (s *FileServer) OnPeer(p p2plib.Peer, outbound bool) error {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	if !outbound {
		// fmt.Printf("[%s] !outbould %s\n", s.Transport.Addr(), p.(*p2plib.TCPPeer).ListenAddr)
		peers := make([]string, 0)
		for _, p := range s.peers {
			peers = append(peers, p.(*p2plib.TCPPeer).ListenAddr)
		}
		msg := Message{
			Payload: MessageDiscovery{
				Peers: peers,
			},
		}
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			log.Fatal(err)
		}
		if err := p.Send([]byte{p2plib.IncomingMessage}); err != nil {
			log.Fatal(err)
		}
		if err := p.Send(buf.Bytes()); err != nil {
			log.Fatal(err)
		}

	}

	s.peers[p.RemoteAddr().String()] = p

	fmt.Printf("[%s] has connected with %s\n", s.Transport.Addr(), p.RemoteAddr())

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
			case MessageDeleteData:
				s.handleMessageDeleteData(v, rpc.From)
			case MessageSyncData:
				s.handleMessageSyncData(v, rpc.From)
			case MessageDiscovery:
				s.handleMessageDiscovery(v, rpc.From)
			}
		case <-s.quitchan:
			return nil
		}
	}
}
func (s *FileServer) handleMessageDiscovery(msg MessageDiscovery, from string) error {
	for _, addr := range msg.Peers {
		if _, ok := s.peers[addr]; ok {
			continue
		}
		fmt.Printf("[%s] discovered %s\n", s.Transport.Addr(), addr)
		if err := s.Transport.Dial(addr); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

// TODO: check if i can retrieve key of file (maybe encrypt instead of hashing)
// FIXME: fix duplicate files

func (s *FileServer) handleMessageSyncData(msg MessageSyncData, from string) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("[%s] peer (%s) not found", s.Transport.Addr(), peer.RemoteAddr())
	}

	sizes, files, err := s.storage.FindAll(msg.ID)
	if err != nil {
		return err
	}
	peer.Send([]byte{p2plib.IncomingStream})
	binary.Write(peer, binary.LittleEndian, int64(len(files)))
	for i := 0; i < len(files); i++ {
		binary.Write(peer, binary.LittleEndian, sizes[i])
		if _, err := io.Copy(peer, files[i]); err != nil {
			return err
		}
	}
	return nil
}
func (s *FileServer) handleMessageDeleteData(msg MessageDeleteData, from string) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("[%s] peer (%s) not found", s.Transport.Addr(), peer.RemoteAddr())
	}
	return s.storage.Delete(msg.Key, msg.ID)
}
func (s *FileServer) handleMessageReadData(msg MessageReadData, from string) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("[%s] peer (%s) not found", s.Transport.Addr(), peer.RemoteAddr())
	}
	peer.Send([]byte{p2plib.IncomingStream})

	if !s.storage.Has(msg.Key, msg.ID) {
		peer.Send([]byte{p2plib.DontHaveFile})
		return fmt.Errorf("[%s] dont have file (%s)", s.Transport.Addr(), msg.Key)
	}

	size, file, err := s.storage.Read(msg.Key, msg.ID)
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
		return fmt.Errorf("[%s] peer (%s) not found", s.Transport.Addr(), peer.RemoteAddr())
	}
	r := io.LimitReader(peer, msg.Size)
	n, err := s.storage.Write(msg.Key, msg.ID, r)

	if err != nil {
		return err
	}

	fmt.Printf("[%s] received message from %s and wrote %d on disk\n", s.Transport.Addr(), from, n)
	peer.CloseStream()
	return nil
}
func (s *FileServer) Close() {
	close(s.quitchan)
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
	n, err := s.storage.Write(key, s.ID, tee)
	if err != nil {
		return err
	}
	msg := Message{
		Payload: MessageStoreData{
			Key:  encryptFileKey(key),
			Size: n + int64(IV_SIZE),
			ID:   s.ID,
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
	if s.storage.Has(key, s.ID) {
		fmt.Printf("[%s] found file (%s) on disk.\n", s.Transport.Addr(), key)
		_, r, err := s.storage.Read(key, s.ID)
		return r, err
	}

	msg := Message{
		Payload: MessageReadData{
			Key: encryptFileKey(key),
			ID:  s.ID,
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
		_, err := s.storage.ReadEncryptedData(s.EncKey, key, s.ID, io.LimitReader(peer, filesize))
		if err != nil {
			return nil, err
		}
		peer.CloseStream()
		if peekBuf[0] == p2plib.HaveFile {
			fmt.Printf("[%s] peer %s sent file\n", s.Transport.Addr(), peer.RemoteAddr())
			break
		}
	}
	_, r, err := s.storage.Read(key, s.ID)
	return r, err
}

func (s *FileServer) Delete(key string) error {
	if err := s.storage.Delete(key, s.ID); err != nil {
		return err
	}

	msg := Message{
		Payload: MessageDeleteData{
			Key: encryptFileKey(key),
			ID:  s.ID,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}
	return nil
}

func (s *FileServer) SyncStorage() error {
	fmt.Printf("[%s] trying to sync...\n", s.Transport.Addr())
	msg := Message{
		Payload: MessageSyncData{
			ID: s.ID,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		return err
	}
	var numOfFiles int64
	for _, peer := range s.peers {
		binary.Read(peer, binary.LittleEndian, &numOfFiles)
		for i := 0; i < int(numOfFiles); i++ {
			var fileSize int64
			binary.Read(peer, binary.LittleEndian, &fileSize)
			s.storage.ReadEncryptedData(s.EncKey, string(generateKey()), s.ID, io.LimitReader(peer, fileSize))
		}
		peer.CloseStream()
	}
	fmt.Printf("[%s] received %d files\n", s.Transport.Addr(), numOfFiles)
	return nil
}

func init() {
	gob.Register(MessageStoreData{})
	gob.Register(MessageReadData{})
	gob.Register(MessageDeleteData{})
	gob.Register(MessageDiscovery{})
	gob.Register(MessageSyncData{})
}
