package p2plib

import "io"

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type NOPDecoder struct{}

func (d NOPDecoder) Decode(r io.Reader, msg *RPC) error {
	buf := make([]byte, 1024)

	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]
	return nil
}
