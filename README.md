# TileMakerGO

Generate vector tiles with custom information from osm pbf extracts. 

## CPU / Ram requirements

For generating tiles a huge amount of ram is needed. E.g. for a Germany extract 32 GB is the minimum needed. 

## Install dependencies 

```
go get github.com/mattn/go-sqlite3
go get github.com/golang/protobuf/proto
go get github.com/gogo/protobuf/proto

```

## Build from source

To build this project from source you need:
* golang >= v1.11.1 

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
./tilemakergo -in=karlsruhe.pbf -out=karlsruhe.mbtiles
```

## Install process [Ubuntu 18.04]

```
cd /tmp
wget https://dl.google.com/go/go1.11.1.linux-amd64.tar.gz

sudo tar -xvf go1.11.1.linux-amd64.tar.gz
sudo mv go /usr/local

sudo apt-get -y install gcc

export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH

cd $GOPATH/src
git clone https://github.com/pace/tilemakergo.git
cd tilemakergo

go get github.com/mattn/go-sqlite3
go get github.com/golang/protobuf/proto
go get github.com/gogo/protobuf/proto

go build
```
