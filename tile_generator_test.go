package main

import "testing"

import "fmt"

func TestCutCommand_LeaveAndEnterAgain(t *testing.T) {
	encoder := &encoder{0, 0}

	// Inside tile:
	step0 := coordinate{49.01985919086641, 8.469658203125}
	step1 := coordinate{49.01985914353444, 8.469658203125}
	step2 := coordinate{49.01985919012332, 8.469658203125}
	step3 := coordinate{49.01987919012332, 8.469658203125}

	// Outside tile:
	step4 := coordinate{49.01985123232333, 9.469658203125}
	step5 := coordinate{49.01985123232333, 9.639658203125}

	// Inside tile again:
	step6 := coordinate{49.01985919086641, 8.469658203125}
	step7 := coordinate{49.01985919012332, 8.469658203125}

	way := []coordinate{step0, step1, step2, step3, step4, step5, step6, step7}

	tileRow := RowFromLatitude(float64(step0.latitude), 16)
	tileCol := ColumnFromLongitude(float64(step0.longitude), 16)

	for _, index := range []int{0, 1, 2, 3, 6, 7} {
		row := RowFromLatitude(float64(way[index].latitude), 16)
		col := ColumnFromLongitude(float64(way[index].longitude), 16)

		if (tileRow != row) || (tileCol != col) {
			fmt.Printf("Expected step %d to be inside the tile\n", index)
			t.Fail()
			return
		}
	}

	for _, index := range []int{4, 5} {
		row := RowFromLatitude(float64(way[index].latitude), 16)
		col := ColumnFromLongitude(float64(way[index].longitude), 16)

		if (tileRow == row) && (tileCol == col) {
			fmt.Printf("Expected step %d to be outside the tile\n", index)
			t.Fail()
			return
		}
	}

	(*encoder).currentX = 0.0
	(*encoder).currentY = 0.0

	// Expect commands steps[0 : 5], steps[5:8]
	command0 := append((*encoder).Command(commandMoveTo, uint32(tileRow), uint32(tileCol), 16, way[0:1]), (*encoder).Command(commandLineTo, uint32(tileRow), uint32(tileCol), 16, way[1:5])...)
	command1 := append((*encoder).Command(commandMoveTo, uint32(tileRow), uint32(tileCol), 16, way[5:6]), (*encoder).Command(commandLineTo, uint32(tileRow), uint32(tileCol), 16, way[6:8])...)
	expectedCommand := append(command0, command1...)

	(*encoder).currentX = 0
	(*encoder).currentY = 0

	cutCommand := (*encoder).CutCommand(uint32(tileRow), uint32(tileCol), 16, way)

	if len(expectedCommand) != len(cutCommand) {
		fmt.Printf("Expected %d bytes but got %d bytes\n", len(expectedCommand), len(cutCommand))
		t.Fail()
		return
	}

	for i, b := range cutCommand {
		if b != expectedCommand[i] {
			fmt.Printf("At index %d, got different bytes\n", i)
			t.Fail()
			break
		}
	}
}

func TestCutCommand_NotInTile(t *testing.T) {
	encoder := &encoder{0, 0}
	step0 := coordinate{49.01985919086641, 8.469658203125}
	step1 := coordinate{49.01985914353444, 8.469658203125}
	step2 := coordinate{49.01985919012332, 8.469658203125}

	way := []coordinate{step0, step1, step2}

	// Bogus tile row and columns
	tileRow := 1200
	tileCol := 1300

	result := (*encoder).CutCommand(uint32(tileRow), uint32(tileCol), 16, way)

	if len(result) != 0 {
		fmt.Printf("Expected resulting command to be empty but got %d bytes\n", len(result))
		t.Fail()
		return
	}
}

func TestCutCommand_CompletelyInTile(t *testing.T) {
	encoder := &encoder{0, 0}
	// Inside tile:
	step0 := coordinate{49.01985919086641, 8.469658203125}
	step1 := coordinate{49.01985914353444, 8.469658203125}
	step2 := coordinate{49.01985919012332, 8.469658203125}
	step3 := coordinate{49.01987919012332, 8.469658203125}

	way := []coordinate{step0, step1, step2, step3}

	tileRow := RowFromLatitude(float64(step0.latitude), 16)
	tileCol := ColumnFromLongitude(float64(step0.longitude), 16)

	for _, index := range []int{0, 1, 2, 3} {
		row := RowFromLatitude(float64(way[index].latitude), 16)
		col := ColumnFromLongitude(float64(way[index].longitude), 16)

		if (tileRow != row) || (tileCol != col) {
			fmt.Printf("Expected step %d to be inside the tile\n", index)
			t.Fail()
			return
		}
	}

	(*encoder).currentX = 0
	(*encoder).currentY = 0

	// Expect commands steps[0 : 4]
	expectedCommand := append((*encoder).Command(commandMoveTo, uint32(tileRow), uint32(tileCol), 16, way[0:1]), (*encoder).Command(commandLineTo, uint32(tileRow), uint32(tileCol), 16, way[1:4])...)

	(*encoder).currentX = 0
	(*encoder).currentY = 0

	cutCommand := (*encoder).CutCommand(uint32(tileRow), uint32(tileCol), 16, way)

	if len(expectedCommand) != len(cutCommand) {
		fmt.Printf("Expected %d bytes but got %d bytes\n", len(expectedCommand), len(cutCommand))
		t.Fail()
		return
	}

	for i, b := range cutCommand {
		if b != expectedCommand[i] {
			fmt.Printf("At index %d, got different bytes\n", i)
			t.Fail()
			break
		}
	}
}

