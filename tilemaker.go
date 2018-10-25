package main

import (
	"sync"
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
	defer profile.Start().Stop()

	var threads = 4
	var qlen = 10000

	inChan := make(chan interface{}, qlen)
	storeChan := make(chan feature, qlen)
	exportChan := make(chan tileFeatures, qlen)
	writeChan := make(chan tileData, qlen)

	var wg sync.WaitGroup

	// Launch reader routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		reader(0, "karlsruhe.pbf", inChan)
		close(inChan)
	}()

	// Launch processor routines
	for w := 0; w < threads; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			processor("example/pace.js", inChan, storeChan)
		}(w)
	}

	// store bounds for writing meta data later
	var (
		minLatitude  float64
		minLongitude float64
		maxLatitude  float64
		maxLongitude float64
	)

	// run our storage routine
	go func() {
		for f := range storeChan {
			// store somehow
			var (
				minCol int
				minRow int
				maxCol int
				maxRow int
			)

			for _, coordinate := range f.coordinates {
				var col = int(ColumnFromLongitudeF(float64(coordinate.longitude), ZOOM))
				var row = int(RowFromLatitudeF(float64(coordinate.latitude), ZOOM))

				if minCol == 0 || minCol > col {
					minCol = col
				}
				if maxCol < col {
					maxCol = col
				}
				if minRow == 0 || minRow > row {
					minRow = row
				}
				if maxRow < row {
					maxRow = row
				}

				if minLatitude == 0 || minLatitude > coordinate.latitude {
					minLatitude = coordinate.latitude
				}
				if maxLatitude == 0 || maxLatitude < coordinate.latitude {
					maxLatitude = coordinate.latitude
				}
				if minLongitude == 0 || minLongitude > coordinate.longitude {
					minLongitude = coordinate.longitude
				}
				if maxLongitude == 0 || maxLongitude < coordinate.longitude {
					maxLongitude = coordinate.longitude
				}
			}

			for c := minCol; c <= maxCol; c++ {
				for r := minRow; r <= maxRow; r++ {
					var idx = ZOOM + (int64(r) * MAX_ROWS) + (int64(c) * (MAX_COLS * MAX_ROWS))
					if t, ok := tiles[idx]; ok {
						t.features = append(t.features, f)
						tiles[idx] = t
					} else {
						t = tileFeatures{ZOOM, int(r), int(c), []feature{f}}
						tiles[idx] = t
					}
				}
			}

		}
	}()

	// Wait until all data is processed (all routines ended)
	wg.Wait()

	// Start exporter routines
	var wgExporter sync.WaitGroup

	// TODO: This seems to be not multi-thread safe
	// TODO: Check why and improve speed
	go func() {
		defer wg.Done()
		defer wgExporter.Done()
		exporter(w, exportChan, writeChan)
	}()

	// Starter writer routine
	wg.Add(1)
	go func() {
		defer wg.Done()

		meta := metadata{name: "pace", description: "pacetiles", bounds: []float64{
			float64(minLongitude),
			float64(minLatitude),
			float64(maxLongitude),
			float64(maxLatitude)}}

		writer(0, writeChan, "karlsruhe.mbtiles", &meta)
	}()

	// Write stored data into exportChan
	for _, features := range tiles {
		exportChan <- features
	}

	close(exportChan)

	// Wait until expoerter finished it's jobs and close write channel
	wgExporter.Wait()

	close(writeChan)

	// Wait until all data is processed (all routines ended)
	wg.Wait()
}
