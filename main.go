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

const arinPath = "src/github.com/demskie/networktree/inputdata/delegated-arin-extended-latest"    // https://ftp.arin.net/pub/stats/arin/
const ripePath = "src/github.com/demskie/networktree/inputdata/delegated-ripencc-extended-latest" // https://ftp.ripe.net/ripe/stats/
// http://ftp.apnic.net/stats/apnic/
// http://ftp.apnic.net/stats/afrinic/
// https://ftp.lacnic.net/pub/stats/lacnic/

func main() {

	t := time.Now()

	tree := NewTree(32)
	ingest(tree, arinPath)
	ingest(tree, ripePath)

	ticker.Stop()
	fmt.Println("finished in", time.Since(t))
}

var rate uint64
var parentRate uint64
var noParentRate uint64
var ticker *time.Ticker

func init() {
	ticker = time.NewTicker(time.Second)
	go func() {
		var total uint64
		for range ticker.C {
			old := atomic.SwapUint64(&rate, 0)
			total += old
			fmt.Printf("%v count/sec    %v total    %v insertWithParent()    %v insertWithoutParent()\n",
				old, total, atomic.SwapUint64(&parentRate, 0), atomic.SwapUint64(&noParentRate, 0))
		}
	}()
}

func ingest(tree *Tree, p string) {
	gopath, _ := os.LookupEnv("GOPATH")
	txtFile, err := os.Open(path.Join(gopath, p))
	if err != nil {
		log.Fatalf("unable to ingest IPv4 data because: %v", err)
	}
	reader := csv.NewReader(bufio.NewReader(txtFile))
	reader.Comma = '|'

	for {
		lineColumns, err := reader.Read()
		if err == io.EOF {
			break
		}
		if len(lineColumns) < 5 || lineColumns[2] != "ipv4" || lineColumns[3] == "*" {
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
		atomic.AddUint64(&rate, uint64(len(networks)))
		tree.insert(networks, country, position)
	}
	//fmt.Println(tree.JSON())
}
