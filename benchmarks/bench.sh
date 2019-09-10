#!/bin/bash

set -e

cd $(dirname "${BASH_SOURCE[0]}")

mkdir -p out/

./bench-http.sh 2>&1 | tee out/http.txt
./bench-echo.sh 2>&1 | tee out/echo.txt
./bench-redis.sh 1 2>&1 | tee out/redis1.txt
./bench-redis.sh 8 2>&1 | tee out/redis8.txt
./bench-redis.sh 16 2>&1 | tee out/redis16.txt

go run analyze.go