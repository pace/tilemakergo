## Requirements

To build this project from source you need:
* golang >= v1.11.1 

## Install dependencies 

```
go get github.com/mattn/go-sqlite3
go get github.com/robertkrimen/otto
go get github.com/qedus/osmpbf

```

## Build

Build the go application:
```
go build
```

Build the protobuffers:
```
protoc -I=. -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf --gogoslick_out=. *.proto
```

## Run

```
./tilemakergo -in=karlsruhe.pbf -out=karlsruhe.mbtiles -processor=example/pace.js
```