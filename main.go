package main

// TODO:
//		 - add syncing
//       - refactor store pathing
// 		 - add peer discovery
//       - make more tests

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/GiorgosMarga/p2p_storage/p2plib"
)

func makeServer(addr string, bootstrap ...string) *FileServer {
	tcpOpts := p2plib.TCPTransportOpts{
		Decoder:       p2plib.NOPDecoder{},
		ListenAddr:    addr,
		HandshakeFunc: p2plib.NOPHandshakeFunc,
	}
	tcpTransport := p2plib.NewTCPTransport(tcpOpts)
	fileServerOpts := FileServerOpts{
		Transport:      tcpTransport,
		BootstrapNodes: bootstrap,
	}
	s := NewFileServer(tcpOpts.ListenAddr, fileServerOpts)
	tcpTransport.OnPeer = s.OnPeer
	return s
}

func main() {
	var (
		s1 = makeServer(":3000")
		s2 = makeServer(":4000", ":3000")
		s3 = makeServer(":5000", ":3000", ":4000")
	)

	if err := s1.Start(); err != nil {
		log.Fatal(err)
	}
	if err := s2.Start(); err != nil {
		log.Fatal(err)
	}
	if err := s3.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	for i := 0; i < 10; i++ {
		data := []byte("test data")
		key := fmt.Sprintf("key_%d", i)
		if err := s2.Store(key, bytes.NewReader(data)); err != nil {
			log.Fatal(err)
		}
		if err := s2.storage.Delete(key, s2.ID); err != nil {
			log.Fatal(err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := s2.SyncStorage(); err != nil {
		log.Fatal(err)
	}

	time.Sleep(1 * time.Second)
}
