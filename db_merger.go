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
	dbB := OpenDatabase(sourceFileB)
	dbOut := CreateOrOpenDatabase(outFile)

	defer dbA.Close()
	defer dbB.Close()
	defer dbOut.Close()

	rowsA, err := dbA.Query("SELECT zoom_level, tile_column, tile_row FROM tiles")
	defer rowsA.Close()

	if err != nil {
		log.Fatal(err)
	}

	// Copy and merge all tiles from database A

	transaction, err := dbOut.Begin()
	if err != nil {
		log.Fatal("Can't start transaction")
	}

	for rowsA.Next() {
		var zoomLevel int
		var tileRow int64
		var tileCol int64

		rowsA.Scan(&zoomLevel, &tileCol, &tileRow)

		// check if the tile exists in database B, too:

		rowB, err := dbB.Query("SELECT tile_data FROM tiles WHERE zoom_level = $1 AND tile_row = $2 AND tile_column = $3", zoomLevel, tileRow, tileCol)

		if err != nil {
			log.Fatal(err)
		}

		var fullTileData []byte

		rowTileA, err := dbA.Query("SELECT tile_data FROM tiles WHERE zoom_level = $1 AND tile_row = $2 AND tile_column = $3", zoomLevel, tileRow, tileCol)

		if err != nil {
			log.Fatal(err)
		}

		if rowTileA.Next() {
			rowTileA.Scan(&fullTileData)
		} else {
			log.Fatalf("Did not find tile %d %d in db A\n", tileRow, tileCol)
		}

		if rowB.Next() {
			var tileData []byte

			rowB.Scan(&tileData)

			// Merge tiles.
			fullTileData = mergeTiles(fullTileData, tileData)
		}

		// Write tile to output database:
		insertStatement, err2 := transaction.Prepare("INSERT OR REPLACE INTO tiles(zoom_level, tile_column, tile_row, tile_data) values(?, ?, ?, ?)")
		if err2 != nil {
			log.Fatal("Can't start transaction")
		}

		_, err2 = insertStatement.Exec(zoomLevel, tileCol, tileRow, fullTileData)

		rowTileA.Close()
		rowB.Close()
		insertStatement.Close()
	}

	transaction.Commit()

	// Start second large transaction.
	transaction, err = dbOut.Begin()

	// Copy all missing tiles from database B  (only tiles which are not in A)

	rowsB, err2 := dbB.Query("SELECT zoom_level, tile_column, tile_row FROM tiles")
	defer rowsB.Close()

	if err2 != nil {
		log.Fatal(err2)
	}

	for rowsB.Next() {
		var zoomLevel int
		var tileRow int64
		var tileCol int64

		rowsB.Scan(&zoomLevel, &tileCol, &tileRow)
		// Check if tile exists in A (if so, it was already merged)

		rowA, err := dbA.Query("SELECT tile_row FROM tiles WHERE zoom_level = $1 AND tile_row = $2 AND tile_column = $3", zoomLevel, tileRow, tileCol)

		if err != nil {
			log.Fatal(err)
		}

		if !rowA.Next() {
			// Tile does not exist in db A.
			var tileData []byte

			rowB, err := dbB.Query("SELECT tile_data FROM tiles WHERE zoom_level = $1 AND tile_row = $2 AND tile_column = $3", zoomLevel, tileRow, tileCol)

			if err != nil {
				log.Fatal(err)
			}

			if !rowB.Next() {
				log.Fatalf("Could not find tile %d %d\n", tileCol, tileRow)
			}

			rowB.Scan(&tileData)

			insertStatement, err2 := transaction.Prepare("INSERT OR REPLACE INTO tiles(zoom_level, tile_column, tile_row, tile_data) values(?, ?, ?, ?)")

			if err2 != nil {
				log.Fatal("Can't start transaction")
			}

			_, err2 = insertStatement.Exec(zoomLevel, tileCol, tileRow, tileData)

			if err2 != nil {
				log.Fatal("Can't write to output db.")
			}

			rowB.Close()
			insertStatement.Close()
		}

		rowA.Close()
	}

	transaction.Commit()

	// Read bounds and write metadata:

	boundsA, err := dbA.Query("SELECT value FROM metadata WHERE name = ?", "bounds")

	if err != nil {
		log.Fatal(err)
	}

	boundsB, err := dbB.Query("SELECT value FROM metadata WHERE name = ?", "bounds")

	if err != nil {
		log.Fatal(err)
	}

	mergedBounds := mergeGeoBounds(getBounds(boundsA), getBounds(boundsB))

	meta := metadata{name: "pace", description: "pacetiles", bounds: mergedBounds}

	UpdateMetadata(dbOut, &meta)
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
