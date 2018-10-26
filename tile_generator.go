package main

import (
	"log"

	proto "github.com/golang/protobuf/proto"
)

type CommandId uint8

const (
	commandMoveTo    = uint8(1)
	commandLineTo    = uint8(2)
	commandClosePath = uint8(7)
)
const extent uint32 = 4096 // TODO: This needs to be read from config

var currentX = float64(0.0)
var currentY = float64(0.0)

type layerMeta struct {
	keyIndex   uint32
	valueIndex uint32
	keys       map[string]uint32
	values     map[interface{}]uint32
}

var count = 0

// Debug entry point
func exporter(id int, jobs <-chan tileFeatures, results chan<- tileData) {
	for features := range jobs {
		results <- EncodeFeatures(&features)
	}
}

func EncodeFeatures(tile *tileFeatures) tileData {
	// Create a protobuffer tile file
	var pbTile = Tile{}
	var layerMetas = make(map[string]layerMeta)

	var c = 0
	for _, feature := range tile.features {
		c++
		var pbLayer = GetOrCreateLayer(&pbTile, feature.layer)
		var currentMeta layerMeta

		// Check if there is already a supporting layer meta
		if meta, ok := layerMetas[*feature.layer]; ok {
			currentMeta = meta
		} else {
			currentMeta = layerMeta{0, 0, make(map[string]uint32), make(map[interface{}]uint32)}
		}

		var pbFeature = Tile_Feature{}
		id := uint64(feature.id)
		typ := Tile_GeomType(feature.typ)
		pbFeature.Id = &id
		pbFeature.Type = &typ
		row := uint32(tile.row)
		column := uint32(tile.column)

		// Encode all commands needed to draw this feature
		// Reset the pointer to allow correct relative drawing
		currentX = 0.0
		currentY = 0.0

		// TODO: Split up feature so that we only ever draw 1 "pixel" out of our bounds
		// TODO: We must make sure that a feature is not just cut of but split, in case it enters
		// TODO: the current tile again

		var commands []uint32
		switch feature.typ {
		case featureTypePoint:
			commands = EncodeNode(row, column, tile.zoomLevel, feature)
		case featureTypeLine:
			commands = EncodeWay(row, column, tile.zoomLevel, feature)
		case featureTypePolygon:
			commands = EncodePolygon(row, column, tile.zoomLevel, feature)
		}

		if len(commands) > 0 {
			pbFeature.Geometry = commands

			// Encode all keys (properties) for this feature.
			// NOTE: Multiple features can reference the same key / value.
			// Process:
			// If a key (or value) is not yet in this tile, append it and reference it in this feature
			// If a key (or value) exists in this tile, only reference it
			for key, value := range feature.properties {
				if _, ok := currentMeta.keys[key]; ok {
					pbFeature.Tags = append(pbFeature.Tags, currentMeta.keys[key])
				} else {
					pbFeature.Tags = append(pbFeature.Tags, currentMeta.keyIndex)
					currentMeta.keys[key] = currentMeta.keyIndex
					currentMeta.keyIndex++
				}

				if _, ok := currentMeta.values[value]; ok {
					pbFeature.Tags = append(pbFeature.Tags, currentMeta.values[value])
				} else {
					pbFeature.Tags = append(pbFeature.Tags, currentMeta.valueIndex)
					currentMeta.values[value] = currentMeta.valueIndex
					currentMeta.valueIndex++
				}
			}

			// Variant type encoding
			// The use of values is described in section 4.1 of the specification
			// type Tile_Value struct {
			//  // Exactly one of these values must be present in a valid message
			//  StringValue                  *string  `protobuf:"bytes,1,opt,name=string_value,json=stringValue" json:"string_value,omitempty"`
			//  FloatValue                   *float32 `protobuf:"fixed32,2,opt,name=float_value,json=floatValue" json:"float_value,omitempty"`
			//  DoubleValue                  *float64 `protobuf:"fixed64,3,opt,name=double_value,json=doubleValue" json:"double_value,omitempty"`
			//  IntValue                     *int64   `protobuf:"varint,4,opt,name=int_value,json=intValue" json:"int_value,omitempty"`
			//  UintValue                    *uint64  `protobuf:"varint,5,opt,name=uint_value,json=uintValue" json:"uint_value,omitempty"`
			//  SintValue                    *int64   `protobuf:"zigzag64,6,opt,name=sint_value,json=sintValue" json:"sint_value,omitempty"`
			//  BoolValue                    *bool    `protobuf:"varint,7,opt,name=bool_value,json=boolValue" json:"bool_value,omitempty"`
			//  proto.XXX_InternalExtensions `json:"-"`
			//  XXX_unrecognized             []byte `json:"-"`
			// }
			// Append features to the layers feature
			pbLayer.Features = append(pbLayer.Features, &pbFeature)

			layerMetas[*feature.layer] = currentMeta
		}
	}

	for name, meta := range layerMetas {
		var pbLayer = GetLayer(&pbTile, name)

		// Add all keys in order
		pbKeys := make([]string, len(meta.keys))
		for i, k := range meta.keys {
			pbKeys[k] = i
		}
		pbLayer.Keys = pbKeys

		// Sort all values based on their index
		pbValues := make([]*Tile_Value, len(meta.values))

		for value, index := range meta.values {
			tileValue := Tile_Value{}

			switch v := value.(type) {
			case string:
				tileValue.StringValue = &v
			}

			pbValues[index] = &tileValue
		}

		pbLayer.Values = pbValues
	}

	// Write the protobuffer tile file to the database
	out, err := proto.Marshal(&pbTile)
	if err != nil {
		log.Fatal("Could not export pbf files")
	}

	return tileData{zoomLevel: tile.zoomLevel, row: tile.row, column: tile.column, data: out}
}
func EncodeNode(tileRow uint32, tileColumn uint32, zoom int, node feature) []uint32 {
	// A node consists of a single moveTo command. This can be repeated for multipoints.
	return Command(commandMoveTo, tileRow, tileColumn, zoom, node.coordinates[0:1])
}
func EncodeWay(tileRow uint32, tileColumn uint32, zoom int, way feature) []uint32 {
	// A way consists of a initial moveTo command followed by one or more lineTo command.
	// For a way, skip parts which are outside the tile.
	return CutCommand(tileRow, tileColumn, zoom, way.coordinates[0:len(way.coordinates)])
}
func EncodePolygon(tileRow uint32, tileColumn uint32, zoom int, polygon feature) []uint32 {
	// A way consist of a initial moveTo command followed by one or more lineTo command and a closePath command.
	return append(
		EncodeWay(tileRow, tileColumn, zoom, polygon),
		Command(commandClosePath, tileRow, tileColumn, zoom, []coordinate{})...)
}

