package p2plib

const (
	IncomingMessage = iota
	IncomingStream
	HaveFile
	DontHaveFile
)

type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
