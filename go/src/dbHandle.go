package main

import (
	"database/sql"
	"fmt"
	"github.com/rwcarlsen/goexif/exif"
)

func connectDatabase() *sql.DB {
	connectString := "host=localhost port=5432 user=imagecurl password=imagecurl dbname=imagecurl sslmode=disable"
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

func saveToDatabase(exif *exif.Exif, exifAsJson string) int {
	lat, lon, _ := exif.LatLong()
	response, err := db.Exec("INSERT INTO exifs(exif) SELECT exif FROM exifs UNION VALUES($1) EXCEPT SELECT exif FROM exifs", exifAsJson)
	handleError(err)
	rowsAffected, err := response.RowsAffected()
	handleError(err)
	if rowsAffected == 0 {
		return -1
	}
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
