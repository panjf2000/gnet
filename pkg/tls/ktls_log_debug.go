//go:build debug
package tls

import (
	"log"
)

const Dev = true

func Debugln(a ...interface{}) {
	log.Println(a...)
}

func Debugf(format string, a ...interface{}) {
	log.Printf(format, a...)
}