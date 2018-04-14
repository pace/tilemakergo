package main

import (
   proto "github.com/golang/protobuf/proto"
   "log"
)

const (  
	featureTypeUnknown = iota
    featureTypePoint
    featureTypeLine
    featureTypePolygon
)

type CommandId uint8

const (
	commandMoveTo = uint8(1)
	commandLineTo = uint8(2)
	commandClosePath = uint8(7)
)

const extent uint32 = 4096 // TODO: This needs to be read from config

// Access to array:
// tiles[zoomlevel + (row * row_count) + (column * (column_count * row_count))]

type tileFeatures struct {
	zoomLevel int
    row  int
    column int
    features []feature
}

 type feature struct {
 	id int64
    typ uint8
    coordinates []coordinate
    layer string
    properties map[string]interface{}
}

type coordinate struct {
	latitude float32 
	longitude float32
}

// Debug entry point
func main() {
	props := make(map[string]interface{})
	coords := []coordinate{ coordinate{10.0, 10.0} }

	tf := make(map[int64]tileFeatures)
	tf[16 * 0 * 0] = tileFeatures{
			zoomLevel: 16,
			row: 0,
			column: 0,
			features: []feature{
				feature{
					typ: featureTypePoint,
					coordinates: coords,
					layer: "road",
					properties: props}}}

	meta := metadata{name: "pace", description: "pacetiles", bounds: []float32{
				0.0, 
				10.0, 
				10.0,
				0.0}}

	EncodeFeatures(&tf, &meta)
}

func EncodeFeatures(tiles *map[int64]tileFeatures, meta *metadata) {
	for _, tile := range *tiles {

		// Create a protobuffer tile file
		var pbTile = Tile{}
		var keyIndex = uint32(0)
		var valueIndex = uint32(0)
		var keys = make(map[string]uint32)
		var stringValues = make(map[string]uint32)
		var doubleValues = make(map[string]uint32)
		var intValues = make(map[string]uint32)
		var boolValues = make(map[bool]uint32)


		for _, feature := range tile.features {

			var pbLayer = GetOrCreateLayer(&pbTile, &feature.layer)
   			var pbFeature = Tile_Feature{}

   			id := uint64(feature.id)
   			typ := Tile_GeomType(feature.typ)

			pbFeature.Id = &id
        	pbFeature.Type = &typ

        	row := uint32(tile.row)
        	column := uint32(tile.column)

			// Encode all commands needed to draw this feature
			var commands []uint32
			switch (feature.typ) {
			case featureTypePoint:
				commands = EncodeNode(row, column, tile.zoomLevel, feature)
			case featureTypeLine:
				commands = EncodeWay(row, column, tile.zoomLevel, feature)
			case featureTypePolygon:
				commands = EncodePolygon(row, column, tile.zoomLevel, feature)
			}

			pbFeature.Geometry = commands

			// Encode all keys (properties) for this feature. 
			// NOTE: Multiple features can reference the same key / value.
			// Process:
			// If a key (or value) is not yet in this tile, append it and reference it in this feature
			// If a key (or value) exists in this tile, only reference it
			for key, value := range feature.properties {
				if keys[key] {
					feature.Tags = append(feature.Tags, keys[key])
				} else {
					keys[key] = keyIndex++
				}

				switch v := value.(type) {
				case string:
					if stringValues[value] {
						feature.Tags = append(feature.Tags, stringValues[value])
					} else {
						stringValues[value] = valueIndex++
					}
				}
			}

			// Variant type encoding
			// The use of values is described in section 4.1 of the specification
			// type Tile_Value struct {
			// 	// Exactly one of these values must be present in a valid message
			// 	StringValue                  *string  `protobuf:"bytes,1,opt,name=string_value,json=stringValue" json:"string_value,omitempty"`
			// 	FloatValue                   *float32 `protobuf:"fixed32,2,opt,name=float_value,json=floatValue" json:"float_value,omitempty"`
			// 	DoubleValue                  *float64 `protobuf:"fixed64,3,opt,name=double_value,json=doubleValue" json:"double_value,omitempty"`
			// 	IntValue                     *int64   `protobuf:"varint,4,opt,name=int_value,json=intValue" json:"int_value,omitempty"`
			// 	UintValue                    *uint64  `protobuf:"varint,5,opt,name=uint_value,json=uintValue" json:"uint_value,omitempty"`
			// 	SintValue                    *int64   `protobuf:"zigzag64,6,opt,name=sint_value,json=sintValue" json:"sint_value,omitempty"`
			// 	BoolValue                    *bool    `protobuf:"varint,7,opt,name=bool_value,json=boolValue" json:"bool_value,omitempty"`
			// 	proto.XXX_InternalExtensions `json:"-"`
			// 	XXX_unrecognized             []byte `json:"-"`
			// }

			// Append features to the layers feature
			pbLayer.Features = append(pbLayer.Features, &pbFeature)
		}

		// Add all keys in order
		pbKeys := make([]string, len(keys))

		for i, k := range keys {
			pbKeys[i] = k
		}

		pbTile.Keys = &pbKeys

		pbValues := make([]Tile_Value, len(stringValues) + len(intValues) + len(doubleValues) + len(boolValues))

		for i, v := range stringValues {
			tileValue := Tile_Value{}
			tileValue.StringValue = v
			pbValues[i] = tileValue
		}

		pbTile.Values = &pbValues

		// Write the protobuffer tile file to the database
		out, err := proto.Marshal(&pbTile)

		if err != nil {
			log.Fatal("Could not export pbf files")
		}

		// TODO: Do not open the database every time
		ExportTiles("export.mbtiles", []tileData{tileData{zoomLevel: tile.zoomLevel, row: tile.row, column: tile.column, data: out}}, meta)
	}
}

