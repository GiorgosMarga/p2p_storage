package p2plib

import "io"

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type NOPDecoder struct{}

func (d NOPDecoder) Decode(r io.Reader, msg *RPC) error {

	peekBuff := make([]byte, 1)
	r.Read(peekBuff)
	isStream := peekBuff[0] == IncomingStream
	if isStream {
		msg.Stream = true
		return nil
	}
	buf := make([]byte, 1024)

	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]
	return nil
}
