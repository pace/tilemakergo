package main

import (
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

	// set up the lists
	for {
		if v, err := d.Decode(); err == io.EOF {
			break

		} else if err != nil {
			log.Fatal(err)

		} else {
			node <- v
		}
	}
}
