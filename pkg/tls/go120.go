//go:build go1.20

package tls

import (
	"net"
	_ "unsafe"

	gtls "github.com/0-haha/gnet_go_tls/v120"
)

//nolint:revive
const (
	// TLS 1.0 - 1.2 cipher suites.
	TLS_RSA_WITH_RC4_128_SHA                      uint16 = gtls.TLS_RSA_WITH_RC4_128_SHA
	TLS_RSA_WITH_3DES_EDE_CBC_SHA                 uint16 = gtls.TLS_RSA_WITH_3DES_EDE_CBC_SHA
	TLS_RSA_WITH_AES_128_CBC_SHA                  uint16 = gtls.TLS_RSA_WITH_AES_128_CBC_SHA
	TLS_RSA_WITH_AES_256_CBC_SHA                  uint16 = gtls.TLS_RSA_WITH_AES_256_CBC_SHA
	TLS_RSA_WITH_AES_128_CBC_SHA256               uint16 = gtls.TLS_RSA_WITH_AES_128_CBC_SHA256
	TLS_RSA_WITH_AES_128_GCM_SHA256               uint16 = gtls.TLS_RSA_WITH_AES_128_GCM_SHA256
	TLS_RSA_WITH_AES_256_GCM_SHA384               uint16 = gtls.TLS_RSA_WITH_AES_256_GCM_SHA384
	TLS_ECDHE_ECDSA_WITH_RC4_128_SHA              uint16 = gtls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA          uint16 = gtls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
	TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA          uint16 = gtls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
	TLS_ECDHE_RSA_WITH_RC4_128_SHA                uint16 = gtls.TLS_ECDHE_RSA_WITH_RC4_128_SHA
	TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA           uint16 = gtls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA
	TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA            uint16 = gtls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
	TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA            uint16 = gtls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
	TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256       uint16 = gtls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256
	TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256         uint16 = gtls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256
	TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256         uint16 = gtls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256       uint16 = gtls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384         uint16 = gtls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384       uint16 = gtls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256   uint16 = gtls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256 uint16 = gtls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256

	// TLS 1.3 cipher suites.
	TLS_AES_128_GCM_SHA256       uint16 = gtls.TLS_AES_128_GCM_SHA256
	TLS_AES_256_GCM_SHA384       uint16 = gtls.TLS_AES_256_GCM_SHA384
	TLS_CHACHA20_POLY1305_SHA256 uint16 = gtls.TLS_CHACHA20_POLY1305_SHA256

	// TLS_FALLBACK_SCSV isn't a standard cipher suite but an indicator
	// that the client is doing version fallback. See RFC 7507.
	TLS_FALLBACK_SCSV uint16 = gtls.TLS_FALLBACK_SCSV

	// Legacy names for the corresponding cipher suites with the correct _SHA256
	// suffix, retained for backward compatibility.
	TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305   = gtls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
	TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305 = gtls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
)

type (
	// A Certificate is gnet_go_tls101.Certificate.
	Certificate = gtls.Certificate
	// CertificateRequestInfo contains information about a certificate request.
	CertificateRequestInfo = gtls.CertificateRequestInfo
	// ClientHelloInfo contains information about a ClientHello.
	ClientHelloInfo = gtls.ClientHelloInfo
	// ClientSessionCache is a cache used for session resumption.
	ClientSessionCache = gtls.ClientSessionCache
	// ClientSessionState is a state needed for session resumption.
	ClientSessionState = gtls.ClientSessionState
	// A Config is a gnet_go_tls101.Config.
	Config = gtls.Config
	// A Conn is a gnet_go_tls101.Conn.
	Conn = gtls.Conn
)

const (
	VersionTLS10 = gtls.VersionTLS10
	VersionTLS11 = gtls.VersionTLS11
	VersionTLS12 = gtls.VersionTLS12
	VersionTLS13 = gtls.VersionTLS13

	// Deprecated: SSLv3 is cryptographically broken, and is no longer
	// supported by this package. See golang.org/issue/32716.
	VersionSSL30 = gtls.CurveP256
)

//go:linkname Server github.com/0-haha/gnet_go_tls/v120.Server
func Server(conn net.Conn, config *Config) *Conn

//go:linkname LoadX509KeyPair github.com/0-haha/gnet_go_tls/v120.LoadX509KeyPair
func LoadX509KeyPair(certFile, keyFile string) (Certificate, error)

//go:linkname X509KeyPair github.com/0-haha/gnet_go_tls/v120.X509KeyPair
func X509KeyPair(certPEMBlock, keyPEMBlock []byte) (Certificate, error)
