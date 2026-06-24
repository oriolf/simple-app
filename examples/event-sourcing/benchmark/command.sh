#!/usr/bin/env bash

curl -b cookies -H 'Content-Type: application/json' -d '{"name": "Member", "nif": "11111111H", "joined_on": "2020-01-01"}' http://localhost:8080/api/members
