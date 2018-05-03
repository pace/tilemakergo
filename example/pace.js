var road_1_layer = "r1";
var road_2_layer = "r2";
var road_3_layer = "r3";
var road_4_layer = "r4";
var poi_layer = "pi";
var house_number_layer = "hn";

var road_1_highway_types = [ "residential", "unclassified", "tertiary", "secondary", "primary", "living_street", "motorway", "motorway_link", "trunk", "trunk_link", "primary_link", "motorway_junction", "tertiary_link" ]
var road_2_highway_types = [ "service", "turning_cycle", "turning_loop", "mini_roundabout", "raceway", "rest_area", "services"  ]
var road_3_highway_types = [ "passing_place", "construction" ]
var road_4_highway_types = [ "path", "footway", "bus_stop", "cycleway", "crossing", "pedestrian", "bridleway" ]


function useNode(node) {
  return ('highway' in node[0].Tags || 'addr:housenumber' in node[0].Tags);
}

function useWay(way) {
  return ('highway' in way[0].Tags || 'addr:housenumber' in way[0].Tags);
}

function useRelation(a) {
  return false;
}

function processNode(node) {
  var properties = {};
  var layer = "";

  if ('addr:housenumber' in node[0].Tags) {
    layer = house_number_layer;
    properties["nm"] = node[0].Tags["name"];
    properties["hn"] = node[0].Tags["addr:housenumber"];
    properties["st"] = node[0].Tags["addr:street"];
  } else if ('highway' in node[0].Tags && node[0].Tags['highway'] == 'speed_camera') {
    layer = poi_layer;
    properties["nm"] = node[0].Tags["name"];
    properties["hw"] = node[0].Tags["highway"];
    properties["ms"] = node[0].Tags["maxspeed"];
  } 

  return {
    "layer": layer,
    "properties": properties
  };
}

function processWay(way) {
  var layer = road_4_layer;
  var properties = {};
  var highway = way[0].Tags["highway"];

  if (highway != null) {
    if (arrayContains(road_1_highway_types, highway)) {
      layer = road_1_layer;
    } else if (arrayContains(road_2_highway_types, highway)) {
      layer = road_2_layer;
    } else if (arrayContains(road_3_highway_types, highway)) {
      layer = road_3_layer;
    }

    properties = { "hw":  highway };
  } else {
    if ('addr:housenumber' in way[0].Tags) {
      layer = house_number_layer;

      properties = {
        "hn": way[0].Tags["addr:housenumber"],
        "st": way[0].Tags["addr:street"],
      };
    }
  }

  return {
    "layer": layer,
    "properties": properties
  };
}

function processRelation(a) {
  //console.log(JSON.stringify(a))
  return {
    "layer": "rel",
    "properties": {
      "aa": "bb",
    }
  };
}

/* Helper function for array access */
function arrayContains(array, needle)
{
    return (array.indexOf(needle) > -1);
}