// Remove coordinates which are outside of the tile, cut in multiple lists and process as Command
func CutCommand(tileRow uint32, tileColumn uint32, zoom int, coordinates []coordinate) []uint32 {
	if len(coordinates) == 0 {
		return make([]uint32, 0, 0)
	}
	insideTile := false
	didEnterTile := false

	combinedCommands := make([]uint32, 0, len(coordinates)*2+1)

	currentCoordinates := make([]coordinate, 0, len(coordinates))

	for index, coord := range coordinates {
		tileX := uint32(ColumnFromLongitude(float64(coord.longitude), zoom))
		tileY := uint32(RowFromLatitude(float64(coord.latitude), zoom))

		if (tileX == tileColumn) && (tileY == tileRow) {
			didEnterTile = true
			// Was outside? -> restart.
			if insideTile == false {
				currentCoordinates = make([]coordinate, 0, len(coordinates)-index)

				// Use previous coordinate as a starting point if possible:
				if index > 0 {
					currentCoordinates = append(currentCoordinates, coordinates[index-1])
				}
			}

			// Add current coordinate.
			currentCoordinates = append(currentCoordinates, coord)

			insideTile = true
		} else {
			// Was inside? -> draw.
			if insideTile {

				// Add last coordinate to draw outside of bounds.
				currentCoordinates = append(currentCoordinates, coord)

				combinedCommands = append(combinedCommands, Command(commandMoveTo, tileRow, tileColumn, zoom, currentCoordinates[0:1])...)
				if len(currentCoordinates) > 1 {
					combinedCommands = append(combinedCommands, Command(commandLineTo, tileRow, tileColumn, zoom, currentCoordinates[1:len(currentCoordinates)])...)
				}
			}

			insideTile = false
		}
	}

	// Return empty command if we never entered the tile.
	if !didEnterTile {
		return make([]uint32, 0, 0)
	}

	if insideTile {
		// Encode last slice.
		combinedCommands = append(combinedCommands, Command(commandMoveTo, tileRow, tileColumn, zoom, currentCoordinates[0:1])...)
		if len(currentCoordinates) > 1 {
			combinedCommands = append(combinedCommands, Command(commandLineTo, tileRow, tileColumn, zoom, currentCoordinates[1:len(currentCoordinates)])...)
		}
	}

	return combinedCommands
}

func Command(id uint8, tileRow uint32, tileColumn uint32, zoom int, coordinates []coordinate) []uint32 {
	command := make([]uint32, len(coordinates)*2+1)
	command[0] = uint32(uint32(id&0x7)) | uint32((len(coordinates) << 3))

	for index, coordinate := range coordinates {
		// We have the TILE coordinates stored in the feature itself.
		// We now need a offset to this coordinates and multiply that by the tiles pixels resolution
		x := (ColumnFromLongitudeF(float64(coordinate.longitude), zoom) - float64(tileColumn)) * float64(extent)
		y := (RowFromLatitudeF(float64(coordinate.latitude), zoom) - float64(tileRow)) * float64(extent)

		dX := -currentX + x
		dY := -currentY + y

		command[(index*2)+1] = uint32((int64(dX) << 1) ^ (int64(dX) >> 31)) // Longitude
		command[(index*2)+2] = uint32((int64(dY) << 1) ^ (int64(dY) >> 31)) // Latitude

		currentX = x
		currentY = y
	}

	return command
}

/* Protobuffer helper */
func GetOrCreateLayer(tile *Tile, name *string) *Tile_Layer {
	// Check if this tile already contains the layer and if not create it
	for _, layer := range tile.Layers {
		if *layer.Name == *name {
			return layer
		}
	}

	newName := name
	version := uint32(2)
	ex := extent
	layer := Tile_Layer{}
	layer.Features = make([]*Tile_Feature, 0)
	layer.Extent = &ex
	layer.Name = newName
	layer.Version = &version
	tile.Layers = append(
		tile.Layers,
		&layer)

	return &layer
}

func GetLayer(tile *Tile, name string) *Tile_Layer {
	for _, layer := range tile.Layers {
		if *layer.Name == name {
			return layer
		}
	}

	return nil
}
