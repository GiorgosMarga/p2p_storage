package main

import (
	"fmt"
	"time"

	"github.com/GiorgosMarga/p2p_storage/p2plib"
)

func makeServer(addr string) *p2plib.TCPTransport {
	opts := p2plib.TCPTransportOpts{
		Decoder:       p2plib.NOPDecoder{},
		ListenAddr:    addr,
		HandshakeFunc: p2plib.NOPHandshakeFunc,
	}
	return p2plib.NewTCPTransport(opts)
}

func main() {
	var (
		s1 = makeServer(":3000")
		s2 = makeServer(":4000")
	)
	if err := s1.ListenAndAccept(); err != nil {
		fmt.Println(err)
	}
	time.Sleep(1 * time.Second)
	if err := s2.ListenAndAccept(); err != nil {
		fmt.Println(err)
	}
	time.Sleep(1 * time.Second)

	s2.Dial(":3000")
	select {}
}