func TestCutCommand_StartOutside(t *testing.T) {
	encoder := &encoder{0, 0}
	// Outside tile:
	step0 := coordinate{49.01985123232333, 9.469658203125}
	step1 := coordinate{49.01985123232333, 9.629658203125}

	// Inside tile:
	step2 := coordinate{49.01985123232333, 8.467658203125}
	step3 := coordinate{49.01985123232354, 8.467658203125}

	way := []coordinate{step0, step1, step2, step3}

	tileRow := RowFromLatitude(float64(step2.latitude), 16)
	tileCol := ColumnFromLongitude(float64(step2.longitude), 16)

	for _, index := range []int{2, 3} {
		row := RowFromLatitude(float64(way[index].latitude), 16)
		col := ColumnFromLongitude(float64(way[index].longitude), 16)

		if (tileRow != row) || (tileCol != col) {
			fmt.Printf("Expected step %d to be inside the tile\n", index)
			t.Fail()
			return
		}
	}

	for _, index := range []int{0, 1} {
		row := RowFromLatitude(float64(way[index].latitude), 16)
		col := ColumnFromLongitude(float64(way[index].longitude), 16)

		if (tileRow == row) && (tileCol == col) {
			fmt.Printf("Expected step %d to be outside the tile\n", index)
			t.Fail()
			return
		}
	}

	(*encoder).currentX = 0
	(*encoder).currentY = 0

	// Expect commands steps[1 : 4]
	expectedCommand := append((*encoder).Command(commandMoveTo, uint32(tileRow), uint32(tileCol), 16, way[1:2]), (*encoder).Command(commandLineTo, uint32(tileRow), uint32(tileCol), 16, way[2:4])...)

	(*encoder).currentX = 0
	(*encoder).currentY = 0

	cutCommand := (*encoder).CutCommand(uint32(tileRow), uint32(tileCol), 16, way)

	if len(expectedCommand) != len(cutCommand) {
		fmt.Printf("Expected %d bytes but got %d bytes\n", len(expectedCommand), len(cutCommand))
		t.Fail()
		return
	}

	for i, b := range cutCommand {
		if b != expectedCommand[i] {
			fmt.Printf("At index %d, got different bytes\n", i)
			t.Fail()
			break
		}
	}
}

func TestCutCommand_BrieflyLeaveAndEnterAgain(t *testing.T) {
	encoder := &encoder{0, 0}
	// Inside tile:
	step0 := coordinate{49.01985919086641, 8.469658203125}
	step1 := coordinate{49.01985914353444, 8.469658203125}
	step2 := coordinate{49.01985919012332, 8.469658203125}
	step3 := coordinate{49.01987919012332, 8.469658203125}

	// Outside tile:
	step4 := coordinate{49.01985123232333, 9.469658203125}

	// Inside tile again:
	step5 := coordinate{49.01985919012332, 8.469658203125}
	step6 := coordinate{49.01985919012332, 8.469658203125}

	way := []coordinate{step0, step1, step2, step3, step4, step5, step6}

	tileRow := RowFromLatitude(float64(step0.latitude), 16)
	tileCol := ColumnFromLongitude(float64(step0.longitude), 16)

	for _, index := range []int{0, 1, 2, 3, 5, 6} {
		row := RowFromLatitude(float64(way[index].latitude), 16)
		col := ColumnFromLongitude(float64(way[index].longitude), 16)

		if (tileRow != row) || (tileCol != col) {
			fmt.Printf("Expected step %d to be inside the tile\n", index)
			t.Fail()
			return
		}
	}

	for _, index := range []int{4} {
		row := RowFromLatitude(float64(way[index].latitude), 16)
		col := ColumnFromLongitude(float64(way[index].longitude), 16)

		if (tileRow == row) && (tileCol == col) {
			fmt.Printf("Expected step %d to be outside the tile\n", index)
			t.Fail()
			return
		}
	}

	(*encoder).currentX = 0
	(*encoder).currentY = 0

	// Expect commands steps[0 : 4], steps[5:7]
	command0 := append((*encoder).Command(commandMoveTo, uint32(tileRow), uint32(tileCol), 16, way[0:1]), (*encoder).Command(commandLineTo, uint32(tileRow), uint32(tileCol), 16, way[1:5])...)
	command1 := append((*encoder).Command(commandMoveTo, uint32(tileRow), uint32(tileCol), 16, way[4:5]), (*encoder).Command(commandLineTo, uint32(tileRow), uint32(tileCol), 16, way[5:7])...)
	expectedCommand := append(command0, command1...)

	(*encoder).currentX = 0
	(*encoder).currentY = 0

	cutCommand := (*encoder).CutCommand(uint32(tileRow), uint32(tileCol), 16, way)

	if len(expectedCommand) != len(cutCommand) {
		fmt.Printf("Expected %d bytes but got %d bytes\n", len(expectedCommand), len(cutCommand))
		t.Fail()
		return
	}

	for i, b := range cutCommand {
		if b != expectedCommand[i] {
			fmt.Printf("At index %d, got different bytes\n", i)
			t.Fail()
			break
		}
	}
}
