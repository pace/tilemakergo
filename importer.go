package main

import (
	"io"
	"log"
	"sort"

	// "sync"
	// "fmt"

	"runtime"
	// "runtime/debug"
)

type osmnode struct {
	id        int64
	longitude float64
	latitude  float64
}

type bounds struct {
	minLatitude  float64
	minLongitude float64
	maxLatitude  float64
	maxLongitude float64
}

/*ImportPbf imports a .pbf file*/
func reader(sourceFile string, results chan<- feature, boundsChan chan<- bounds) {
	var totalNodeCount = 0
	var totalWayCount = 0
	var totalRelationCount = 0

	var bounds = bounds{}

	dec := &decoder{}
	(*dec).readOsmPbf(&sourceFile)

	// We decode the pbf two times. This is due to the fact, that the osm pbf is in a
	// very strange format. To decode a way we need to have access to ALL nodes.
	// Keeping all nodes in memory is very inefficent, so we load all ways first, checking which nodes
	// are required, and then do it again but read all data in and skip unused nodes.

	log.Printf("Extracting required nodes from ways\n")

	// Node cache
	cacheNodeCount := 0
	cacheIndex := 0
	cacheHit := false
	cachedNodePositions := make([]osmnode, cacheNodeCount)

	duplicateCache := make([]int64, 1000)
	duplicateCount := 0

	for {
		features, err := (*dec).read()

		if err == io.EOF {
			break
		}

		for _, v := range features {
			switch v := v.(type) {
			case *Node:
				totalNodeCount++

			case *Way:
				totalWayCount++
				if wayIncluded(&v.Tags) {
					for _, nodeID := range v.NodeIDs {
						cacheHit = false
						// cacheIteratorIndex = 0

						for i := 0; i < len(duplicateCache); i++ {
							if duplicateCache[i] == nodeID {
								duplicateCount++
								cacheHit = true
								break
							}
						}

						if !cacheHit {
							cacheNodeCount++
							duplicateCache[cacheIndex] = nodeID
							cacheIndex = (cacheIndex + 1) % len(duplicateCache)

							cachedNodePositions = append(cachedNodePositions, osmnode{nodeID, 0, 0})
						}
					}
				}

			case *Relation:
				totalRelationCount++

			default:
				continue
			}
		}
	}

	dec = nil
	log.Printf("Preprocessed %d nodes (removed %d duplicates)\n", cacheNodeCount, duplicateCount)

	// log.Println("Filled array with node IDs.")

	log.Println("Sorting node cache")

	sort.Slice(cachedNodePositions, func(i, j int) bool {
		return cachedNodePositions[i].id < cachedNodePositions[j].id
	})

	log.Println("Sorting done")

	// Read in all features
	dec = &decoder{}
	(*dec).readOsmPbf(&sourceFile)

	go func() {
		log.Printf("Decoding %d nodes, %d ways\n", totalNodeCount, totalWayCount)

		var processedNodeProgress = 0
		var processedNodeCount = 0

		var processedWayProgress = 0
		var processedWayCount = 0

		var foundCachedNodeCount = 0

		for {
			features, err := (*dec).read()

			if err == io.EOF {
				for c := range cachedNodePositions {
					if cachedNodePositions[c].latitude != 0 && cachedNodePositions[c].longitude != 0 {
						foundCachedNodeCount++
					}
				}

				log.Printf("Found positions of %d / %d nodes (%f)", foundCachedNodeCount, len(cachedNodePositions), float32(foundCachedNodeCount)/float32(len(cachedNodePositions)))
				log.Printf("OSM PBF file decoded completly")
				log.Printf("Imported area bounds: [%f, %f, %f, %f]\n", bounds.minLatitude, bounds.minLongitude, bounds.maxLatitude, bounds.maxLongitude)

				// Don't need nodes any more.
				cachedNodePositions = nil
				runtime.GC()

				// Close output channel
				close(results)

				// Put meta data into channel
				boundsChan <- bounds
				close(boundsChan)
				break
			}

			for _, v := range features {
				switch v := v.(type) {
				case *Node:
					nodeCoordinate := coordinate{float64(v.Lat), float64(v.Lon)}
					foundNode := setLatLong(cachedNodePositions, v.ID, nodeCoordinate)

					if nodeIncluded(&v.Tags) {
						foundNode = true
						layerString, propertiesInterface := processNode(&v.Tags, v.ID)

						retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{nodeCoordinate}, propertiesInterface}
						results <- retFeature
					}

					if foundNode {
						if bounds.minLatitude == 0 || bounds.minLatitude > nodeCoordinate.latitude {
							bounds.minLatitude = nodeCoordinate.latitude
						}
						if bounds.maxLatitude == 0 || bounds.maxLatitude < nodeCoordinate.latitude {
							bounds.maxLatitude = nodeCoordinate.latitude
						}
						if bounds.minLongitude == 0 || bounds.minLongitude > nodeCoordinate.longitude {
							bounds.minLongitude = nodeCoordinate.longitude
						}
						if bounds.maxLongitude == 0 || bounds.maxLongitude < nodeCoordinate.longitude {
							bounds.maxLongitude = nodeCoordinate.longitude
						}
					}

					processedNodeCount++
					var newProgress = int((float32(processedNodeCount) / float32(totalNodeCount)) * 100)
					if newProgress > processedNodeProgress {
						processedNodeProgress = newProgress

						if processedNodeProgress%10 == 0 {
							log.Printf("Decoded %d pct of nodes\n", processedNodeProgress)
						}
					}
				case *Way:
					if wayIncluded(&v.Tags) {
						layerString, propertiesInterface := processWay(&v.Tags)

						coordinates := make([]coordinate, 0, len(v.NodeIDs))

						for _, nodeID := range v.NodeIDs {
							coordinates = append(coordinates, getLatLong(cachedNodePositions, nodeID))
						}

						retFeature := feature{v.ID, featureTypeLine, layerString, coordinates, propertiesInterface}
						results <- retFeature
					}

					processedWayCount++
					var newProgress = int((float32(processedWayCount) / float32(totalWayCount)) * 100)
					if newProgress > processedWayProgress {
						processedWayProgress = newProgress

						if processedWayProgress%10 == 0 {
							log.Printf("Decoded %d pct of ways\n", processedWayProgress)
						}
					}
				default:
				}
			}
		}
	}()
}
