console.log("Javascript says: I'm ready.");

function useNode(a) {
  return true;
}

function useWay(a) {
  return true;
}

function useRelation(a) {
  return true;
}

function processNode(a) {
  //console.log(JSON.stringify(a))
  return {
    "layer": "dot",
    "properties": {
      "aa": "bb",
    }
  }
}

function processWay(a) {
  //console.log(JSON.stringify(a))
  return {
    "layer": "road",
    "properties": {
      "aa": "bb",
    }
  }
}

function processRelation(a) {
  //console.log(JSON.stringify(a))
  return {
    "layer": "rel",
    "properties": {
      "aa": "bb",
    }
  }
}
