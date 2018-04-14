# Import

* Stream information (https://github.com/qedus/osmpbf)

# Process

* Load javascript / lua / go process script which has entry points:
* https://github.com/wendigo/go-bind-plugin
 * useNode()
 * useWay()
 * useRelation()
 * processNode()
 * processWay()
 * processRelation()

# Export

* Export for definied zoom level
* Export as .mbtiles database which contains X, Y, Payload
 * Payload is a single tile: https://www.mapbox.com/vector-tiles/specification/