package main

import (
	"fmt"
	"io"
	"log"
	"os"

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

	d.Start(1)

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

	fmt.Printf("Processed %d nodes, %d ways and %d relations.\n", nodeCount, wayCount, relationCount)
}


// Goes through every node / way / relation and checks:
// - If it should be included
// - In which layer and with which attributes it should be included
func processor(javascript string, jobs <-chan interface{}, results chan<- feature) {
	var js = new(JavascriptEngine)
	js.Load(javascript)

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
				processedNode, _ := js.Call(`processNode`, v)

				// layer
				layerValue, _ := processedNode.Object().Get("layer")
				layerString, _ := layerValue.ToString()

				// The js had the chance to check / modify / copy the tags associated to 
				// the way and return a KV set of values itself to encode
				processedValues, _ := processedNode.Object().Get("properties")
				exportedProperties, err := processedValues.Export()
				propertiesInterface := make(map[string]interface{})

				if (err == nil) {
					propertiesInterface = exportedProperties.(map[string]interface{})
				}

				// create and pass feature
				nodeCoordinatesSemaphore <- struct{}{} // Acquire semaphore token
				retFeature := feature{v.ID, featureTypePoint, layerString, []coordinate{nodeCoordinates[v.ID]}, propertiesInterface}
				results <- retFeature
				<-nodeCoordinatesSemaphore // Release
			}

		case *osmpbf.Way:
			// Process Way v.
			retValue, _ := js.Call(`useWay`, v)
			retBool, _ := retValue.ToBoolean()
			if retBool {
				processedWay, _ := js.Call(`processWay`, v)

				// The layer of the way was determined by the js
				layerValue, _ := processedWay.Object().Get("layer")
				layerString, _ := layerValue.ToString()

				// The js had the chance to check / modify / copy the tags associated to 
				// the way and return a KV set of values itself to encode
				processedValues, _ := processedWay.Object().Get("properties")
				exportedProperties, err := processedValues.Export()
				propertiesInterface := make(map[string]interface{})

				if (err == nil) {
					propertiesInterface = exportedProperties.(map[string]interface{})
				}

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