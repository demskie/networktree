package main

import (
	"fmt"
	"net"
	"testing"

	"github.com/demskie/subnetmath"
)

var benchTree32 *Tree

func createBenchTree32() {
	if benchTree32 == nil {
		ticker.Stop()
		benchTree32 = NewTree(25)
		ingest(benchTree32, arinPath)
		//ingest(benchTree32, ripePath)
		// 204.29.8.0/23
		// 204.29.10.0/24
		fmt.Println(findClosestSupernet(subnetmath.ParseNetworkCIDR("204.29.8.0/23"), benchTree32.roots).network.String())
	}
}

func BenchmarkFindNetwork(b *testing.B) {
	createBenchTree32()
	addr := net.ParseIP("8.8.8.8")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findNetwork(addr, benchTree32.roots)
	}
}

func BenchmarkFindClosestSupernet(b *testing.B) {
	createBenchTree32()
	network := subnetmath.ParseNetworkCIDR("8.8.8.8/8")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findClosestSupernet(network, benchTree32.roots)
	}
}
