#!/bin/bash

docker-compose -f db/docker-compose.yml up -d
go run go/src/main.go
