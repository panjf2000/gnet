#!/bin/bash

set -e

echo ""
echo "--- BENCH ECHO START ---"
echo ""

cd $(dirname "${BASH_SOURCE[0]}")
function cleanup {
    echo "--- BENCH ECHO DONE ---"
    kill -9 $(jobs -rp)
    wait $(jobs -rp) 2>/dev/null
}
trap cleanup EXIT

mkdir -p bin
$(pkill -9 net-echo-server || printf "")
$(pkill -9 evio-echo-server || printf "")

function gobench {
    echo "--- $1 ---"
    if [ "$3" != "" ]; then
        go build -o $2 $3
    fi
    GOMAXPROCS=1 $2 --port $4 &
    sleep 1
    echo "*** 50 connections, 10 seconds, 6 byte packets"
    nl=$'\r\n'
    tcpkali --workers 1 -c 50 -T 10s -m "PING{$nl}" 127.0.0.1:$4
    echo "--- DONE ---"
    echo ""
}

gobench "GO STDLIB" bin/net-echo-server net-echo-server/main.go 5001
gobench "EVIO" bin/evio-echo-server ../examples/echo-server/main.go 5002
