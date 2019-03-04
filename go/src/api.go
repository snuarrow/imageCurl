package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"os"
	"strconv"
	"time"
)

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

func badRequest(c *gin.Context) {
	fmt.Println("bad request")
	c.JSON(400, gin.H{"error": "bad request"})
}

func query(c *gin.Context) {
	lat64, valid := parseFloat(c.Query("decimal_latitude"))
	fmt.Println("valid", valid, "validateLat(lat64)", validateLat(lat64))
	if !valid || !validateLat(lat64) {
		badRequest(c)
		return
	}
	lon64, valid := parseFloat(c.Query("decimal_longitude"))
	if !valid || !validateLon(lon64) {
		badRequest(c)
		return
	}
	dist64, valid := parseFloat(c.Query("distance_km"))
	if !valid || !validateDist(dist64) {
		badRequest(c)
		return
	}
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