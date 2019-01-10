package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"math/big"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/demskie/simplesync"

	"github.com/demskie/subnetmath"
)

const basePath = "src/github.com/demskie/networktree/inputdata/"
const arinPath = basePath + "delegated-arin-extended-latest"       // https://ftp.arin.net/pub/stats/arin/
const ripePath = basePath + "delegated-ripencc-extended-latest"    // https://ftp.ripe.net/ripe/stats/
const apnicPath = basePath + "delegated-apnic-extended-latest"     // http://ftp.apnic.net/stats/apnic/
const afrinicPath = basePath + "delegated-afrinic-extended-latest" // http://ftp.apnic.net/stats/afrinic/
const lacnicPath = basePath + "delegated-lacnic-extended-latest"   // https://ftp.lacnic.net/pub/stats/lacnic/

func main() {
	t := time.Now()

	tree := NewTree(64)

	cpuProf, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(cpuProf)
	defer pprof.StopCPUProfile()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			pprof.StopCPUProfile()
			runtime.GC()
			heapProf, err := os.Create("heap.prof")
			if err != nil {
				log.Fatal(err)
			}
			pprof.WriteHeapProfile(heapProf)
			f, _ := os.Create("output.json")
			f.Truncate(0)
			f.Seek(0, 0)
			f.WriteString(tree.JSON())
			os.Exit(1)
		}
	}()

	ingest(tree, arinPath)
	ingest(tree, ripePath)
	ingest(tree, apnicPath)
	ingest(tree, afrinicPath)
	ingest(tree, lacnicPath)

	ticker.Stop()
	fmt.Println("finished in", time.Since(t))

	f, _ := os.Create("output.json")
	f.Truncate(0)
	f.Seek(0, 0)
	f.WriteString(tree.JSON())
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

	type lineInfo struct {
		startAddr net.IP
		increment float64
		country   string
	}

	lineInfoChan := make(chan lineInfo, 4096)

	go func() {
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

	type nodesToInsert struct {
		networks []*net.IPNet
		country  string
		position *Position
	}

	nodesToInsertChan := make(chan nodesToInsert, 4096)

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
				nodesToInsertChan <- nodesToInsert{
					networks, l.country, position,
				}
			}
		})
		close(nodesToInsertChan)
	}()

	for val := range nodesToInsertChan {
		atomic.AddUint64(&rate, uint64(len(val.networks)))
		tree.insert(val.networks, val.country, val.position)
	}
}
