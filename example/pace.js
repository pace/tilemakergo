console.log("Javascript says: I'm ready.");

function useNode(a) {
  return false;
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
    "layer": "road",
    "properties": {
    }
  }
}

function processWay(a) {
  return {
    "layer": "road",
    "properties": {
      "hw":  a[0].Tags["highway"],
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

function isASCII(str) {
    return /^[\x00-\x7F]*$/.test(str);
}