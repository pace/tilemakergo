package main

import (
	"log"
	"math"
)

func ColumnFromLongitude(lon float64, zoom int) int {
	return int(ColumnFromLongitudeF(lon, zoom))
}

func RowFromLatitude(lat float64, zoom int) int {
	return int(RowFromLatitudeF(lat, zoom))
}

func ColumnFromLongitudeF(lon float64, zoom int) float64 {
	return (lon + 180.0) / 360.0 * float64(math.Pow(2.0, float64(zoom)))
}

func RowFromLatitudeF(lat float64, zoom int) float64 {
	return float64((1.0 - math.Log(math.Tan(float64(lat)*math.Pi/180.0)+1.0/math.Cos(float64(lat)*math.Pi/180.0))/math.Pi) / 2.0 * math.Pow(2.0, float64(zoom)))
}

func mergeGeoBounds(boundsA []float64, boundsB []float64) []float64 {
	if (len(boundsA) != 4) || (len(boundsB) != 4) {
		log.Fatal("Expected bounds to have length 4.")
	}

	// bound order: left, bottom, right, top
	// TODO: naive merging is wrong for most countries.

	result := make([]float64, 4)

	result[0] = math.Min(boundsA[0], boundsB[0]) // left
	result[1] = math.Min(boundsA[1], boundsB[1]) // bottom
	result[2] = math.Max(boundsA[2], boundsB[2]) // right
	result[3] = math.Max(boundsA[3], boundsB[3]) // top

	return result
}
