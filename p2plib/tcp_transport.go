package p2plib

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type TCPPeer struct {
	net.Conn
	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn) *TCPPeer {
	return &TCPPeer{
		Conn: conn,
		wg:   &sync.WaitGroup{},
	}
}
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}
func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Write(b)
	if err != nil {
		return err
	}
	return nil
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}
type TCPTransport struct {
	TCPTransportOpts
	Ln      net.Listener
	rpcchan chan RPC
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcchan:          make(chan RPC, 10),
	}
}

func (tr *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", tr.ListenAddr)
	if err != nil {
		return err
	}

	tr.Ln = ln
	fmt.Printf("[%s] TCP server started...\n", tr.ListenAddr)
	go tr.acceptLoop()
	return nil
}

func (tr *TCPTransport) acceptLoop() {
	for {
		conn, err := tr.Ln.Accept()
		if err != nil {
			continue
		}

		go tr.handleConn(conn)
	}
}

func (tr *TCPTransport) handleConn(conn net.Conn) {
	p := NewTCPPeer(conn)
	if tr.HandshakeFunc != nil {
		if err := tr.HandshakeFunc(p); err != nil {
			return
		}
	}
	if tr.OnPeer != nil {
		if err := tr.OnPeer(p); err != nil {
			return
		}
	}

	for {
		rpc := RPC{}

		if err := tr.Decoder.Decode(conn, &rpc); err != nil {
			log.Println(err)
			continue
		}
		rpc.From = conn.RemoteAddr().String()
		if rpc.Stream {
			p.wg.Add(1)
			log.Printf("[%s] Incoming stream. Waiting....\n", tr.ListenAddr)
			p.wg.Wait()
			log.Printf("[%s] Stream closed. Resuming....\n", tr.ListenAddr)
			continue

		}
		tr.rpcchan <- rpc
	}
}

func (tr *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go tr.handleConn(conn)
	return nil
}

func (tr *TCPTransport) Close() error {
	return tr.Ln.Close()
}

func (tr *TCPTransport) Consume() <-chan RPC {
	return tr.rpcchan
}

func (tr *TCPTransport) Addr() string {
	return tr.ListenAddr
}
