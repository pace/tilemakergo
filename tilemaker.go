package main

import (
	"fmt"
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

// Access to array:
// tiles[zoomlevel + (row * row_count) + (column * (column_count * row_count))]

var tiles = map[int64]tileFeatures{}
var nodeCoordinates = map[int64]coordinate{}
var nodeCoordinatesSemaphore = make(chan struct{}, 1)

type tileFeatures struct {
	zoomLevel int
	row       int
	column    int
	features  []feature
}

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

				// TODO: Generate properties map

				// create and pass feature
				nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				var coordinates []coordinate
				for _, nodeID := range v.NodeIDs {
					coordinates = append(coordinates, nodeCoordinates[nodeID])
				}
				retFeature := feature{v.ID, featureTypePoint, layerString, coordinates, nil}
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

func exporter(id int, jobs <-chan int, results chan<- int) {}

func writer(id int, jobs <-chan int, destFile string) {}

func main() {
	var threads = 4
	var qlen = 100

	inChan := make(chan interface{}, qlen)
	storeChan := make(chan feature, qlen)
	exportChan := make(chan int)
	writeChan := make(chan int)

	var wg sync.WaitGroup

	// Launch reader routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		reader(0, "karlsruhe-regbez-latest.osm.pbf", inChan)
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

	go func() {
		for v := range storeChan {
			// store somehow
			fmt.Printf("%v\n", v)
		}
	}()

	// Wait until all data is processed (all routines ended)
	wg.Wait()

	// Start exporter routines
	for w := 0; w < threads; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			exporter(w, exportChan, writeChan)
		}(w)
	}

	// Starter writer routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		writer(0, writeChan, "karlsruhe.mbtiles")
	}()

	// Wait until all data is processed (all routines ended)
	wg.Wait()
}
