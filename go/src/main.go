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
)

func main() {
	initializeApi()
}

func initializeApi() {
	router := gin.Default()
	router.GET("/inRangess", query)
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
	defer os.Exit(0)
	c.String(200, "shutdown ok")
}

// /query?decimal_latitude=1.23&decimal_longitude=2.34&distance_km=50.2
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
		fmt.Println(element)
		returnVal += element+"\n"
	}

	c.String(200, returnVal)
}

// 60.17935277777778 ,  24.816994444444444
func harvest(initialLatitude, initialLongitude, distance float64) []string {
	db := connectDatabase()
	sqlStatement := `SELECT * FROM points`
	rows, _ := db.Query(sqlStatement)
	defer rows.Close()

	ids := ""
	for rows.Next() {
		var lat, lon float64
		var id int
		rows.Scan(&id, &lat, &lon)
		if inRange(initialLatitude, initialLongitude, lat, lon, distance) {
			ids += fmt.Sprintf("%d,", id)
		}
	}

	exifs := make([]string, 0)
	if len(ids) > 1 {
		ids = ids[:(len(ids)-1)]
		sqlStatement = "SELECT exif FROM exifs WHERE id IN (" + ids + ")"
		rows, _ := db.Query(sqlStatement)
		defer rows.Close()
		for rows.Next() {
			var exif string
			err := rows.Scan(&exif)
			handleError(err)
			exifs = append(exifs, exif+"\n")
		}
	}
	return exifs
}

func getId(c *gin.Context) {
	//idAsString := c.Query("id")
	id, err := strconv.Atoi(c.Query("id"))
	handleError(err)
	db := connectDatabase()
	rows, _ := db.Query(fmt.Sprintf("SELECT exif FROM exifs WHERE id IN (%d)", id))
	defer rows.Close()
	var exif string
	for rows.Next() {
		error := rows.Scan(&exif)
		handleError(error)
	}
	c.String(200, exif)
}

func imagePost(c *gin.Context) {
	file, _ , err := c.Request.FormFile("file")
	handleError(err)
	exifAsJson, decodedExif := decodeExif(file) // exif from here
	id := saveToDatabase(decodedExif, exifAsJson)
	c.JSON(200, gin.H{ "id": id })
}

func connectDatabase() *sql.DB {
	//connectString := "user=postgres dbname=exifdb password=salasana12 host=0.0.0.0 sslmode=disable"
	connectString := "host=127.0.0.1 port=10042 user=imagecurl password=salasana12 dbname=imagecurl sslmode=disable"
	fmt.Println(connectString)
	db, err := sql.Open("postgres", connectString)
	handleError(err)
	initDatabase(db)
	return db
}

func initDatabase(db *sql.DB) {
	db.Exec("CREATE TABLE IF NOT EXISTS points (id SERIAL PRIMARY KEY, lat float, lon float)")
	db.Exec("CREATE TABLE IF NOT EXISTS exifs (id SERIAL PRIMARY KEY, exif VARCHAR NOT NULL)")
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func saveToDatabase(exif *exif.Exif, exifAsJson string) int {
	//connectString := "user=postgres dbname=exifdb password=salasana12 host=0.0.0.0 sslmode=disable"
	//db, err := sql.Open("postgres", connectString)
	//handleError(err)
	db := connectDatabase()

	lat, lon, _ := exif.LatLong()
	db.Exec("INSERT INTO exifs(exif) VALUES($1)", exifAsJson)
	rows, _ := db.Query("SELECT currval(pg_get_serial_sequence('exifs', 'id'))")
	defer rows.Close()
	var id int
	for rows.Next() {
		rows.Scan(&id)
		fmt.Println("current id:", id)
		_, err := db.Exec("INSERT INTO points(id, lat, lon) VALUES($1, $2, $3)", id, lat, lon)
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
	//fmt.Println("Miles:",mi,"Kilometers:",km)
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