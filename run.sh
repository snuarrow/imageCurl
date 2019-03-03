#!/bin/bash

docker-compose -f db/docker-compose.yml up -d
go get github.com/gin-gonic/gin
go get github.com/lib/pq
go get github.com/rwcarlsen/goexif/exif
go get github.com/rwcarlsen/goexif/mknote
go get github.com/umahmood/haversine
go run go/src/main.go
