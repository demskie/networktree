package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/demskie/subnetmath"
)

func main() {
	t := time.Now()
	ingestIPv4("src/github.com/demskie/networktree/IpToCountry.csv")
	fmt.Println(time.Since(t))
}

// http://ftp.apnic.net/stats/apnic/
// http://ftp.apnic.net/stats/afrinic/
// https://ftp.arin.net/pub/stats/arin/
// https://ftp.lacnic.net/pub/stats/lacnic/
// https://ftp.ripe.net/ripe/stats/

func ingestIPv4(p string) {
	gopath, _ := os.LookupEnv("GOPATH")
	csvFile, err := os.Open(path.Join(gopath, p))
	if err != nil {
		log.Fatalf("unable to ingest IPv4 data because: %v", err)
	}
	tree := NewTree()
	tree.insertAggregatesV4()
	reader := csv.NewReader(bufio.NewReader(csvFile))

	rate := uint64(0)
	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			fmt.Printf("%v insertions/second\n", atomic.SwapUint64(&rate, 0))
			fmt.Printf("%v root size\n", tree.Length())
			fmt.Printf("%v tree size\n\n", tree.Size())
		}
	}()

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
		startAddr := subnetmath.ConvertV4IntegerToAddress(uint32(start))
		endAddr := subnetmath.ConvertV4IntegerToAddress(uint32(end))
		subnets := subnetmath.FindInbetweenV4Subnets(startAddr, endAddr)

		country := lineColumns[4]

		atomic.AddUint64(&rate, uint64(len(subnets)))

		tree.insertIPv4(subnets, country, position)
	}

	ticker.Stop()

	tree.Print()
}
