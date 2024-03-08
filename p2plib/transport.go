package p2plib

type Peer interface {
	Send([]byte) error
}

type Transport interface {
	Dial(string) error
	ListenAndAccept() error
	Close() error
	Consume() <-chan RPC
}
