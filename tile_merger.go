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

	for i, layerA := range pbTileA.Layers {

		// Merge Layer string tables:
		tableA := layerA.Values
		tableB := pbTileB.Layers[i].Values

		tableMap := make(map[int]int)
		tagsToAdd := make([]*Tile_Value, 0, 0)

		for k, v := range tableB {
			found := -1
			for i, a := range tableA {
				if *a.StringValue == *v.StringValue {
					found = i
					break
				}
			}

			if found > -1 {
				tableMap[k] = i
			} else {
				tagsToAdd = append(tagsToAdd, v)
				tableMap[k] = len(tagsToAdd) + len(tableA) - 1
			}
		}

		layerA.Values = append(tableA, tagsToAdd...)

		// Collect feature IDs:
		idMap := make(map[uint64]bool)
		featuresToAdd := make([]*Tile_Feature, 0, len(pbTileB.Layers[i].Features))

		for _, featureA := range layerA.Features {
			idMap[*featureA.Id] = true
		}

		for _, featureB := range pbTileB.Layers[i].Features {
			_, hasKey := idMap[*featureB.Id]
			if !hasKey {
				// Remap tag IDs.
				for i := range featureB.Tags {
					featureB.Tags[i] = uint32(tableMap[int(featureB.Tags[i])])
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
