package tls

import (
	"os"
	"strings"
)

var kTLSEnabled bool

// kTLSCipher is a placeholder to tell the record layer to skip wrapping.
type kTLSCipher struct{}

func init() {
	kTLSEnabled = strings.ToLower(os.Getenv("GOKTLS")) == "true" ||
		strings.ToLower(os.Getenv("GOKTLS")) == "on" ||
		os.Getenv("GOKTLS") == "1"
}
