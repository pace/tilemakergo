package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/qedus/osmpbf"
)

const (
	//Path of the .pbf file to process
	Path = "hamburg-latest.osm.pbf"
)

func main() {
	fmt.Printf("Process '%s'...\n", Path)
	ImportPbf(Path)
}

/*ImportPbf imports a .pbf file*/
func ImportPbf(path string) {
	var start = time.Now()
	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	d := osmpbf.NewDecoder(f)

	// use more memory from the start, it is faster
	d.SetBufferSize(osmpbf.MaxBlobSize)

	// start decoding with several goroutines, it is faster
	err = d.Start(runtime.GOMAXPROCS(-1))
	if err != nil {
		log.Fatal(err)
	}

	// set up the lists
	var nodes = make([]*osmpbf.Node, 0)
	var ways = make([]*osmpbf.Way, 0)

	var nc, wc int64
	for {
		if v, err := d.Decode(); err == io.EOF {
			break

		} else if err != nil {
			log.Fatal(err)

		} else {
			switch v := v.(type) {
			case *osmpbf.Node:
				// Process Node v.
				nc++
				nodes = append(nodes, v)

			case *osmpbf.Way:
				// Process Way v.
				wc++
				ways = append(ways, v)

			case *osmpbf.Relation:
				// Ignore relations

			default:
				log.Fatalf("unknown type %T\n", v)
			}
		}
	}
	elapsed := time.Now().Sub(start)

	fmt.Printf("Processed %d nodes and %d ways in %s\n", nc, wc, elapsed)

	var way = ways[0]
	fmt.Println("Way:", way)

	for _, node := range nodes {
		if contains(way.NodeIDs, node.ID) {
			fmt.Println("Node:", node)
		}
	}
}

func contains(s []int64, e int64) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
