package main

import (
	"log"

	proto "github.com/golang/protobuf/proto"
)

func mergeTiles(tileA []byte, tileB []byte) []byte {

	var pbTileA Tile
	var pbTileB Tile

	proto.Unmarshal(tileA, &pbTileA)
	proto.Unmarshal(tileB, &pbTileB)

	// Add same layers to both tiles.
	for _, layerA := range pbTileA.Layers {
		GetOrCreateLayer(&pbTileB, layerA.Name)
	}

	for _, layerB := range pbTileB.Layers {
		GetOrCreateLayer(&pbTileA, layerB.Name)
	}

	// Add all features from tile B to tile A if they are not yet there:

	for _, layerA := range pbTileA.Layers {

		// Find smae layer in tile B.
		k := 0
		for j := range pbTileB.Layers {
			if *pbTileB.Layers[j].Name == *layerA.Name {
				k = j
				break
			}
		}

		// Merge Layer string tables.

		// Merge key tables:
		keysA := layerA.Keys
		keysB := pbTileB.Layers[k].Keys

		keyMap := make(map[int]int)
		keysToAdd := make([]string, 0, 0)

		for k, b := range keysB {
			found := -1
			for i, a := range keysA {
				if b == a {
					found = i
					break
				}
			}

			if found > -1 {
				keyMap[k] = found
			} else {
				keysToAdd = append(keysToAdd, b)
				keyMap[k] = len(keysToAdd) + len(keysA) - 1
			}
		}

		layerA.Keys = append(keysA, keysToAdd...)

		// Merge value tables:
		valuesA := layerA.Values
		valuesB := pbTileB.Layers[k].Values

		valueMap := make(map[int]int)
		valuesToAdd := make([]*Tile_Value, 0, 0)

		for k, b := range valuesB {
			found := -1

			for i, a := range valuesA {
				if a.StringValue != nil && b.StringValue != nil {
					if *a.StringValue == *b.StringValue {
						found = i
						break
					}
				} else if a.IntValue != nil && b.IntValue != nil {
					if *a.IntValue == *b.IntValue {
						found = i
						break
					}
				}
			}

			if found > -1 {
				valueMap[k] = found
			} else {
				valuesToAdd = append(valuesToAdd, b)
				valueMap[k] = len(valuesToAdd) + len(valuesA) - 1
			}
		}

		layerA.Values = append(valuesA, valuesToAdd...)

		// Collect feature IDs:
		idMap := make(map[uint64]bool)
		featuresToAdd := make([]*Tile_Feature, 0, len(pbTileB.Layers[k].Features))

		for _, featureA := range layerA.Features {
			idMap[*featureA.Id] = true
		}

		for _, featureB := range pbTileB.Layers[k].Features {
			_, hasKey := idMap[*featureB.Id]
			if !hasKey {
				// Remap tag IDs. TODO.
				for i := range featureB.Tags {
					if (i % 2) == 0 {
						// Use keys.
						featureB.Tags[i] = uint32(keyMap[int(featureB.Tags[i])])
					} else {
						// Use Values.
						featureB.Tags[i] = uint32(valueMap[int(featureB.Tags[i])])
					}
				}
				// Add feature to Layer.
				featuresToAdd = append(featuresToAdd, featureB)
			}
		}

		layerA.Features = append(layerA.Features, featuresToAdd...)
	}

	result, err := proto.Marshal(&pbTileA)

	if err != nil {
		log.Fatal("Error marshalling merged tile.")
	}

	return result
}
