package main

import (
	"io"
	"log"
	"sync"
)

type bounds struct {
	minLatitude  float64
	minLongitude float64
	maxLatitude  float64
	maxLongitude float64
}

/*ImportPbf imports a .pbf file*/
func reader(sourceFile string, results chan<- feature, boundsChan chan<- bounds) {
	var wg sync.WaitGroup

	var nodeCount = 0
	var wayCount = 0
	var relationCount = 0

	var bounds = bounds{}

    decoder := &decoder{}
	(*decoder).readOsmPbf(&sourceFile)

	var nodeMap = map[int64]coordinate{}
	// var sm sync.Map

	threads := 1
	for i := 0; i < threads; i++ { 
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {

				features, err := (*decoder).read()

				if err == io.EOF {
					log.Printf("All objects read")
					break
				}

				for _, v := range features {
					switch v := v.(type) {
						case *Node:
							nodeCoordinate := coordinate{v.Lat, v.Lon}

							// sm.Store((*v).ID, coordinate{(*v).Lat, (*v).Lon})
							nodeMap[(*v).ID] = coordinate{(*v).Lat, (*v).Lon}

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
								layerString, propertiesInterface := processNode(v.Tags)

								retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{nodeCoordinate}, propertiesInterface}
								results <- retFeature

								nodeCount++
							}
						case *Way:
							if wayIncluded(&v.Tags) {
								layerString, propertiesInterface := processWay(v.Tags)

								var coordinates []coordinate
								for _, nodeID := range v.NodeIDs {
									node, ok := nodeMap[nodeID]
									// node, ok := sm.Load(nodeID)

									if !ok {
										log.Printf("Node not found: %d", nodeID)
									}

									coordinates = append(coordinates, node)
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
	}

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