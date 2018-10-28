package main

import (
	"io"
	"log"
	"sync"
)

type bounds struct {
	minLatitude  float32
	minLongitude float32
	maxLatitude  float32
	maxLongitude float32
}

/*ImportPbf imports a .pbf file*/
func reader(sourceFile string, results chan<- feature, boundsChan chan<- bounds) {
	var wg sync.WaitGroup

	var nodeCount = 0
	var wayCount = 0
	var relationCount = 0

	var bounds = bounds{}

    dec := &decoder{}
	(*dec).readOsmPbf(&sourceFile)

	// We decode the pbf two times. This is due to the fact, that the osm pbf is in a 
	// very strange format. To decode a way we need to have access to ALL nodes.
	// Keeping all nodes in memory is very inefficent, so we load all ways first, checking which nodes
	// are required, and then do it again but read all data in and skip unused nodes.

	var nodeMap = map[int64]coordinate{}

	log.Printf("Extracting required nodes from ways\n")

	for {
		features, err := (*dec).read()

		if err == io.EOF {
			break
		}

		for _, v := range features {
			switch v := v.(type) {
				case *Way:
					if wayIncluded(&v.Tags) {
						for _, nodeID := range v.NodeIDs {
							if _, ok := nodeMap[nodeID]; !ok {
								nodeMap[nodeID] = coordinate{}
							}
						}
					}

				default: continue
			}
		}
	}

	log.Printf("\nDone. %d nodes required for ways in this extract\n", len(nodeMap))

	// Read in all features
	dec = &decoder{}
	(*dec).readOsmPbf(&sourceFile)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			features, err := (*dec).read()

			if err == io.EOF {
				log.Printf("OSM PBF file decoded completly")
				break
			}

			for _, v := range features {
				switch v := v.(type) {
					case *Node:
						nodeCoordinate := coordinate{float32(v.Lat), float32(v.Lon)}
						
						nodeMap[v.ID] = nodeCoordinate

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

						if nodeIncluded(&v.Tags) {
							layerString, propertiesInterface := processNode(&v.Tags)

							retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{nodeCoordinate}, propertiesInterface}
							results <- retFeature

							nodeCount++
						}
					case *Way:
						if wayIncluded(&v.Tags) {
							layerString, propertiesInterface := processWay(&v.Tags)

							var coordinates []coordinate

							for _, nodeID := range v.NodeIDs {
								coordinates = append(coordinates, nodeMap[nodeID])
							}

							retFeature := feature{v.ID, featureTypeLine, layerString, coordinates, propertiesInterface}
							results <- retFeature

							wayCount++
						}
					case *Relation:
						if relationIncluded(&v.Tags) {
							relationCount++
						}
					default: 
						log.Printf("%s", v)
				}
			}
		}
	}()

	go func() {
		// Wait for all readers to be finished
		wg.Wait()

		// Close output channel
		close(results)
		
		// Put meta data into channel
		boundsChan <- bounds
		close(boundsChan)

		log.Printf("Finished importing: %d nodes, %d ways and %d relations.\n", nodeCount, wayCount, relationCount)
		log.Printf("Imported area bounds: [%f, %f, %f, %f]\n", bounds.minLatitude, bounds.minLongitude, bounds.maxLatitude, bounds.maxLongitude)
	}()
}