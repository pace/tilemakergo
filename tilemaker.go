package main

import (
	"flag"
	"sync"
	"log"

	"github.com/pkg/profile"
)

const (
	featureTypeUnknown = iota
	featureTypePoint
	featureTypeLine
	featureTypePolygon
)

const (
	ZOOM     = 16
	MAX_ROWS = 524288
	MAX_COLS = 524288
)

type coordinate struct {
	latitude  float64
	longitude float64
}

type feature struct {
	id          int64
	typ         uint8
	layer       string
	coordinates []coordinate
	properties  map[string]interface{}
}

type tileFeatures struct {
	zoomLevel int
	row       int
	column    int
	features  []feature
}

var nodeCoordinates = map[int64]coordinate{}
var nodeCoordinatesSemaphore = make(chan struct{}, 1)

// Access to array:
// tiles[zoomlevel + (row * row_count) + (column * (column_count * row_count))]
var tiles = map[int64]tileFeatures{}

func main() {
	defer profile.Start(profile.MemProfile).Stop()

	// Parse & validate arguments
	inputFilePtr := flag.String("in", "input.osm.pbf", "The osm pbf file to parse")
	outputFilePtr := flag.String("out", "output.mbtiles", "The output mbtiles database. If it already exists, an upsert will be performed")
	processorFilePtr := flag.String("processor", "processor.js", "The javascript file to process the content")

    flag.Parse()

    // Wait group

    log.Printf("Start parsing of %s -> %s [%s]", *inputFilePtr, *outputFilePtr, *processorFilePtr)

	var wg sync.WaitGroup
	var qlen = 10000
	// var threads = 8

	storeChan := make(chan feature, qlen)
	metaChan := make(chan bounds)
	exportChan := make(chan tileFeatures, qlen)
	writeChan := make(chan tileData, qlen)

	reader(*inputFilePtr, storeChan, metaChan)

	// run our storage routine
	// TODO: make multithread?
	go func() {
		for f := range storeChan {
			writtenIndexes := map[int64]bool{}

			for _, coord := range f.coordinates {
				var col = int(ColumnFromLongitudeF(float64(coord.longitude), ZOOM))
				var row = int(RowFromLatitudeF(float64(coord.latitude), ZOOM))
				var idx = ZOOM + (int64(row) * MAX_ROWS) + (int64(col) * (MAX_COLS * MAX_ROWS))

				if written, contained := writtenIndexes[idx]; contained && written {
					continue
				}

				writtenIndexes[idx] = true

				if t, ok := tiles[idx]; ok {
					t.features = append(t.features, f)
					tiles[idx] = t
				} else {
					t = tileFeatures{ZOOM, int(row), int(col), []feature{f}}
					tiles[idx] = t
				}
			}
		}

		log.Printf("Store")

		for _, features := range tiles {
			exportChan <- features
		}

		close(exportChan)

		log.Printf("Mapper done")
	}()

	// Wait until all data is processed (all routines ended)
	wg.Wait()

	log.Printf("Start export")

	// Start exporter routines
	var wgExporter sync.WaitGroup

	// TODO: This seems to be not multi-thread safe
	// TODO: Check why and improve speed
	wg.Add(1)
	wgExporter.Add(1)
	go func() {
		defer wg.Done()
		defer wgExporter.Done()
		exporter(0, exportChan, writeChan)
	}()

	// Starter writer routine
	wg.Add(1)
	go func() {
		defer wg.Done()

		for m := range metaChan {
			meta := metadata{name: "pace", description: "pacetiles", bounds: []float64{
				float64(m.minLongitude),
				float64(m.minLatitude),
				float64(m.maxLongitude),
				float64(m.maxLatitude)}}

			writer(0, writeChan, *outputFilePtr, &meta)
		}
	}()

	// Wait until expoerter finished it's jobs and close write channel
	wgExporter.Wait()

	close(writeChan)

	// Wait until all data is processed (all routines ended)
	wg.Wait()
}
