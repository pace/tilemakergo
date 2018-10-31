package main

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	proto "github.com/golang/protobuf/proto"
)

func TestTileMerger(t *testing.T) {
	tagsAofRoads := []string{"rf"}
	b10 := "B10"
	b10value := Tile_Value{StringValue: &b10}
	valuesAofRoads := []*Tile_Value{&b10value}

	tagsAofAddresses := []string{"hn", "st"}

	hn39 := int64(39)
	value39 := Tile_Value{IntValue: &hn39}
	mainzerStr := "Mainzer Straße"
	valueMainzer := Tile_Value{StringValue: &mainzerStr}

	valuesAofAddresses := []*Tile_Value{&value39, &valueMainzer}

	tagsBofRoads := []string{"ab", "rf"}
	b10b := "B10"
	b10valueB := Tile_Value{StringValue: &b10b}
	docks := "dock"
	docksValue := Tile_Value{StringValue: &docks}
	valuesBofRoads := []*Tile_Value{&b10valueB, &docksValue}

	tagsBofTraffic := []string{"a"}
	vB := "b"
	vC := "c"
	value1 := Tile_Value{StringValue: &vB}
	value2 := Tile_Value{StringValue: &vC}
	valuesBofTraffic := []*Tile_Value{&value1, &value2}

	id100 := uint64(100)
	id101 := uint64(101)
	id101B := uint64(101)
	id103 := uint64(103)
	id200 := uint64(200)
	id300 := uint64(300)

	feature100A := Tile_Feature{Id: &id100, Geometry: []uint32{1, 2, 3}, Tags: []uint32{0, 0}}
	feature101A := Tile_Feature{Id: &id101, Geometry: []uint32{4, 5}, Tags: []uint32{0, 0}}
	feature200A := Tile_Feature{Id: &id200, Geometry: []uint32{1}, Tags: []uint32{0, 0, 1, 1}}

	feature101B := Tile_Feature{Id: &id101B, Geometry: []uint32{4, 5}, Tags: []uint32{1, 0}}
	feature103B := Tile_Feature{Id: &id103, Geometry: []uint32{6, 7}, Tags: []uint32{1, 0, 0, 1}}
	feature300B := Tile_Feature{Id: &id300, Geometry: []uint32{6, 7}, Tags: []uint32{0, 0, 0, 1}}

	roadNameA := "roads"
	roadNameB := "roads"
	addrNameA := "addresses"
	trafficNameB := "traffic"

	layerVersion := uint32(1)

	layerARoads := Tile_Layer{Name: &roadNameA, Features: []*Tile_Feature{&feature100A, &feature101A}, Keys: tagsAofRoads, Values: valuesAofRoads, Version: &layerVersion}
	layerAAddresses := Tile_Layer{Name: &addrNameA, Features: []*Tile_Feature{&feature200A}, Keys: tagsAofAddresses, Values: valuesAofAddresses, Version: &layerVersion}

	layerBRoads := Tile_Layer{Name: &roadNameB, Features: []*Tile_Feature{&feature101B, &feature103B}, Keys: tagsBofRoads, Values: valuesBofRoads, Version: &layerVersion}
	layerBtraffic := Tile_Layer{Name: &trafficNameB, Features: []*Tile_Feature{&feature300B}, Keys: tagsBofTraffic, Values: valuesBofTraffic, Version: &layerVersion}

	tileA := Tile{Layers: []*Tile_Layer{&layerARoads, &layerAAddresses}}
	tileB := Tile{Layers: []*Tile_Layer{&layerBRoads, &layerBtraffic}}

	packedA, err := proto.Marshal(&tileA)

	if err != nil {
		failTest(fmt.Sprintf("Failed to marshal tile A: %s", err), t)
	}

	packedB, err := proto.Marshal(&tileB)

	if err != nil {
		failTest(fmt.Sprintf("Failed to marshal tile B: %s", err), t)
	}

	packedMerge := mergeTiles(packedA, packedB)

	var merged Tile

	err = proto.Unmarshal(packedMerge, &merged)

	if err != nil {
		failTest(fmt.Sprintf("Failed to unmarshal merged tile: %s", err), t)
	}

	// Verify content of merged tile:

	gotRoads := false
	gotAddr := false
	gotTraffic := false

	for _, layer := range merged.Layers {
		if *(*layer).Name == "roads" {
			gotRoads = true

			if len(layer.Features) != 3 {
				failTest(fmt.Sprintf("Expected 3 features in roads layer but got %d", len(layer.Features)), t)
			}

			verifyFeature(&feature100A, findFeature(layer.Features, id100), id100, []string{"rf", "B10"}, layer.Keys, layer.Values, t)
			verifyFeature(&feature101A, findFeature(layer.Features, id101), id101, []string{"rf", "B10"}, layer.Keys, layer.Values, t)
			verifyFeature(&feature103B, findFeature(layer.Features, id103), id103, []string{"rf", "B10", "ab", "dock"}, layer.Keys, layer.Values, t)
		} else if *(*layer).Name == "addresses" {
			gotAddr = true

			if len(layer.Features) != 1 {
				failTest(fmt.Sprintf("Expected 1 feature in addresses layer but got %d", len(layer.Features)), t)
			}

			verifyFeature(&feature200A, findFeature(layer.Features, id200), id200, []string{"hn", "39", "st", "Mainzer Straße"}, layer.Keys, layer.Values, t)
		} else if *(*layer).Name == "traffic" {
			gotTraffic = true

			if len(layer.Features) != 1 {
				failTest(fmt.Sprintf("Expected 1 feature in traffic layer but got %d", len(layer.Features)), t)
			}

			verifyFeature(&feature300B, findFeature(layer.Features, id300), id300, []string{"a", "b", "a", "c"}, layer.Keys, layer.Values, t)
		} else {
			failTest(fmt.Sprintf("Unknown layer %s", *(*layer).Name), t)
		}
	}

	if !gotRoads {
		failTest("Missing roads layer", t)
	}

	if !gotAddr {
		failTest("Missing addresses layer", t)
	}

	if !gotTraffic {
		failTest("Missing traffic layer", t)
	}
}

