package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"math"

	_ "github.com/mattn/go-sqlite3"
)

type metadata struct {
	name        string
	description string
	bounds      []float64 // Order: left, bottom, right, top
}

type tileData struct {
	zoomLevel int
	row       int
	column    int
	data      []byte
}

func writer(id int, jobs <-chan tileData, destFile string, meta *metadata) {
	log.Printf("Exporting to mbtiles database `%s`. Note: If database already exists, tiles will be replaced / added", destFile)
	db := CreateOrOpenDatabase(destFile)
	defer db.Close()

	UpdateMetadata(db, meta)
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

	for tile := range jobs {
		flippedRow := int(math.Pow(2, float64(tile.zoomLevel))) - tile.row - 1

		_, err = insertStatement.Exec(tile.zoomLevel, tile.column, flippedRow, tile.data)

		if err != nil {
			log.Fatal(err)
		}
	}

	insertStatement.Close()
	transaction.Commit()
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
	statements := []string{
		"PRAGMA application_id = 0x4d504258;",
		"CREATE TABLE metadata (name text, value text);",
		"CREATE TABLE tiles (zoom_level integer, tile_column integer, tile_row integer, tile_data blob);",
		"CREATE UNIQUE INDEX metadata_index on metadata (name);",
		"CREATE UNIQUE INDEX tile_index on tiles (zoom_level, tile_column, tile_row);"}

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
	InsertMetadata(db, "version", "3.3")
	InsertMetadata(db, "description", meta.description)
	InsertMetadata(db, "format", "pbf")

	bounds := strings.Trim(strings.Replace(fmt.Sprint(meta.bounds[:]), " ", ",", -1), "[]")
	InsertMetadata(db, "bounds", bounds)

	InsertMetadata(db, "json", `{"vector_layers":[{"id":"r1","minzoom":16,"maxzoom":16,"fields":{"class":"String"}},{"id":"r2","minzoom":16,"maxzoom":16,"fields":{"class":"String"}},{"id":"r3","minzoom":16,"maxzoom":16,"fields":{"class":"String"}},{"id":"r4","minzoom":16,"maxzoom":16,"fields":{"class":"String"}},{"id":"pi","minzoom":16,"maxzoom":16,"fields":{"class":"String"}},{"id":"hn","minzoom":16,"maxzoom":16,"fields":{"class":"String"}}]}`)

	// PB: Not sure if needed
	InsertMetadata(db, "scheme", "tms")
	InsertMetadata(db, "maskLevel", "8")
	InsertMetadata(db, "minzoom", "16")
	InsertMetadata(db, "maxzoom", "16")
	InsertMetadata(db, "id", "openmaptiles")
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
