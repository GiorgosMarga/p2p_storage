package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/GiorgosMarga/p2p_storage/p2plib"
	"github.com/stretchr/testify/assert"
)

func createServer(listenAddr string, bootstrapNodes ...string) *FileServer {
	tcpOpt := p2plib.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2plib.NOPHandshakeFunc,
		Decoder:       p2plib.NOPDecoder{},
	}
	tcpTransport := p2plib.NewTCPTransport(tcpOpt)
	fileServerOpts := FileServerOpts{
		Transport:      tcpTransport,
		BootstrapNodes: bootstrapNodes,
	}

	s := NewFileServer(listenAddr, fileServerOpts)

	tcpTransport.OnPeer = s.OnPeer
	return s
}

func TestStore(t *testing.T) {
	var (
		s1   = createServer(":3000")
		s2   = createServer(":4000", ":3000")
		key  = "test_key"
		data = []byte("test data")
	)

	err := s1.Start()
	assert.Nil(t, err)

	err = s2.Start()
	assert.Nil(t, err)

	err = s2.Store(key, bytes.NewReader(data))
	assert.Nil(t, err)

	time.Sleep(1 * time.Second)
}