func findFeature(features []*Tile_Feature, id uint64) *Tile_Feature {
	for _, f := range features {
		if *(*f).Id == id {
			return f
		}
	}
	return nil
}

func featureEquals(a *Tile_Feature, b *Tile_Feature) bool {

	if (*a.Id) != (*b.Id) {
		fmt.Printf("Expected feature id %d but got %d\n", (*a.Id), (*b.Id))
		return false
	}

	if len(a.Geometry) != len(b.Geometry) {
		fmt.Printf("Expected %d geometry elements but got %d\n", len(a.Geometry), len(b.Geometry))
		return false
	}

	for i, geomA := range a.Geometry {
		geomB := b.Geometry[i]

		if geomA != geomB {
			fmt.Printf("Expected geometry command %d but got %d\n", geomA, geomB)
			return false
		}
	}

	return true
}

func getFeatureTags(a *Tile_Feature, keys []string, values []*Tile_Value) []string {
	result := make([]string, len(a.Tags))

	for i, t := range a.Tags {
		if (i % 2) == 0 {
			// Use keys.
			result[i] = keys[t]
		} else {
			// Use values.
			if values[t].StringValue != nil {
				result[i] = *(values[t].StringValue)
			} else if values[t].IntValue != nil {
				result[i] = strconv.Itoa(int(*(values[t].IntValue)))
			}
		}

	}

	return result
}

func compareFeatureTags(expected []string, got []string) bool {
	if len(expected) != len(got) {
		fmt.Printf("Expected %d elements in tags but got %d\n", len(expected), len(got))
		return false
	}

	for i, e := range expected {
		if e != got[i] {
			fmt.Printf("Expected %s but got %s\n", e, got[i])
			return false
		}
	}

	return true
}

func verifyFeature(expected *Tile_Feature, got *Tile_Feature, id uint64, expectedTags []string, keys []string, values []*Tile_Value, t *testing.T) {
	if got == nil {
		failTest(fmt.Sprintf("Could not find feature with id %d", id), t)
	} else {
		if !featureEquals(expected, got) {
			failTest(fmt.Sprintf("Feature %d not as expected", id), t)
		}

		gotTags := getFeatureTags(got, keys, values)

		if !compareFeatureTags(expectedTags, gotTags) {
			failTest(fmt.Sprintf("Feature %d got different tags as expected", id), t)
		}
	}
}

func failTest(message string, t *testing.T) {
	fmt.Printf("%s\n", message)
	t.Fail()
	os.Exit(1)
}
