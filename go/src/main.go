package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/umahmood/haversine"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strconv"
	"time"
)

var db *sql.DB

func main() {
	db = connectDatabase()
	initializeApi()
}

func initializeApi() {
	router := gin.Default()
	router.GET("/inRange", query)
	router.GET("/id", getId)
	router.POST("/image", imagePost)
	router.GET("/", ping)
	router.POST("/shutdown", shutdown)
	err := router.Run() // listen and serve on 0.0.0.0:8080
	handleError(err)
}

func ping(c *gin.Context) {
	c.String(200, "ping")
}

func shutdown(c *gin.Context) {
	c.String(200, "shutdown ok")
	go func() {
		time.Sleep(time.Second)
		os.Exit(0)
	}()
}

func query(c *gin.Context) {
	lat := c.Query("decimal_latitude")
	lon := c.Query("decimal_longitude")
	dist := c.Query("distance_km")
	lat64, err := strconv.ParseFloat(lat, 64)
	handleError(err)
	lon64, err := strconv.ParseFloat(lon, 64)
	handleError(err)
	dist64, err := strconv.ParseFloat(dist, 64)
	harvested := harvest(lat64, lon64, dist64)
	var returnVal string
	for _, element := range harvested {
		returnVal += element+","
	}
	if len(returnVal) > 0 {
		returnVal = fixJson(trimLast(returnVal))
		c.String(200, returnVal)
	} else {
		c.String(200, fixJson(returnVal))
	}
}

func harvest(initialLatitude, initialLongitude, distance float64) []string {
	sqlStatement := `SELECT * FROM points`
	rows, _ := db.Query(sqlStatement)
	defer rows.Close()

	ids := ""
	for rows.Next() {
		var lat, lon float64
		var id int
		err := rows.Scan(&id, &lat, &lon)
		handleError(err)
		if inRange(initialLatitude, initialLongitude, lat, lon, distance) {
			ids += fmt.Sprintf("%d,", id)
		}
	}

	exifs := make([]string, 0)
	if len(ids) > 1 {
		//ids = ids[:(len(ids)-1)]
		ids = trimLast(ids)
		sqlStatement = "SELECT exif FROM exifs WHERE id IN (" + ids + ")"
		rows, _ := db.Query(sqlStatement)
		defer rows.Close()
		for rows.Next() {
			var exif string
			err := rows.Scan(&exif)
			handleError(err)
			exifs = append(exifs, exif)
		}
	}
	return exifs
}

// this is here, due json convert back and forth of open source exif library did not work
func fixJson(returnVal string) string {
	return "{\"exifs\":[" + returnVal + "]}"
}

func trimLast(string string) string {
	return string[:(len(string)-1)]
}

func getId(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	handleError(err)
	rows, _ := db.Query(fmt.Sprintf("SELECT exif FROM exifs WHERE id IN (%d)", id))
	defer rows.Close()
	var exif string
	for rows.Next() {
		error := rows.Scan(&exif)
		handleError(error)
	}
	if exif == "" {
		c.JSON(400, gin.H{"error": "not found"})
	} else {
		c.String(200, exif)
	}
}

func imagePost(c *gin.Context) {
	file, _ , err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "bad request"})
		return
	}
	exifAsJson, decodedExif := decodeExif(file)
	id := saveToDatabase(decodedExif, exifAsJson)
	if id == -1 {
		c.JSON(409, gin.H{"error": "exif conflicts with existing"})
	} else {
		c.JSON(201, gin.H{"id": id})
	}
}

func connectDatabase() *sql.DB {
	connectString := "host=localhost port=5432 user=imagecurl password=salasana12 dbname=imagecurl sslmode=disable"
	db, err := sql.Open("postgres", connectString)
	if err != nil {
		fmt.Println("database open error", err)
	}
	handleError(err)
	initDatabase(db)
	return db
}

func initDatabase(db *sql.DB) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS points (id SERIAL PRIMARY KEY, lat float, lon float)")
	handleError(err)
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS exifs (id SERIAL PRIMARY KEY, exif VARCHAR NOT NULL)")
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func saveToDatabase(exif *exif.Exif, exifAsJson string) int {
	lat, lon, _ := exif.LatLong()
	response, err := db.Exec("INSERT INTO exifs(exif) SELECT exif FROM exifs UNION VALUES($1) EXCEPT SELECT exif FROM exifs", exifAsJson)
	handleError(err)
	rowsAffected, err := response.RowsAffected()
	handleError(err)
	if rowsAffected == 0 {
		return -1
	}
	//rows, _ := db.Query("SELECT currval(pg_get_serial_sequence('exifs', 'id'))")
	rows, _ := db.Query("SELECT id FROM exifs")
	defer rows.Close()
	var id = -1
	for rows.Next() {
		err := rows.Scan(&id)
		handleError(err)
	}
	if id != -1 {
		_, err = db.Exec("INSERT INTO points(id, lat, lon) VALUES($1, $2, $3)", id, lat, lon)
		handleError(err)
	}
	return id
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