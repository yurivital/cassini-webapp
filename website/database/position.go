package database

import (
	"database/sql"
	"errors"
	"log"
)

// Position represents a user's position on a map at a specific timestamp
type Position struct {
	Timestamp int64
	MapId     int64
	Nickname  string
	Latitude  float64
	Longitude float64
}

// CreatePositionTables creates the position sqlite objects
func CreatePositionTables(db *sql.DB) {

	tableDefinition := `
	CREATE TABLE IF NOT EXISTS position (
		Id INTEGER PRIMARY KEY AUTOINCREMENT,
		MapId INTEGER NOT NULL,
		Nickname TEXT NOT NULL,
		Timestamp INTEGER NOT NULL,
		Latitude REAL NOT NULL,
		Longitude REAL NOT NULL,
		FOREIGN KEY (MapId) REFERENCES map(Id)
	);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique ON position (MapId, Nickname, Timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_map ON position (MapId);
CREATE INDEX IF NOT EXISTS idx_timestamp ON position (Timestamp ASC);
`
	log.Print("Creating table position")

	_, err := db.Exec(tableDefinition)
	if err != nil {
		panic(err)
	}
}

// GetLastPositions gets the last position of each user on a map
func GetLastPositions(db *sql.DB, mapPublicId string) ([]Position, error) {
	stmt, err := db.Prepare(`
WITH RankedPositions AS (
    SELECT 
        Nickname,
        Timestamp,
        Latitude,
        Longitude,
        ROW_NUMBER() OVER (PARTITION BY Nickname ORDER BY Timestamp DESC) AS rn
    FROM position 
    INNER JOIN map ON position.MapId = map.Id AND map.MapPublicId = ?
)
SELECT Nickname, Timestamp, Latitude, Longitude
FROM RankedPositions
WHERE rn = 1`)
	if err != nil {
		log.Fatal("Unable to prepare position select statement", err)
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(mapPublicId)
	defer rows.Close()
	if err != nil {
		log.Fatal("Unable to execute position select statement", err)
		return nil, err
	}

	var positions []Position
	for rows.Next() {
		currentPosition := Position{}
		err := rows.Scan(&currentPosition.Nickname, &currentPosition.Timestamp, &currentPosition.Latitude, &currentPosition.Longitude)
		if err != nil {
			return nil, err
		}
		positions = append(positions, currentPosition)
	}
	return positions, nil
}

// AppendPosition appends a position to a map
func AppendPosition(db *sql.DB, mapPublicId string, position Position) error {
	parentMapStmt, err := db.Prepare("SELECT Id FROM map WHERE MapPublicId=? LIMIT 1")
	if err != nil {
		log.Fatal("Unable to prepare map select statement", err)
		return err
	}
	defer parentMapStmt.Close()
	err = parentMapStmt.QueryRow(mapPublicId).Scan(&position.MapId)
	if err != nil {
		return err
	}

	if position.MapId == 0 {
		return errors.New("map not found")
	}

	stmt, err := db.Prepare("INSERT INTO position (MapId, Nickname, Timestamp, Latitude, Longitude) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		log.Fatal("Unable to prepare position insert statement", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(position.MapId, position.Nickname, position.Timestamp, position.Latitude, position.Longitude)
	return err
}
