package main

import (
	"io"
	"log"
	"os"
	"runtime"

	"github.com/qedus/osmpbf"
)

/*ImportPbf imports a .pbf file*/
func reader(sourceFile string) (nodes map[int64]osmpbf.Node, ways []osmpbf.Way, relations []osmpbf.Relation) {
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

	nodes = map[int64]osmpbf.Node{}
	ways = []osmpbf.Way{}
	relations = []osmpbf.Relation{}

	// set up the lists
	for {
		if v, err := d.Decode(); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		} else {
			switch v := v.(type) {
			case *osmpbf.Node:
				nodes[v.ID] = *v
				nodeCount++
			case *osmpbf.Way:
				ways = append(ways, *v)
				wayCount++
			case *osmpbf.Relation:
				relations = append(relations, *v)
				relationCount++
			}
		}
	}

	log.Printf("Finished importing: %d nodes, %d ways and %d relations.\n", nodeCount, wayCount, relationCount)
	return
}

// func processNodes(jobs <-chan interface{}, results chan<- feature) {
// 	for v:= range jobs {
// 		switch v := v.(type) {
// 		case *osmpbf.Node:
// 			// Store coords for later useWay
// 			nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
// 			nodeCoordinates[v.ID] = coordinate{v.Lat, v.Lon}
// 			<-nodeCoordinatesSemaphore // Release

// 			// Process Node v.
// 			if nodeIncluded(v.Tags) {
// 				layerString, propertiesInterface := processNode(v.Tags)

// 				// create and pass feature
// 				// nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
// 				retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{coordinate{v.Lat, v.Lon}}, propertiesInterface}
// 				results <- retFeature
// 				// <-nodeCoordinatesSemaphore // Release
// 			}
// 		}
// 	}
// }

// func processWays(jobs <-chan interface{}, results chan<- feature) {
// 	for v:= range jobs {
// 		switch v := v.(type) {
// 		case *osmpbf.Way:
// 			//	Process Way v.
// 			if wayIncluded(v.Tags) {
// 				layerString, propertiesInterface := processWay(v.Tags)

// 				// create and pass feature
// 				// nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
// 				var coordinates []coordinate
// 				for _, nodeID := range v.NodeIDs {
// 					coordinates = append(coordinates, nodeCoordinates[nodeID])
// 				}
// 				retFeature := feature{v.ID, featureTypeLine, layerString, coordinates, propertiesInterface}
// 				results <- retFeature
// 				// <-nodeCoordinatesSemaphore // Release
// 			}
// 		}
// 	}
// }


// Goes through every node / way / relation and checks:
// - If it should be included
// - In which layer and with which attributes it should be included
func processor(nodes *map[int64]osmpbf.Node, jobs <-chan interface{}, results chan<- feature) {
	for v := range jobs {
		switch v := v.(type) {
		case osmpbf.Node:
			// // Store coords for later useWay
			// nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
			// nodeCoordinates[v.ID] = coordinate{v.Lat, v.Lon}
			// <-nodeCoordinatesSemaphore // Release

			// Process Node v.
			if nodeIncluded(v.Tags) {
				layerString, propertiesInterface := processNode(v.Tags)

				// create and pass feature
				// nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{coordinate{v.Lat, v.Lon}}, propertiesInterface}
				results <- retFeature
				// <-nodeCoordinatesSemaphore // Release
			}

		case osmpbf.Way:
			// Process Way v.
			if wayIncluded(v.Tags) {
				layerString, propertiesInterface := processWay(v.Tags)

				// create and pass feature
				// nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				var coordinates []coordinate
				for _, nodeID := range v.NodeIDs {
					coordinates = append(coordinates, coordinate{(*nodes)[nodeID].Lat, (*nodes)[nodeID].Lon})
				}
				retFeature := feature{v.ID, featureTypeLine, layerString, coordinates, propertiesInterface}
				results <- retFeature
				// <-nodeCoordinatesSemaphore // Release
			}

		case osmpbf.Relation:
			// Process Relation v.
			if relationIncluded(v.Tags) {
				// TODO: Do "something" with that data ;-)
			}

		default:
			log.Fatalf("unknown type %T\n", v)
		}
	}
}