package main

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/demskie/subnetmath"
)

func main() {
	spew.Dump(ingestIPv4("/Users/alex.demskie/go/src/github.com/demskie/networktree/IpToCountry.csv"))
}

type geoDataV4 struct {
	ipStart  *net.IP
	ipEnd    *net.IP
	country  string
	position *Position
}

func ingestIPv4(path string) []geoDataV4 {
	csvFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("unable to ingest IPv4 data because: %v", err)
	}
	results := []geoDataV4{}
	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		lineColumns, err := reader.Read()
		if err == io.EOF {
			break
		}
		if len(lineColumns) < 7 || strings.HasPrefix(lineColumns[0], "#") {
			continue
		}
		start, err := strconv.ParseInt(lineColumns[0], 10, 64)
		if err != nil {
			continue
		}
		end, err := strconv.ParseInt(lineColumns[1], 10, 64)
		if err != nil {
			continue
		}
		position, exists := countryPositions[lineColumns[4]]
		if !exists {
			log.Fatalf("country code '%v' has no defined position", lineColumns[4])
		}
		results = append(results, geoDataV4{
			ipStart:  subnetmath.ConvertIntegerIPv4(uint64(start)),
			ipEnd:    subnetmath.ConvertIntegerIPv4(uint64(end)),
			country:  lineColumns[4],
			position: position,
		})
	}
	return results
}
