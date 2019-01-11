package main

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	"math"
	"math/big"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"sync/atomic"

	"github.com/demskie/simplesync"
	"github.com/demskie/subnetmath"
)

func ingest(tree *Tree, p string) {
	lineInfoChan := parseLineInfo(p)
	nodeInfoChan := outputNodeInfo(lineInfoChan)
	for val := range nodeInfoChan {
		atomic.AddUint64(&counters.rate, uint64(len(val.networks)))
		tree.insert(val.networks, val.country, val.position)
	}
}

type lineInfo struct {
	startAddr net.IP
	increment float64
	country   string
}

func parseLineInfo(p string) chan lineInfo {
	lineInfoChan := make(chan lineInfo, 4096)
	go func() {
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
				// txtFile.Seek(0, 0)
				// continue
				break
			}
			if len(lineColumns) < 5 || lineColumns[3] == "*" {
				continue
			}
			startAddr := net.ParseIP(lineColumns[3])
			country := lineColumns[1]
			if startAddr == nil || country == "" {
				country = "ZZ"
			}
			if lineColumns[2] == "ipv4" {
				increment, err := strconv.ParseUint(lineColumns[4], 10, 64)
				if err != nil || increment%2 != 0 {
					continue
				}
				lineInfoChan <- lineInfo{
					startAddr, float64(increment), country,
				}
			} else if lineColumns[2] == "ipv6" {
				startAddr := net.ParseIP(lineColumns[3])
				cidr, err := strconv.ParseUint(lineColumns[4], 10, 64)
				if err != nil || cidr < 1 || cidr > 128 {
					continue
				}
				lineInfoChan <- lineInfo{
					startAddr, math.Exp2(float64(cidr)), country,
				}
			}
		}
		close(lineInfoChan)
	}()
	return lineInfoChan
}

type nodeInfo struct {
	networks []*net.IPNet
	country  string
	position *Position
}

func outputNodeInfo(lineInfoChan chan lineInfo) chan nodeInfo {
	nodeInfoChan := make(chan nodeInfo, 4096)
	go func() {
		simplesync.NewWorkerPool(runtime.NumCPU()).Execute(func(i int) {
			sbuf := subnetmath.NewBuffer()
			stopBigInt := new(big.Int)
			bigIncrement := new(big.Int)
			bigOne := big.NewInt(1)
			for l := range lineInfoChan {
				stopBigInt = subnetmath.AddrToInt(l.startAddr)
				stopBigInt.Add(stopBigInt, bigIncrement.SetInt64(int64(l.increment)))
				stopBigInt.Sub(stopBigInt, bigOne)
				stopAddr := subnetmath.IntToAddr(stopBigInt)
				// fmt.Println(l.startAddr, stopAddr, l.increment)
				networks := sbuf.FindInbetweenSubnets(l.startAddr, stopAddr)
				// fmt.Println(networks)
				position, exists := countryPositions[l.country]
				if !exists {
					log.Fatalf("country code '%v' has no defined position", l.country)
				}
				nodeInfoChan <- nodeInfo{
					networks, l.country, position,
				}
			}
		})
		close(nodeInfoChan)
	}()
	return nodeInfoChan
}
