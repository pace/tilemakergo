package main

import (
	"encoding/binary"
	"io"
	"bytes"
	"os"
	"log"
	"time"
	"errors"
	"fmt"
	"sync"
	"tilemakergo/OSMPBF"
	"compress/zlib"
)

// type Decoder struct {
// 	r          io.Reader
// 	serializer chan pair

// 	buf *bytes.Buffer

// 	// store header block
// 	header *Header
// 	// synchronize header deserialization
// 	headerOnce sync.Once

// 	// for data decoders
// 	inputs  []chan<- pair
// 	outputs []<-chan pair
// }


const (
	maxBlobHeaderSize = 64 * 1024
	maxBlobSize = 32 * 1024 * 1024
)

type BoundingBox struct {
	Left   float64
	Right  float64
	Top    float64
	Bottom float64
}

type Header struct {
	BoundingBox                      *BoundingBox
	RequiredFeatures                 []string
	OptionalFeatures                 []string
	WritingProgram                   string
	Source                           string
	OsmosisReplicationTimestamp      time.Time
	OsmosisReplicationSequenceNumber int64
	OsmosisReplicationBaseUrl        string
}

var (
	parseCapabilities = map[string]bool{
		"OsmSchema-V0.6": true,
		"DenseNodes":     true,
	}
)

type decoder struct {
	output chan []byte
	decode chan []interface{}
}

func (d *decoder) readOsmPbf(path *string) {
	var buffer = bytes.NewBuffer(make([]byte, 0, 32 * 1024 * 1024))
	reader, err := os.Open(*path)

	if err != nil {
		log.Printf("Error while reading input file: %s\n", err)
	}

	log.Printf("Start decoding")

	blobHeader, blob, err := (*d).readFileBlock(buffer, reader)

	if err != nil {
		log.Printf("Error while decoding blob header: %s\n", err)
		return
	}

	switch(blobHeader.GetType()) {
		case "OSMHeader": 
			osmHeader, err := (*d).decodeOSMHeader(blob)

			if err != nil {
				log.Printf("Error while decoding osm header: %s", err)
			}

			log.Printf("OSM Header read: %s\n", osmHeader)

			(*d).decodeOsmData(buffer, reader)
		default:
			log.Printf("Unknown blob header type")
			return
	}
}

func (d *decoder) decodeOsmData(buffer *bytes.Buffer, reader io.Reader) {	
	(*d).output = make(chan []byte, 8000)
	(*d).decode = make(chan []interface{}, 8000)

	// Read with a single thread :-(
	go func() {
		for {
			_, blob, err := (*d).readFileBlock(buffer, reader)

			if err != nil {
				log.Printf("Error while decoding blob header: %s\n", err)
				close((*d).output)
				break
			}

			data, err := (*d).getData(blob)

			(*d).output <- data
		}
	}()

	// Decode with multiple threads
    var wg sync.WaitGroup

	threads := 8
	for i := 0; i < threads; i++ { 
		wg.Add(1)
		go func() {
			defer wg.Done()
			for out := range (*d).output {
				dd := new(dataDecoder)
				objects, err := dd.Decode(out)

				if err != nil {
					break
				}

				(*d).decode <- objects
			}
		}()
	}

	go func() {
		wg.Wait()
		close((*d).decode)
	}()
}

func (d *decoder) read() ([]interface{}, error) {
	objects, ok := <-(*d).decode

	if !ok {
		return nil, io.EOF
	}

	return objects, nil
}  

/* File block decoding */

func (d *decoder) readFileBlock(buffer *bytes.Buffer, reader io.Reader) (*OSMPBF.BlobHeader, *OSMPBF.Blob, error) {
	blobHeaderSize, err := (*d).readBlobHeaderSize(buffer, reader)

	if err != nil {
		return nil, nil, err
	}

	blobHeader, err := (*d).readBlobHeader(blobHeaderSize, buffer, reader)
	if err != nil {
		return nil, nil, err
	}

	blob, err := (*d).readBlob(blobHeader, buffer, reader)
	if err != nil {
		return nil, nil, err
	}

	return blobHeader, blob, err
}

