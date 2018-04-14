package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
)

type metadata struct {
	name string
	description string
	bounds []float32 // Order: left, bottom, right, top
}

type tileData struct {
    zoomLevel int
    row  int
    column int
    data []byte
}

func ExportTiles(path string, tiles []tileData, meta *metadata) {
	log.Printf("Exporting to mbtiles database `%s`. Note: If database already exists, tiles will be replaced / added", path)
	db := CreateOrOpenDatabase(path)

	UpdateMetadata(db, meta)
	WriteTileData(db, tiles)

	defer db.Close()
}

func CreateOrOpenDatabase(path string) *sql.DB {
	exist := true

	if _, err := os.Stat(path); os.IsNotExist(err) {
		exist = false
	}

	db, err := sql.Open("sqlite3", path)

	if err != nil {
		log.Fatal(fmt.Sprintf("Could not find database at %s", path))
	}

	if !exist {
		log.Printf("Creating database schema")
		CreateSchema(db)
	}

	return db
}

func CreateSchema(db *sql.DB) {
	// This creates the mbtiles schema as described in https://github.com/mapbox/mbtiles-spec/blob/master/1.3/spec.md
	statements := []string {
		"CREATE TABLE metadata (name text, value text);",
		"CREATE TABLE tiles (zoom_level integer, tile_column integer, tile_row integer, tile_data blob);",
		"CREATE UNIQUE INDEX tile_index on tiles (zoom_level, tile_column, tile_row);" }

	for _, statement := range statements {
		_, err := db.Exec(statement)

		if err != nil {
			log.Printf("%q: %s\n", err, statement)
			return
		}
	}
}

func UpdateMetadata(db *sql.DB, meta *metadata) {
	InsertMetadata(db, "name", meta.name)
	InsertMetadata(db, "type", "overlay")
	InsertMetadata(db, "version", "1")
	InsertMetadata(db, "description", meta.description)
	InsertMetadata(db, "format", "pbf")


	bounds := strings.Trim(strings.Replace(fmt.Sprint(meta.bounds[:]), " ", ",", -1), "[]")
	InsertMetadata(db, "bounds", bounds)
}

func InsertMetadata(db *sql.DB, name string, value string) {
	transaction, err := db.Begin()

	if err != nil {
		log.Fatal("Can't start transaction")
		return
	}

	insertStatement, err := transaction.Prepare("INSERT OR REPLACE INTO metadata(name, value) values(?, ?)")
	_, err = insertStatement.Exec(name, value)
	transaction.Commit()
}

func WriteTileData(db *sql.DB, tiles []tileData) {
	log.Printf("Writing %d tiles to the database", len(tiles))

	transaction, err := db.Begin()

	if err != nil {
		log.Fatal("Can't start transaction")
		return
	}

	insertStatement, err := transaction.Prepare("insert or replace into tiles(zoom_level, tile_column, tile_row, tile_data) values(?, ?, ?, ?)")

	if err != nil {
		log.Fatal("Can't start transaction")
		return
	}

	defer insertStatement.Close()

	for _, tile := range tiles {
		_, err = insertStatement.Exec(tile.zoomLevel, tile.column, tile.row, tile.data)

		if err != nil {
			log.Fatal(err)
		}
	}

	transaction.Commit()

	log.Printf("Successfully wrote %d tiles to the database", len(tiles))

}