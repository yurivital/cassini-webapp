package main

import (
	"bytes"
	"cassini/website/apiv1"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"cassini/website/database"
)

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	var err error
	mapDb, err = sql.Open("sqlite3", "file::memory:?parseTime=true")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	t.Cleanup(func() {
		if err := mapDb.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	})

	database.CreateMapTables(mapDb)
	database.CreatePositionTables(mapDb)

	router := gin.New()
	router.LoadHTMLGlob("templates/*")
	router.GET("/", GetHome)
	router.POST("/create", CreateMap)
	apiv1.ConfigureRouter(router, mapDb)
	return router
}

func TestGetHome(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Cassini map") {
		t.Fatalf("expected response body to contain page title, got %q", body)
	}
}

func TestCreateMap(t *testing.T) {
	router := setupTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}

	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/map/") {
		t.Fatalf("expected redirect location to start with /map/, got %q", location)
	}

	mapID := strings.TrimPrefix(location, "/map/")
	if mapID == "" {
		t.Fatal("expected redirect location to include a map id")
	}

	// Verify expiration date
	var currentMap, err = database.GetMap(mapDb, mapID)
	t.Log(mapID)
	if err != nil {
		t.Fatalf("failed to execute query: %v", err)
	}
	if currentMap.Created.Add(24*time.Hour) != currentMap.Expire {
		t.Fatalf("expected expiration date to be 24 hour after creation. Got expiration %v ", currentMap.Expire)
	}
}

func TestApiCreateMap(t *testing.T) {
	router := setupTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/map", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	resp := rec.Body.String()
	if resp == "" {
		t.Fatal("expected non-empty response from /api/v1/map")
	}
	t.Log(resp)
	var b bytes.Buffer
	b.WriteString(resp)
	var mapInfo apiv1.MapResponse
	err := json.Unmarshal(b.Bytes(), &mapInfo)
	if err != nil {
		t.Fatalf("failed to unmarshal map info: %v", err)
		return
	}
	if mapInfo.PublicId == "" {
		t.Fatal("expected non-empty map public id")
		return
	}
	var originalMapId = mapInfo.PublicId

	// Retrieve map info
	b.Reset()
	mapInfo = apiv1.MapResponse{}
	var targetUrl = "/api/v1/map/" + originalMapId
	t.Log(targetUrl)
	req = httptest.NewRequest(http.MethodGet, targetUrl, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Get Map Info, expected status %d, got %d", http.StatusOK, rec.Code)
	}
	resp = rec.Body.String()
	if resp == "" {
		t.Fatal("expected non-empty response from /api/v1/map")
	}
	b.WriteString(resp)
	t.Log(b.String())

	err = json.Unmarshal(b.Bytes(), &mapInfo)
	if err != nil {
		t.Fatalf("failed to unmarshal map info: %v", err)
		return
	}
	if mapInfo.PublicId == "" {
		t.Fatal("expected map public id to be non-empty")
	}
	if mapInfo.PublicId != originalMapId {
		t.Fatal("expected map public id to match original map id")
	}
}

func TestGetLastPosition(t *testing.T) {
	router := setupTestRouter(t)

	currentMap, err := database.CreateMap(mapDb)
	if err != nil {
		t.Fatalf("failed to create map: %v", err)
	}

	t0, _ := time.Parse(time.RFC3339, "2023-07-01T12:00:00Z")
	t1, _ := time.Parse(time.RFC3339, "2023-07-01T14:00:00Z")

	_ = database.AppendPosition(mapDb, currentMap.MapPublicId.String(),
		database.Position{
			Timestamp: t0.Unix(),
			Nickname:  "Bob",
			Latitude:  48.8581,
			Longitude: 2.2942,
		})
	_ = database.AppendPosition(mapDb, currentMap.MapPublicId.String(),
		database.Position{
			Timestamp: t1.Unix(),
			Nickname:  "Bob",
			Latitude:  48.8744,
			Longitude: 2.2949,
		})

	var targetGet = "/api/v1/map/" + currentMap.MapPublicId.String() + "/positions"
	var targetPost = "/api/v1/map/" + currentMap.MapPublicId.String() + "/position"

	t.Log(targetGet)

	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(apiv1.PositionResponse{Nickname: "Alice",
		Latitude:  48.8463,
		Longitude: 2.3461,
		Timestamp: t0.Unix()})
	req := httptest.NewRequest(http.MethodPost, targetPost, bytes.NewReader(b.Bytes()))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	b.Reset()

	_ = json.NewEncoder(&b).Encode(apiv1.PositionResponse{Nickname: "Alice",
		Latitude:  48.8530,
		Longitude: 2.3496,
		Timestamp: t1.Unix()})
	req = httptest.NewRequest(http.MethodPost, targetPost, bytes.NewReader(b.Bytes()))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, targetGet, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	resp := rec.Body.String()

	if resp == "" {
		t.Fatal("expected non-empty response from /api/v1/:mapPublicId/positions")
	}
	t.Log(resp)

	var positions apiv1.LastPositionResponse
	err = json.Unmarshal([]byte(resp), &positions)
	if err != nil {
		t.Fatalf("failed to unmarshal positions: %v", err)
		return
	}

	aliceIndex := slices.IndexFunc(positions.LastPositions, func(p apiv1.PositionResponse) bool {
		return p.Nickname == "Alice"
	})
	if aliceIndex == -1 {
		t.Fatal("Alice position not found in response")
	}

	var alicePosition = positions.LastPositions[aliceIndex]

	if alicePosition.Latitude != 48.8530 && alicePosition.Longitude != 2.3496 {
		t.Fatalf("Alice position has unexpected coordinates: %v", alicePosition)
	}
}
