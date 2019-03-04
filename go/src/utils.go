package main

import (
	"encoding/json"
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/umahmood/haversine"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strconv"
)

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func validateLat(latitude float64) bool {
	return latitude <= 90 && latitude >= -90
}

func validateLon(longitude float64) bool {
	return longitude <= 180 && longitude >= -180
}

func validateDist(dist float64) bool {
	return dist <= 40075 && dist >= 0
}

func parseFloat(input string) (float64, bool) {
	result, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, false
	}
	return result, true
}

// this is here, due json convert back and forth of open source exif library did not work
func fixJson(returnVal string) string {
	return "{\"exifs\":[" + returnVal + "]}"
}

func trimLast(string string) string {
	return string[:(len(string)-1)]
}

func decodeExif(image multipart.File) (string, *exif.Exif) {
	exif.RegisterParsers(mknote.All...)
	x, err := exif.Decode(image)
	handleError(err)
	marshaled, err := json.Marshal(x)
	return string(marshaled), x
}

func inRange(
	initialLatitude,
	initialLongitude,
	destinationLatitude,
	destinationLongitude,
	distance float64) bool {
	initialPoint := haversine.Coord{Lat: initialLatitude, Lon: initialLongitude}
	destinationPoint := haversine.Coord{Lat: destinationLatitude, Lon: destinationLongitude}
	_, km := haversine.Distance(initialPoint, destinationPoint)
	return km < distance
}

// currently unused
func saveToFilesystem(file multipart.File, filename string) {
	out, err := os.Create(filename)
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		log.Fatal(err)
	}
}
