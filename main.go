package main

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
	casOpts := CASOpts{
		TransformFunc: TransformFunc,
		RootPath:      addr + "_storage",
	}
	tcpTransport := p2plib.NewTCPTransport(tcpOpts)
	fileServerOpts := FileServerOpts{
		Storage:        NewCAS(casOpts),
		Transport:      tcpTransport,
		StorageRoot:    casOpts.RootPath,
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
	)

	if err := s1.Start(); err != nil {
		log.Fatal(err)
	}
	if err := s2.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	data := []byte("test data")

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("test_key_%d", i)
		if err := s2.Write(key, bytes.NewReader(data)); err != nil {
			log.Fatal(err)
		}
		time.Sleep(time.Millisecond * 100)
	}

	select {}
}
