#!/bin/bash

set -e

pl=$1
if [ "$pl" == "" ]; then
    pl="1"
fi

echo ""
echo "--- BENCH REDIS PIPELINE $pl START ---"
echo ""

cd $(dirname "${BASH_SOURCE[0]}")
function cleanup {
    echo "--- BENCH REDIS PIPELINE $pl DONE ---"
    kill -9 $(jobs -rp)
    wait $(jobs -rp) 2>/dev/null
}
trap cleanup EXIT

mkdir -p bin
$(pkill -9 redis-server || printf "")
$(pkill -9 evio-redis-server || printf "")

function gobench {
    echo "--- $1 ---"
    if [ "$3" != "" ]; then
        go build -o $2 $3
    fi
    GOMAXPROCS=1 $2 --port $4 &
    sleep 1
    echo "*** 50 connections, 1000000 commands, $pl commands pipeline"
    redis-benchmark -p $4 -t ping_inline -q -c 50 -P $pl -n 1000000
    # echo "*** 50 connections, 1000000 commands, 10 commands pipeline"
    # redis-benchmark -p $4 -t ping_inline -q -c 50 -P 10 -n 1000000
    # echo "*** 50 connections, 1000000 commands, 20 commands pipeline"
    # redis-benchmark -p $4 -t ping_inline -q -c 50 -P 20 -n 1000000
    echo "--- DONE ---"
    echo ""
}
gobench "REAL REDIS" redis-server "" 6392
gobench "EVIO REDIS CLONE" bin/evio-redis-server ../examples/redis-server/main.go 6393
