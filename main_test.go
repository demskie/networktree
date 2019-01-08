package main

import (
	"net"
	"reflect"
	"testing"

	"github.com/demskie/subnetmath"
)

var benchTree32 *Tree

func createBenchTree32() {
	if benchTree32 == nil {
		ticker.Stop()
		benchTree32 = NewTree(32)
		ingest(benchTree32, arinPath)
		ingest(benchTree32, ripePath)
	}
}

func TestClosestSupernet(t *testing.T) {
	nodes := []*node{
		&node{network: subnetmath.ParseNetworkCIDR("3.0.0.0/8")},
		&node{network: subnetmath.ParseNetworkCIDR("4.0.0.0/6")},
		&node{network: subnetmath.ParseNetworkCIDR("8.0.0.0/5")},
		&node{network: subnetmath.ParseNetworkCIDR("16.0.0.0/4")},
		&node{network: subnetmath.ParseNetworkCIDR("32.0.0.0/3")},
		&node{network: subnetmath.ParseNetworkCIDR("64.0.0.0/2")},
		&node{network: subnetmath.ParseNetworkCIDR("128.0.0.0/2")},
		&node{network: subnetmath.ParseNetworkCIDR("192.0.0.0/4")},
		&node{network: subnetmath.ParseNetworkCIDR("208.0.0.0/5")},
		&node{network: subnetmath.ParseNetworkCIDR("216.0.0.0/8")},
		&node{network: subnetmath.ParseNetworkCIDR("217.147.184.0/21")},
	}
	result := findClosestSupernet(subnetmath.ParseNetworkCIDR("204.29.8.0/23"), nodes)
	if !reflect.DeepEqual(result.network, subnetmath.ParseNetworkCIDR("192.0.0.0/4")) {
		t.Error(result.network, "does not equal", subnetmath.ParseNetworkCIDR("192.0.0.0/4"))
	}
}

func BenchmarkFindNetwork(b *testing.B) {
	createBenchTree32()
	addr := net.ParseIP("185.48.252.0")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findNetwork(addr, benchTree32.roots)
	}
}

func BenchmarkFindClosestSupernet(b *testing.B) {
	createBenchTree32()
	network := subnetmath.ParseNetworkCIDR("185.48.252.0/22")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findClosestSupernet(network, benchTree32.roots)
	}
}
