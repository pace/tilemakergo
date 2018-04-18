console.log("Javascript says: I'm ready.");

function useNode(a) {
  return true;
}

function useWay(a) {
  if ('highway' in a[0].Tags) {
    return true;
  } else {
    return false;
  }
}

function useRelation(a) {
  return false;
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