func (d *decoder) readBlobHeaderSize(buffer *bytes.Buffer, reader io.Reader) (uint32, error) {
	buffer.Reset()

	if _, err := io.CopyN(buffer, reader, 4); err != nil {
		return 0, err
	}

	size := binary.BigEndian.Uint32(buffer.Bytes())

	if size >= maxBlobHeaderSize {
		return 0, errors.New("BlobHeader size >= 64Kb")
	}

	return size, nil
}

func (d *decoder) readBlobHeader(size uint32, buffer *bytes.Buffer, reader io.Reader) (*OSMPBF.BlobHeader, error) {
	buffer.Reset()
	if _, err := io.CopyN(buffer, reader, int64(size)); err != nil {
		return nil, err
	}

	blobHeader := new(OSMPBF.BlobHeader)
	if err := blobHeader.XXX_Unmarshal(buffer.Bytes()); err != nil {
		return nil, err
	}

	if blobHeader.GetDatasize() >= maxBlobSize {
		return nil, errors.New("Blob size >= 32Mb")
	}
	return blobHeader, nil
}

func (d *decoder) readBlob(blobHeader *OSMPBF.BlobHeader, buffer *bytes.Buffer, reader io.Reader) (*OSMPBF.Blob, error) {
	buffer.Reset()

	if _, err := io.CopyN(buffer, reader, int64(blobHeader.GetDatasize())); err != nil {
		return nil, err
	}

	blob := new(OSMPBF.Blob)
	if err := blob.XXX_Unmarshal(buffer.Bytes()); err != nil {
		return nil, err
	}

	return blob, nil
}

/* */

func (d *decoder) decodeOSMHeader(blob *OSMPBF.Blob) (*Header, error) {
	data, err := (*d).getData(blob)
	if err != nil {
		return nil, err
	}

	headerBlock := new(OSMPBF.HeaderBlock)
	if err := headerBlock.XXX_Unmarshal(data); err != nil {
		return nil, err
	}

	// Check we have the parse capabilities
	requiredFeatures := headerBlock.GetRequiredFeatures()
	for _, feature := range requiredFeatures {
		if !parseCapabilities[feature] {
			return nil, fmt.Errorf("parser does not have %s capability", feature)
		}
	}

	// Read properties to header struct
	header := &Header{
		RequiredFeatures: headerBlock.GetRequiredFeatures(),
		OptionalFeatures: headerBlock.GetOptionalFeatures(),
		WritingProgram:   headerBlock.GetWritingprogram(),
		Source:           headerBlock.GetSource(),
		OsmosisReplicationBaseUrl:        headerBlock.GetOsmosisReplicationBaseUrl(),
		OsmosisReplicationSequenceNumber: headerBlock.GetOsmosisReplicationSequenceNumber(),
	}

	// convert timestamp epoch seconds to golang time structure if it exists
	if headerBlock.OsmosisReplicationTimestamp != 0 {
		header.OsmosisReplicationTimestamp = time.Unix(headerBlock.OsmosisReplicationTimestamp, 0)
	}
	// read bounding box if it exists
	if headerBlock.Bbox != nil {
		// Units are always in nanodegree and do not obey granularity rules. See osmformat.proto
		header.BoundingBox = &BoundingBox{
			Left:   1e-9 * float64(headerBlock.Bbox.Left),
			Right:  1e-9 * float64(headerBlock.Bbox.Right),
			Bottom: 1e-9 * float64(headerBlock.Bbox.Bottom),
			Top:    1e-9 * float64(headerBlock.Bbox.Top),
		}
	}

	return header, nil
}

func (d *decoder) getData(blob *OSMPBF.Blob) ([]byte, error) {
	switch {
	case blob.Raw != nil:
		return blob.GetRaw(), nil

	case blob.ZlibData != nil:
		r, err := zlib.NewReader(bytes.NewReader(blob.GetZlibData()))

		if err != nil {
			return nil, err
		}

		buf := bytes.NewBuffer(make([]byte, 0, blob.GetRawSize() + bytes.MinRead))
		_, err = buf.ReadFrom(r)

		if err != nil {
			return nil, err
		}

		if buf.Len() != int(blob.GetRawSize()) {
			err = fmt.Errorf("raw blob data size %d but expected %d", buf.Len(), blob.GetRawSize())
			return nil, err
		}

		return buf.Bytes(), nil

	default:
		return nil, errors.New("unknown blob data")
	}
}