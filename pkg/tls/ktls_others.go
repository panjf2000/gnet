//go:build !linux
// +build !linux

package tls

import (
	"net"
)

const kTLSOverhead = 0

func (c *Conn) enableKernelTLS(cipherSuiteID uint16, inKey, outKey, inIV, outIV []byte) error {
	return nil
}

func ktlsSendCtrlMessage(fd int, typ recordType, b []byte) (int, error) {
	panic("not implement")
}

func ktlsReadDataFromRecord(fd int, b []byte) (int, error) {
	panic("not implement")
}

func ktlsReadRecord(fd int, b []byte) (recordType, int, error) {
	panic("not implement")
}
