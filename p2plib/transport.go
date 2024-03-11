package p2plib

import "net"

type Peer interface {
	net.Conn
	Send([]byte) error
	CloseStream()
}

type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Close() error
	Consume() <-chan RPC
	Addr() string
}
