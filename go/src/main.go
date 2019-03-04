package main

import (
	"database/sql"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	db = connectDatabase()
	initializeApi()
}
