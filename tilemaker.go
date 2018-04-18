package main

import (
	"log"
	"sync"

	"./javascript"
	"github.com/qedus/osmpbf"
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

func processor(id int, jobs <-chan interface{}, results chan<- feature) {
	var js = new(javascript.JavascriptEngine)
	js.Load("test.js")

	for v := range jobs {
		switch v := v.(type) {
		case *osmpbf.Node:

			// Store coords for later useWay
			nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
			nodeCoordinates[v.ID] = coordinate{v.Lat, v.Lon}
			<-nodeCoordinatesSemaphore // Release

			// Process Node v.
			retValue, _ := js.Call(`useNode`, v)
			retBool, _ := retValue.ToBoolean()
			if retBool {
				retValue, _ = js.Call(`processNode`, v)

				// layer
				layerValue, _ := retValue.Object().Get("layer")
				layerString, _ := layerValue.ToString()

				// TODO: Generate properties map

				// create and pass feature
				nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{nodeCoordinates[v.ID]}, nil}
				results <- retFeature
				<-nodeCoordinatesSemaphore // Release
			}

		case *osmpbf.Way:
			// Process Way v.
			retValue, _ := js.Call(`useWay`, v)
			retBool, _ := retValue.ToBoolean()
			if retBool {
				retValue, _ = js.Call(`processWay`, v)

				// layer
				layerValue, _ := retValue.Object().Get("layer")
				layerString, _ := layerValue.ToString()

				// create and pass feature
				nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				var coordinates []coordinate
				for _, nodeID := range v.NodeIDs {
					coordinates = append(coordinates, nodeCoordinates[nodeID])
				}
				retFeature := feature{v.ID, featureTypeLine, layerString, coordinates, nil}
				results <- retFeature
				<-nodeCoordinatesSemaphore // Release
			}

		case *osmpbf.Relation:
			// Process Relation v.
			retValue, _ := js.Call(`useRelation`, v)
			retBool, _ := retValue.ToBoolean()
			if retBool {
				retValue, _ = js.Call(`processRelation`, v)

				// layer
				//layerValue, _ := retValue.Object().Get("layer")
				//layerString, _ := layerValue.ToString()

				// TODO: Do "something" with that data ;-)
			}

		default:
			log.Fatalf("unknown type %T\n", v)
		}
	}
}

func main() {
	var threads = 1
	var qlen = 100

	inChan := make(chan interface{}, qlen)
	storeChan := make(chan feature, qlen)
	exportChan := make(chan tileFeatures, qlen)
	writeChan := make(chan tileData, qlen)

	var wg sync.WaitGroup

	// Launch reader routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		reader(0, "malta-latest.osm.pbf", inChan)
		close(inChan)
	}()

	// Launch processor routines
	for w := 0; w < threads; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			processor(w, inChan, storeChan)
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
				var col = int(ColumnFromLongitudeF(float32(coordinate.longitude), ZOOM))
				var row = int(RowFromLatitudeF(float32(coordinate.latitude), ZOOM))

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

			// Note: Just store the feature in the tile which it FIRST occurs?
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

	for w := 0; w < threads; w++ {
		wg.Add(1)
		wgExporter.Add(1)
		go func(w int) {
			defer wg.Done()
			defer wgExporter.Done()
			exporter(w, exportChan, writeChan)
		}(w)
	}

	// Starter writer routine
	wg.Add(1)
	go func() {
		defer wg.Done()

		meta := metadata{name: "pace", description: "pacetiles", bounds: []float32{
			float32(minLongitude),
			float32(minLatitude),
			float32(maxLongitude),
			float32(maxLatitude)}}

		writer(0, writeChan, "malta.mbtiles", &meta)
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
