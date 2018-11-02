package main

import (
	"database/sql"
	"log"
	"strconv"
	"strings"
)

func mergeDatabases(sourceFileA string, sourceFileB string, outFile string) {
	log.Printf("Merging %s and %s to %s...\n", sourceFileA, sourceFileB, outFile)
	dbA := OpenDatabase(sourceFileA)
	dbOut := CreateOrOpenDatabase(outFile)

	dbOut.Close()

	defer dbA.Close()

	// attach all databases to dbA:
	_, err := dbA.Exec("ATTACH './" + sourceFileB + "' AS dbB")

	if err != nil {
		log.Fatalf("Failed to attach database: %s", err)
	}

	_, err = dbA.Exec("ATTACH './" + outFile + "' AS dbOut")

	if err != nil {
		log.Fatalf("Failed to attach database: %s", err)
	}

	// Copy all tiles which are exclusively in either database.

	_, err = dbA.Exec("INSERT OR REPLACE INTO dbOut.tiles SELECT * FROM tiles EXCEPT SELECT * FROM dbB.tiles")

	if err != nil {
		log.Fatalf("Failed to copy tiles from A to OUT: %s", err)
	}

	_, err = dbA.Exec("INSERT OR REPLACE INTO dbOut.tiles SELECT * FROM dbB.tiles EXCEPT SELECT * FROM tiles")

	if err != nil {
		log.Fatalf("Failed to copy tiles from B to OUT: %s", err)
	}

	// Detach dbOut:
	_, err = dbA.Exec("DETACH dbOut")

	if err != nil {
		log.Fatalf("Failed to detach dbOut: %s", err)
	}

	// Merge tiles which are in both databases:

	rowsBoth, err := dbA.Query("SELECT t1.zoom_level, t1.tile_column, t1.tile_row, t1.tile_data, t2.tile_data FROM tiles AS t1 INNER JOIN dbB.tiles AS t2 ON t1.tile_column=t2.tile_column AND t1.tile_row=t2.tile_row AND t1.zoom_level = t2.zoom_level")

	if err != nil {
		log.Fatalf("Failed to select border tiles: %s", err)
	}

	tCount := 0

	dbOut = OpenDatabase(outFile)
	defer dbOut.Close()

	transaction, err := dbOut.Begin()
	if err != nil {
		log.Fatalf("Cannot start transaction: %s\n", err)
	}

	for rowsBoth.Next() {
		var zoomLevel int
		var tileRow int64
		var tileCol int64
		var tileDataA []byte
		var tileDataB []byte

		rowsBoth.Scan(&zoomLevel, &tileCol, &tileRow, &tileDataA, &tileDataB)

		fullTileData := mergeTiles(tileDataA, tileDataB)

		insertStatement, err := transaction.Prepare("INSERT OR REPLACE INTO tiles(zoom_level, tile_column, tile_row, tile_data) VALUES(?, ?, ?, ?)")

		if err != nil {
			log.Fatalf("Cannot prepare insert query: %s\n", err)
		}

		_, err = insertStatement.Exec(zoomLevel, tileCol, tileRow, fullTileData)

		if err != nil {
			log.Fatalf("Cannot insert tile %s\n", err)
		}

		insertStatement.Close()

		tCount++
	}

	log.Printf("Merged %d border tiles\n", tCount)

	transaction.Commit()
	rowsBoth.Close()

	// Read bounds and write metadata:

	boundsA, err := dbA.Query("SELECT value FROM metadata WHERE name = ?", "bounds")

	if err != nil {
		log.Fatal(err)
	}

	dbB := OpenDatabase(sourceFileB)
	defer dbB.Close()

	boundsB, err := dbB.Query("SELECT value FROM metadata WHERE name = ?", "bounds")

	if err != nil {
		log.Fatal(err)
	}

	mergedBounds := mergeGeoBounds(getBounds(boundsA), getBounds(boundsB))

	meta := metadata{name: "pace", description: "pacetiles", bounds: mergedBounds}

	UpdateMetadata(dbOut, &meta)

	log.Println("Done.")
}

func getBounds(bounds *sql.Rows) []float64 {
	if bounds.Next() {
		var content string
		bounds.Scan(&content)

		parts := strings.Split(content, ",")

		if len(parts) != 4 {
			log.Fatal("Expected four coordinates as bounds.")
		}

		result := make([]float64, 4)

		for i, a := range parts {
			result[i] = getFloat(a)
		}

		return result
	}

	log.Fatal("Could not find bounds in database metadata table.")

	return make([]float64, 0)
}

func getFloat(a string) float64 {
	result, err := strconv.ParseFloat(a, 64)

	if err != nil {
		log.Fatalf("Failed to parse bound %s to float\n", a)
	}

	return result
}

/* Should only be called for testing, verifies that large db holds all tiles from small db */
func verifyMerge(sourceFile string, outFile string) {
	log.Printf("Verifying %s...\n", sourceFile)
	smallDb := OpenDatabase(sourceFile)
	largeDb := OpenDatabase(outFile)

	defer smallDb.Close()
	defer largeDb.Close()

	rows, err := smallDb.Query("SELECT zoom_level, tile_column, tile_row FROM tiles")
	defer rows.Close()

	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var zoomLevel int
		var tileRow int64
		var tileCol int64

		rows.Scan(&zoomLevel, &tileCol, &tileRow)

		other, err := largeDb.Query("SELECT tile_row, tile_column FROM tiles WHERE zoom_level = $1 AND tile_row = $2 AND tile_column = $3", zoomLevel, tileRow, tileCol)

		if err != nil {
			log.Fatal(err)
		}

		if !other.Next() {
			log.Fatalf("Did not find tile %d %d in large db\n", tileRow, tileCol)
		}

		other.Close()
	}

	log.Printf("Verified that %s is subset of %s\n", sourceFile, outFile)
}
