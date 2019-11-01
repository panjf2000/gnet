package codec

import (
	"github.com/google/gopacket"
)

type ICodec interface {
	Encode(in, out []byte, packet gopacket.Packet) error
	Decode(buf []byte, packet gopacket.Packet) ([]byte, error)
}
