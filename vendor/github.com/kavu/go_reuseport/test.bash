#!/bin/bash

set -e

# Thanks to IPFS team
if [[ "$TRAVIS_OS_NAME" == "linux" ]]; then
    if [[ "$TRAVIS_SUDO" == true ]]; then
        # Ensure that IPv6 is enabled.
        # While this is unsupported by TravisCI, it still works for localhost.
        sudo sysctl -w net.ipv6.conf.lo.disable_ipv6=0
        sudo sysctl -w net.ipv6.conf.default.disable_ipv6=0
        sudo sysctl -w net.ipv6.conf.all.disable_ipv6=0
    fi
else
    # OSX has a default file limit of 256, for some tests we need a
    # maximum of 8192.
    ulimit -Sn 8192
fi

go test -v -cover ./...
go test -v -cover -race ./... -coverprofile=coverage.txt -covermode=atomic
go test -v -cover -race -benchmem -benchtime=5s -bench=.