package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
)

func TestSetAndRetrieve(t *testing.T) {
	osmnodes := make([]osmnode, 5)

	osmnodes[0] = osmnode{int64(1000), float32(1.0), float32(1.5)}
	osmnodes[1] = osmnode{int64(2000), float32(1.0), float32(1.5)}
	osmnodes[2] = osmnode{int64(3000), float32(1.0), float32(1.5)}
	osmnodes[3] = osmnode{int64(4000), float32(1.0), float32(1.5)}
	osmnodes[4] = osmnode{int64(5000), float32(1.0), float32(1.5)}

	setLatLong(osmnodes, int64(3000), coordinate{5.0, 5.5})

	coord := getLatLong(osmnodes, int64(3000))

	epsilon := 0.000001

	if (math.Abs(float64(coord.latitude)-5.0) > epsilon) || (math.Abs(float64(coord.longitude)-5.5) > epsilon) {
		fmt.Printf("Expected coordinate to be set, but was %f %f instead", coord.latitude, coord.longitude)
		t.Fail()
	}

}

func TestSetAndRetrieveLarge(t *testing.T) {
	osmnodes := make([]osmnode, 1000)

	nodeMap := map[int64]coordinate{}

	for i := range osmnodes {
		c := coordinate{rand.Float32(), rand.Float32()}
		id := int64(rand.Int())
		nodeMap[id] = c
		osmnodes[i] = osmnode{id, 0.0, 0.0}
	}

	sort.Slice(osmnodes, func(i, j int) bool {
		return osmnodes[i].id < osmnodes[j].id
	})

	for id, coord := range nodeMap {
		setLatLong(osmnodes, id, coord)
	}

	epsilon := 0.000001

	for id, coord := range nodeMap {
		c := getLatLong(osmnodes, id)

		if (math.Abs(float64(coord.latitude)-float64(c.latitude)) > epsilon) || (math.Abs(float64(coord.longitude)-float64(c.longitude)) > epsilon) {
			fmt.Printf("Expected coordinate to be set to %f %f , but was %f %f instead", coord.latitude, coord.longitude, c.latitude, c.longitude)
			t.Fail()
			break
		}
	}
}
