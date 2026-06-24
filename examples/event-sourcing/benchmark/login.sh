#!/usr/bin/env bash

curl -c cookies -d '{"email": "usuari@example.com", "password": "usuariusuari"}' -H 'Content-Type: application/json' http://localhost:8080/api/login
