#!/usr/bin/env bash

function check {
    curl -s "${@:2}" > "got/$1.json"
    diff <(jq --sort-keys . "expected/$1.json") <(jq --sort-keys . "got/$1.json")
}

check 001 http://localhost:8080/members
check 002 -X POST -d '{"name": "Pepe", "nif": "00000000T", "joined_on": "2000-01-01"}' http://localhost:8080/members