func EncodeNode(tileRow uint32, tileColumn uint32, zoom int, node feature) []uint32 {
	// A node consists of a single moveTo command. This can be repeated for multipoints.
	return Command(commandMoveTo, 1, tileRow, tileColumn, zoom, node.coordinates[0:0])
}

func EncodeWay(tileRow uint32, tileColumn uint32, zoom int, way feature) []uint32 {
	lineToCount := uint8((len(way.coordinates) / 2) - 1)

	// A way consist of a initial moveTo command followed by one or more lineTo command. 
	return append(
		Command(commandMoveTo, 1, tileRow, tileColumn, zoom, way.coordinates[0:0]),
		Command(commandLineTo, lineToCount, tileRow, tileColumn, zoom, way.coordinates[1:len(way.coordinates)-2])...)
}

func EncodePolygon(tileRow uint32, tileColumn uint32, zoom int, polygon feature) []uint32 {
	// A way consist of a initial moveTo command followed by one or more lineTo command and a closePath command. 
	return append(
		EncodeWay(tileRow, tileColumn, zoom, polygon),
		Command(commandClosePath, 1, tileRow, tileColumn, zoom, []coordinate{})...)
}

func Command(id uint8, count uint8, tileRow uint32, tileColumn uint32, zoom int, coordinates []coordinate) []uint32 {
	command := make([]uint32, count + 1)
	command[0] = uint32(uint32(id & 0x7)) | uint32((count << 3))

	currentX := uint32(0)
	currentY := uint32(0)

	for index, coordinate := range coordinates {
		// We have the TILE coordinates stored in the feature itself. 
		// We now need a offset to this coordinates and multiply that by the tiles pixels resolution
		x := uint32((ColumnFromLongitudeF(coordinate.longitude, zoom) - float32(tileColumn)) * float32(extent))
		y := uint32((RowFromLatitudeF(coordinate.latitude, zoom) - float32(tileRow)) * float32(extent))

		dX := -currentX + x
		dY := -currentY + y

		command[(index * 2) + 2] = uint32(dX << 1) ^ uint32(dX >> 31) // Longitude
		command[(index * 2) + 1] = uint32(dY << 1) ^ uint32(dY >> 31) // Latitude

		currentX = x
		currentY = y
	}

	return command
}

/* Protobuffer helper */

func GetOrCreateLayer(tile *Tile, name *string) *Tile_Layer {
	// Check if this tile already contains the layer and if not create it
	for _, layer := range tile.Layers {
		if layer.Name == name {
			return layer
		}
	}

	version := uint32(1)
	ex := extent

	layer := Tile_Layer{}
    layer.Features = make([]*Tile_Feature, 0)
    layer.Extent = &ex
    layer.Name = name
    layer.Version = &version

	tile.Layers = append(
		tile.Layers,
		&layer)

	return &layer
}
