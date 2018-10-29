package main

import "fmt"
import "sort"

func setLatLong(nodes []osmnode, id int64, point coordinate) bool {
	index := searchIndex(nodes, id)
	if index != -1 {
		nodes[index].latitude = point.latitude
		nodes[index].longitude = point.longitude
		return true
	}

	return false
}

// func searchIndex(nodes []osmnode, id int64) int {
// 	mid := len(nodes) / 2
// 	if len(nodes) == 0 {
// 		return -1
// 	} else if nodes[mid].id > id {
// 		return searchIndex(nodes[:mid], id)
// 	} else if nodes[mid].id < id {
// 		return searchIndex(nodes[mid+1:], id)
// 	} else {
// 		return mid // got it.
// 	}
// }

func searchIndex(nodes []osmnode, id int64) int {
	index := sort.Search(len(nodes), func(i int) bool { return id <= nodes[i].id })
	if index == len(nodes) {
		return -1
	} else {
		return index
	}
}

func getLatLong(nodes []osmnode, id int64) coordinate {
	index := searchIndex(nodes, id)
	if index != -1 {
		return coordinate{nodes[index].latitude, nodes[index].longitude}
	}

	fmt.Printf("GetLatLong Error: cannot find node with id %d\n", id)
	return coordinate{float32(0.0), float32(0.0)}
}
