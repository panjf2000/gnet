// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var res string

func main() {
	var port int
	var aaaa bool
	flag.IntVar(&port, "port", 8080, "server port")
	flag.BoolVar(&aaaa, "aaaa", false, "aaaaa....")
	flag.Parse()
	if aaaa {
		res = strings.Repeat("a", 1024)
	} else {
		res = "Hello World!\r\n"
	}
	log.Printf("http server started on port %d", port)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(res))
	})
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
