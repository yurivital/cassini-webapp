package main

import (
	"cassini/website/apiv1"
	"log"
	"net/http"
	"time"

	"database/sql"

	"github.com/gin-gonic/gin"

	_ "github.com/mattn/go-sqlite3"

	"cassini/website/database"
)

// DbFile Path to the sqlite database file
const DbFile = "file:maps.sqlite3?_journal=WAL&parseTime=true&_stmt_cache_size=15&_busy_timeout=15000"

var mapDb *sql.DB

// Application entrypoint
func main() {
	router := gin.Default()
	router.StaticFile("/favicon.ico", "./resources/favicon.ico")
	router.StaticFile("/maps.js", "./js/maps.js")
	router.LoadHTMLGlob("templates/*")
	router.GET("/", GetHome)
	router.POST("/create", CreateMap)
	router.GET("/map/:mapId", GetMap)

	var err error
	mapDb, err = sql.Open("sqlite3", DbFile)
	if err != nil {
		log.Fatal(err)
	}

	mapDb.SetMaxOpenConns(4)
	defer func(mapDb *sql.DB) {
		err := mapDb.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(mapDb)

	apiv1.ConfigureRouter(router, mapDb)

	database.CreateMapTables(mapDb)
	database.CreatePositionTables(mapDb)

	err = router.Run(":8080")
	if err != nil {
		log.Fatal(err)
		return
	}

}

// GetHome Returns the home page
func GetHome(c *gin.Context) {
	c.HTML(http.StatusOK, "home.gotmpl", gin.H{
		"title": "Cassini map",
	})
}

// CreateMap Creates a new map instance in the database and redirects to it
func CreateMap(c *gin.Context) {
	newMap, err := database.CreateMap(mapDb)

	if err != nil {
		log.Printf("Unable to create map: %v", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	mapId := newMap.MapPublicId.String()

	log.Printf("New map %s created.", mapId)
	c.Redirect(http.StatusFound, "/map/"+mapId)
}

// GetMap Retrieves a map from the database
func GetMap(c *gin.Context) {
	var mapId = c.Param("mapId")
	log.Printf("Getting map %s", mapId)
	currentMap, err := database.GetMap(mapDb, mapId)
	if err != nil {
		log.Printf("Unable to get map %s", mapId)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if currentMap == nil {
		log.Printf("Map not found : %v", mapId)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if currentMap.Expire.Before(time.Now()) {
		log.Printf("Map expired : %v", mapId)
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	log.Printf("Found map %s", mapId)
	c.HTML(http.StatusOK, "map.gotmpl", gin.H{
		"title":       "Map",
		"MapPublicId": mapId,
		"Expire":      currentMap.Expire.Format(time.DateTime),
	})
}
