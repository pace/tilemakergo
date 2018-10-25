package main

import (
	"fmt"
	"math"
	"testing"
)

// Simple conversion tests using data from MapKit.
func TestColumnFromLongitude(t *testing.T) {
	column := ColumnFromLongitude(8.4649658203125, 16)
	if column != 34309 {
		fmt.Printf("Expected column number 34309, but got %d\n", column)
		t.Fail()
	}
}

func TestRowFromLatitude(t *testing.T) {
	row := RowFromLatitude(49.01985919086641, 16)
	if row != 22501 {
		fmt.Printf("Expected row number 22501, but got %d\n", row)
		t.Fail()
	}
}

func TestColumnFromLongitudeF(t *testing.T) {
	epsilon := float64(0.00001)
	columnf := ColumnFromLongitudeF(8.466064453124972, 16)
	if math.Abs(columnf-34309.2) > epsilon {
		fmt.Printf("Expected column 34309.2, but got %f\n", columnf)
		t.Fail()
	}
}

func TestRowFromLatitudeF(t *testing.T) {
	epsilon := float64(0.00001)
	rowf := RowFromLatitudeF(49.0198591908, 16)
	if math.Abs(rowf-22501.2) > epsilon {
		fmt.Printf("Expected row 22501.2, but got %f\n", rowf)
	}
}
