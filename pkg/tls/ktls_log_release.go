//go:build !debug
package tls

const Dev = false

func Debugln(a ...interface{}) {}

func Debugf(format string, a ...interface{}) {}