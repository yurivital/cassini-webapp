package database

import (
	"database/sql"
	"log"
	"time"

	"github.com/google/uuid"
)

// Map type represents a map instance
type Map struct {
	Id          int64
	MapPublicId uuid.UUID
	Created     time.Time
	Expire      time.Time
}

// CreateMapTables creates the map sqlite objects
func CreateMapTables(db *sql.DB) {

	mapTableDefinition := `
    CREATE TABLE IF NOT EXISTS map (
    Id INTEGER PRIMARY KEY ASC,
    MapPublicId TEXT UNIQUE NOT NULL,  
    Created DATETIME NOT NULL,
    Expire DATETIME NOT NULL
    ); 
  CREATE INDEX IF NOT EXISTS idx_purge ON map (Expire);
    `

	log.Print("Creating table map")
	db.Exec(mapTableDefinition)
}

// CreateMap creates a new map with a lifetime of 24 hours
func CreateMap(db *sql.DB) (*Map, error) {
	now := time.Now().UTC()
	newMap := Map{
		MapPublicId: uuid.New(),
		Created:     now,
		Expire:      now.Add(24 * time.Hour),
	}

	order := `
    INSERT INTO map (MapPublicId, Created, Expire)
    VALUES (?,?,?)
    `

	stmt, err := db.Prepare(order)
	if err != nil {
		log.Fatal("Unable to prepare map insert statement", err)
		return nil, err
	}

	defer stmt.Close()

	res, err := stmt.Exec(newMap.MapPublicId, newMap.Created, newMap.Expire)
	if err != nil {
		log.Fatal("Unable to insert a new map", err)
		return nil, err
	}
	id, err := res.LastInsertId()

	if err != nil {
		log.Fatal("Unable to obtain internal ID of the new map", err)
		return nil, err
	}
	newMap.Id = id

	return &newMap, nil
}

// GetMap gets a map by id
func GetMap(db *sql.DB, id string) (*Map, error) {
	var newMap Map
	var unixExpire int64
	var unixCreated int64
	stmt, err := db.Prepare("SELECT id, MapPublicId, unixepoch( Expire) Expire, unixepoch( Created) Created FROM map WHERE MapPublicId=?")
	if err != nil {
		log.Fatal("Unable to prepare map select statement", err)
		return nil, err
	}
	defer stmt.Close()
	stmt.QueryRow(id).Scan(&newMap.Id, &newMap.MapPublicId, &unixExpire, &unixCreated)

	if err != nil {
		log.Fatal("Unable to parse expire time", err)
		return nil, err
	}

	newMap.Expire = time.Unix(unixExpire, 0)
	newMap.Created = time.Unix(unixCreated, 0)

	return &newMap, nil
}
