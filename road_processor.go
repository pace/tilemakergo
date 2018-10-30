package main

import "fmt"

var road_1_layer = "r1"
var road_2_layer = "r2"
var road_3_layer = "r3"
var road_4_layer = "r4"
var poi_layer = "pi"
var house_number_layer = "hn"

var road_1_highway_types = []string{"residential", "unclassified", "tertiary", "secondary", "primary", "living_street", "motorway", "motorway_link", "trunk", "trunk_link", "primary_link", "motorway_junction", "tertiary_link"}
var road_2_highway_types = []string{"service", "turning_cycle", "turning_loop", "mini_roundabout", "raceway", "rest_area", "services"}
var road_3_highway_types = []string{"passing_place", "construction"}
var road_4_highway_types = []string{"path", "footway", "bus_stop", "cycleway", "crossing", "pedestrian", "bridleway", "track"}

// Include functions

func nodeIncluded(tags *map[string]string) bool {
	if _, ok := (*tags)["addr:housenumber"]; ok {
		return true
	}

	if v, ok := (*tags)["highway"]; ok && v == "speed_camera" {
		return true
	}

	return false
}

func wayIncluded(tags *map[string]string) bool {
	if _, ok := (*tags)["addr:housenumber"]; ok {
		return true
	}

	if _, ok := (*tags)["highway"]; ok {
		return true
	}

	return false
}

func relationIncluded(tags *map[string]string) bool {
	return false
}

// Process functions

func processNode(tags *map[string]string, id int64) (layer *string, properties map[string]interface{}) {
	properties = map[string]interface{}{}
	layer = nil

	osmID := fmt.Sprint(id)
	properties["id"] = osmID

	if v, ok := (*tags)["addr:housenumber"]; ok {
		layer = &house_number_layer
		properties["nm"] = (*tags)["name"]
		properties["hn"] = v
		properties["st"] = (*tags)["addr:street"]
	} else if v, ok := (*tags)["highway"]; ok && v == "speed_camera" {
		layer = &poi_layer
		properties["nm"] = (*tags)["name"]
		properties["hw"] = v
		properties["ms"] = (*tags)["maxspeed"]
	}

	return
}

func processWay(tags *map[string]string) (layer *string, properties map[string]interface{}) {
	layer = &road_4_layer

	properties = map[string]interface{}{}

	if v, ok := (*tags)["addr:housenumber"]; ok {
		properties["hn"] = v
	}

	if v, ok := (*tags)["addr:street"]; ok {
		properties["st"] = v
	}

	if highway, ok := (*tags)["highway"]; ok {
		if contains(road_1_highway_types, highway) {
			layer = &road_1_layer
		} else if contains(road_2_highway_types, highway) {
			layer = &road_2_layer
		} else if contains(road_3_highway_types, highway) {
			layer = &road_3_layer
		}

		properties["hw"] = highway
	} else if _, ok := (*tags)["addr:housenumber"]; ok {
		layer = &house_number_layer
	}

	if v, ok := (*tags)["name"]; ok {
		properties["nm"] = v
	}

	if v, ok := (*tags)["ref"]; ok {
		properties["rf"] = v
	}

	if v, ok := (*tags)["lanes"]; ok {
		properties["ln"] = v
	}

	if v, ok := (*tags)["maxspeed"]; ok {
		properties["ms"] = v
	}

	if v, ok := (*tags)["overtaking"]; ok {
		properties["ot"] = v
	}

	if v, ok := (*tags)["oneway"]; ok {
		properties["ow"] = v
	}

	return
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
