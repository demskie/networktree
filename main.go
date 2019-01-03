package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"path"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/demskie/subnetmath"
)

func main() {
	t := time.Now()
	ingestARIN("src/github.com/demskie/networktree/delegated-arin-extended-latest")
	fmt.Println(time.Since(t))
}

// http://ftp.apnic.net/stats/apnic/
// http://ftp.apnic.net/stats/afrinic/
// https://ftp.arin.net/pub/stats/arin/
// https://ftp.lacnic.net/pub/stats/lacnic/
// https://ftp.ripe.net/ripe/stats/

func ingestARIN(p string) {
	gopath, _ := os.LookupEnv("GOPATH")
	txtFile, err := os.Open(path.Join(gopath, p))
	if err != nil {
		log.Fatalf("unable to ingest IPv4 data because: %v", err)
	}
	reader := csv.NewReader(bufio.NewReader(txtFile))
	reader.Comma = '|'

	tree := NewTree(32)

	rate := uint64(0)
	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			fmt.Printf("%v insertions/second\n", atomic.SwapUint64(&rate, 0))
			fmt.Printf("%v tree size\n\n", tree.Size())
		}
	}()

	for {
		lineColumns, err := reader.Read()
		if err == io.EOF {
			break
		}
		if len(lineColumns) < 7 || lineColumns[2] != "ipv4" || lineColumns[3] == "*" {
			continue
		}
		startAddr := net.ParseIP(lineColumns[3])
		increment, err := strconv.ParseUint(lineColumns[4], 10, 64)
		if startAddr == nil || err != nil || increment%2 != 0 {
			continue
		}
		stopBigInt := subnetmath.AddrToInt(startAddr)
		stopBigInt.Add(stopBigInt, big.NewInt(int64(increment)))
		stopBigInt.Sub(stopBigInt, big.NewInt(1))
		stopAddr := subnetmath.IntToAddr(stopBigInt)
		networks := subnetmath.FindInbetweenSubnets(startAddr, stopAddr)
		country := lineColumns[1]
		if country == "" {
			country = "ZZ"
		}
		position, exists := countryPositions[country]
		if !exists {
			log.Fatalf("country code '%v' has no defined position", country)
		}
		atomic.AddUint64(&rate, uint64(1))
		tree.insert(networks, country, position)
	}
}
