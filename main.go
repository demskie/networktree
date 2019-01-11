package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sync/atomic"
	"time"
)

const basePath = "src/github.com/demskie/networktree/inputdata/"
const arinPath = basePath + "delegated-arin-extended-latest"       // https://ftp.arin.net/pub/stats/arin/
const ripePath = basePath + "delegated-ripencc-extended-latest"    // https://ftp.ripe.net/ripe/stats/
const apnicPath = basePath + "delegated-apnic-extended-latest"     // http://ftp.apnic.net/stats/apnic/
const afrinicPath = basePath + "delegated-afrinic-extended-latest" // http://ftp.apnic.net/stats/afrinic/
const lacnicPath = basePath + "delegated-lacnic-extended-latest"   // https://ftp.lacnic.net/pub/stats/lacnic/

// https://dev.maxmind.com/geoip/geoip2/geolite2/

func main() {
	profileStart()
	defer pprof.StopCPUProfile()

	t := time.Now()
	tree := NewTree(64)

	catchBreakSequenceForDebug(tree)

	counters = startCounting()

	ingest(tree, arinPath)
	ingest(tree, ripePath)
	ingest(tree, apnicPath)
	ingest(tree, afrinicPath)
	ingest(tree, lacnicPath)

	counters.ticker.Stop()

	fmt.Println("finished in", time.Since(t))

	f, _ := os.Create("output.json")
	f.Truncate(0)
	f.Seek(0, 0)
	f.WriteString(tree.JSON())
}

func profileStart() {
	cpuProf, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(cpuProf)
}

func catchBreakSequenceForDebug(tree *Tree) {
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
}

type tickerCounters struct {
	rate         uint64
	parentRate   uint64
	noParentRate uint64
	ticker       *time.Ticker
}

var counters tickerCounters

func startCounting() tickerCounters {
	counters = tickerCounters{
		ticker: time.NewTicker(time.Second),
	}
	go func() {
		var total uint64
		for range counters.ticker.C {
			old := atomic.SwapUint64(&counters.rate, 0)
			total += old
			fmt.Printf("%v count/sec  %v total  %v insertWithParent()  %v insertWithoutParent()\n",
				old, total, atomic.SwapUint64(&counters.parentRate, 0),
				atomic.SwapUint64(&counters.noParentRate, 0))
		}
	}()
	return counters
}
