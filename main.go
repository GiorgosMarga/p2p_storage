package main

// TODO:
//       - refactor store pathing
// 		 - add peer discovery
//       - make more tests
// FIXME:
//		 - double connection on peer discovery
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
		s3 = makeServer(":5000", ":4000")
		s4 = makeServer(":6000", ":5000")
		s5 = makeServer(":7000", ":6000")
	)

	if err := s1.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	if err := s2.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	if err := s3.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	if err := s4.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	if err := s5.Start(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(1 * time.Second)

	fmt.Printf("%d\n", len(s1.peers))
	fmt.Printf("%d\n", len(s2.peers))
	fmt.Printf("%d\n", len(s3.peers))
	fmt.Printf("%d\n", len(s4.peers))
	fmt.Printf("%d\n", len(s5.peers))

	for i := 0; i < 5; i++ {
		data := []byte("test data")
		key := fmt.Sprintf("key_%d", i)
		if err := s2.Store(key, bytes.NewReader(data)); err != nil {
			log.Fatal(err)
		}
		// if err := s2.storage.Delete(key, s2.ID); err != nil {
		// 	log.Fatal(err)
		// }
		time.Sleep(500 * time.Millisecond)
	}

	// if err := s2.SyncStorage(); err != nil {
	// 	log.Fatal(err)
	// }

	time.Sleep(1 * time.Second)

}
