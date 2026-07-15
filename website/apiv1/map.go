package apiv1

import (
	"cassini/website/database"
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// db is the database connection
var mapDb *sql.DB

type MapResponse struct {
	PublicId string `json:"publicId"`
	Expire   int64  `json:"expire"`
}

// PositionResponse represents the last position of a user on a map
type PositionResponse struct {
	Nickname  string  `json:"nickname"  binding:"required"`
	Latitude  float64 `json:"latitude"  binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Timestamp int64   `json:"timestamp" binding:"required"`
}

// LastPositionResponse represents the last positions of users on a map
type LastPositionResponse struct {
	LastPositions []PositionResponse `json:"lastPositions"`
}

// ConfigureRouter configures the router for the API
func ConfigureRouter(router *gin.Engine, db *sql.DB) {
	mapDb = db
	v1 := router.Group("/api/v1")
	v1.POST("map", CreateMap)
	v1.GET("map/:mapPublicId", GetMapInfo)
	v1.GET("map/:mapPublicId/positions", GetMapPositions)
	v1.POST("map/:mapPublicId/position", SetMapPosition)

}

// CreateMap creates a new map
func CreateMap(c *gin.Context) {
	mapInfo, err := database.CreateMap(mapDb)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create map"})
		return
	}
	c.JSON(http.StatusCreated, MapResponse{
		PublicId: mapInfo.MapPublicId.String(),
		Expire:   mapInfo.Expire.Unix(),
	})
}

// GetMapInfo retrieves the info for a given map
func GetMapInfo(c *gin.Context) {
	var mapPublicId = strings.TrimSpace(c.Param("mapPublicId"))
	if mapPublicId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Map public id is required"})
		return
	}
	var mapInfo, err = database.GetMap(mapDb, mapPublicId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve map info"})
		return
	}
	c.JSON(http.StatusOK, MapResponse{
		PublicId: mapInfo.MapPublicId.String(),
		Expire:   mapInfo.Expire.Unix(),
	})
}

// GetMapPositions retrieves the last users positions for a given map
func GetMapPositions(c *gin.Context) {
	var mapPublicId = strings.TrimSpace(c.Param("mapPublicId"))
	if mapPublicId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Map public id is required"})
		return
	}

	var positions, err = database.GetLastPositions(mapDb, mapPublicId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve last positions"})
		return
	}

	c.JSON(http.StatusOK, LastPositionResponse{LastPositions: positionsToLastPositions(positions)})
}

// SetMapPosition saves a new position for a given map
func SetMapPosition(c *gin.Context) {
	var mapPublicId = strings.TrimSpace(c.Param("mapPublicId"))
	if mapPublicId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Map public id is required"})
		return
	}

	var position PositionResponse

	if err := c.ShouldBindJSON(&position); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		log.Print(err)
		return
	}

	if len(strings.TrimSpace(position.Nickname)) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nickname is required and must be at least 2 characters long"})
		return
	}
	if position.Timestamp == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Timestamp is required"})
		return
	}
	var dbPosition = database.Position{
		Nickname:  strings.TrimSpace(position.Nickname),
		Latitude:  position.Latitude,
		Longitude: position.Longitude,
		Timestamp: position.Timestamp,
	}
	err := database.AppendPosition(mapDb, mapPublicId, dbPosition)
	if err != nil {
		log.Print(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save position"})
		return
	}
	c.JSON(http.StatusNoContent, gin.H{})

}

// convertPositionToLastPosition converts a Position to PositionResponse
func convertPositionToLastPosition(pos database.Position) PositionResponse {
	return PositionResponse{
		Nickname:  pos.Nickname,
		Latitude:  pos.Latitude,
		Longitude: pos.Longitude,
		Timestamp: pos.Timestamp,
	}
}

// positionsToLastPositions converts an array of Position to an array of PositionResponse
func positionsToLastPositions(positions []database.Position) []PositionResponse {
	lastPositions := make([]PositionResponse, len(positions))
	for i, pos := range positions {
		lastPositions[i] = convertPositionToLastPosition(pos)
	}
	return lastPositions
}
