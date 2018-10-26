package main

import (
	"io"
	"log"
	"os"
	"runtime"

	"github.com/qedus/osmpbf"
)

/*ImportPbf imports a .pbf file*/
func reader(id int, sourceFile string, node chan<- interface{}) {
	f, err := os.Open(sourceFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	d := osmpbf.NewDecoder(f)

	// use more memory from the start, it is faster
	d.SetBufferSize(osmpbf.MaxBlobSize)

	d.Start(runtime.GOMAXPROCS(-1))

	// Set up our counters
	var nodeCount int64
	var wayCount int64
	var relationCount int64

	// set up the lists
	for {
		if v, err := d.Decode(); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		} else {
			node <- v

			switch v.(type) {
			case *osmpbf.Node:
				nodeCount++
			case *osmpbf.Way:
				wayCount++
			case *osmpbf.Relation:
				relationCount++
			}
		}
	}

	log.Printf("Finished importing: %d nodes, %d ways and %d relations.\n", nodeCount, wayCount, relationCount)
}

// Goes through every node / way / relation and checks:
// - If it should be included
// - In which layer and with which attributes it should be included
func processor(javascript string, jobs <-chan interface{}, results chan<- feature) {
	for v := range jobs {
		switch v := v.(type) {
		case *osmpbf.Node:
			// Store coords for later useWay
			nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
			nodeCoordinates[v.ID] = coordinate{v.Lat, v.Lon}
			<-nodeCoordinatesSemaphore // Release

			// Process Node v.
			if nodeIncluded(v.Tags) {
				layerString, propertiesInterface := processNode(v.Tags)

				// create and pass feature
				nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{nodeCoordinates[v.ID]}, propertiesInterface}
				results <- retFeature
				<-nodeCoordinatesSemaphore // Release
			}

		case *osmpbf.Way:
			// Process Way v.
			if wayIncluded(v.Tags) {
				layerString, propertiesInterface := processWay(v.Tags)

				// create and pass feature
				nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				var coordinates []coordinate
				for _, nodeID := range v.NodeIDs {
					coordinates = append(coordinates, nodeCoordinates[nodeID])
				}
				retFeature := feature{v.ID, featureTypeLine, layerString, coordinates, propertiesInterface}
				results <- retFeature
				<-nodeCoordinatesSemaphore // Release
			}

		case *osmpbf.Relation:
			// Process Relation v.
			if relationIncluded(v.Tags) {
				// TODO: Do "something" with that data ;-)
			}

		default:
			log.Fatalf("unknown type %T\n", v)
		}
	}
}