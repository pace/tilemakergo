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
