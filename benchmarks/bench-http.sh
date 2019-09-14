#!/bin/bash

set -e

echo ""
echo "--- BENCH HTTP START ---"
echo ""

cd $(dirname "${BASH_SOURCE[0]}")
function cleanup {
    echo "--- BENCH HTTP DONE ---"
    kill -9 $(jobs -rp)
    wait $(jobs -rp) 2>/dev/null
}
trap cleanup EXIT

mkdir -p bin
$(pkill -9 net-http-server || printf "")
$(pkill -9 fasthttp-server || printf "")
$(pkill -9 gnet-http-server || printf "")

function gobench {
    echo "--- $1 ---"
    if [[ "$3" != "" ]]; then
        go build -o $2 $3
    fi

    if [[ "$1" == "GNET" ]]; then
        GOMAXPROCS=8 $2 --port $4 --loops $5 &
    else
        GOMAXPROCS=8 $2 --port $4 &
    fi

    sleep 1
    echo "*** 100 connections, 10 seconds"
    bombardier -c 100 -l http://127.0.0.1:$4
    echo "--- DONE ---"
    echo ""
}

gobench "GO STDLIB" bin/net-http-server net-http-server/main.go 8081
gobench "FASTHTTP" bin/fasthttp-server fasthttp-server/main.go 8083
gobench "GNET" bin/gnet-http-server ../examples/http-server/main.go 8084 -1
