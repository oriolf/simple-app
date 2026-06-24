#!/usr/bin/env bash

cd ..
go build
cp event-sourcing benchmark
cd -

rm -f db.db
./event-sourcing http > /tmp/event-sourcing.http.log 2>&1 &
backendPID=$!

./benchmark seed

trap cleanup EXIT

function cleanup {
    kill $backendPID
    rm -f event-sourcing
}

go build
./benchmark run -seconds 1 -commandsPerSecond 50
./benchmark clear
./benchmark run -seconds 1 -commandsPerSecond 100
./benchmark clear
./benchmark run -seconds 1 -commandsPerSecond 200
./benchmark clear
./benchmark run -seconds 1 -commandsPerSecond 250
./benchmark clear
./benchmark run -seconds 1 -commandsPerSecond 300
./benchmark clear
./benchmark run -seconds 1 -commandsPerSecond 500
./benchmark clear
./benchmark run -seconds 1 -commandsPerSecond 600
