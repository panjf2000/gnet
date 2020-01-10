package tls

import (
	"github.com/panjf2000/gnet"
)

type TlsICodec struct {
	Config *Config
}
type Ctx interface {
	GetConn() *Conn
	SetConn(*Conn)
}

func (code *TlsICodec) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	if ctx, ok := c.Context().(Ctx); ok {
		if conn := ctx.GetConn(); conn != nil {
			if conn.handshakeComplete() {
				return conn.Encode(buf)
			}
		}
	}
	return buf, nil
}

func (code *TlsICodec) Decode(c gnet.Conn) ([]byte, error) {
	if c.BufferLength() > 0 {
		if ctx, ok := c.Context().(Ctx); ok {
			conn := ctx.GetConn()
			if conn == nil {
				conn = Server(c, code.Config.Clone())
				ctx.SetConn(conn)
			}
			return conn.Decode()
		}
	}

	return nil, nil

}
func NewConfig(certFile, keyFile string) (c *Config, err error) {
	c = &Config{NextProtos: []string{"http/1.1"}}
	c.Certificates = make([]Certificate, 1)
	c.Certificates[0], err = LoadX509KeyPair(certFile, keyFile)
	return
}
