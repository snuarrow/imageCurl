FROM golang:latest

COPY src/ /usr/src/
EXPOSE 8080
WORKDIR /usr/src/
RUN go get github.com/gin-gonic/gin
RUN go get github.com/lib/pq
RUN go get github.com/rwcarlsen/goexif/exif
RUN go get github.com/rwcarlsen/goexif/mknote
RUN go get github.com/umahmood/haversine
CMD ["go", "run", "main.go"]
