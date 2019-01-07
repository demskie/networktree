package main

import (
	"net"
	"sort"
	"sync/atomic"

	"github.com/demskie/subnetmath"
)

type node struct {
	network  *net.IPNet
	country  string
	position *Position
	parent   *node
	children []*node
}

// Tree contains the root nodes
type Tree struct {
	roots     []*node
	precision int
	size      int
}

// NewTree creates a new Tree object
func NewTree(precision int) *Tree {
	return &Tree{
		roots:     make([]*node, 0),
		precision: precision,
	}
}

func (tree *Tree) Size() int {
	return tree.size
}

func (tree *Tree) insert(networks []*net.IPNet, country string, position *Position) {
	for _, network := range networks {
		tree.size++
		parent := findClosestSupernet(network, tree.roots)
		if parent != nil && subnetmath.NetworksAreIdentical(network, parent.network) {
			atomic.AddUint64(&parentRate, 1)
			if parent.country == "ZZ" {
				parent.country = country
				parent.position = position
			}
		} else {
			if parent != nil {
				atomic.AddUint64(&parentRate, 1)
			} else {
				atomic.AddUint64(&noParentRate, 1)
			}
			newNode := &node{network, country, position, parent, nil}
			insertNode(tree, newNode)
		}
	}
}

func insertNode(tree *Tree, newNode *node) {

	if newNode.parent != nil {
		for _, sibling := range newNode.parent.children {
			if subnetmath.NetworkContainsSubnet(newNode.network, sibling.network) {
				newNode.children = append(newNode.children, sibling)
			}
		}
	} else {
		for _, sibling := range tree.roots {
			if subnetmath.NetworkContainsSubnet(newNode.network, sibling.network) {
				newNode.children = append(newNode.children, sibling)
			}
		}
	}

	if newNode.parent != nil {
		for _, newChild := range newNode.children {
			newChild.parent = newNode
			newNode.parent.children = removeNodeFromSlice(newNode.parent.children, newChild)
		}
	} else {
		for _, newChild := range newNode.children {
			newChild.parent = newNode
			tree.roots = removeNodeFromSlice(tree.roots, newChild)
		}
	}

	if newNode.parent != nil {
		newNode.parent.children = insertIntoSortedNodes(newNode.parent.children, newNode)
		for len(newNode.parent.children) > tree.precision {
			splitParent(newNode.parent, tree)
		}
	} else {
		tree.roots = insertIntoSortedNodes(tree.roots, newNode)
		if len(tree.roots) > tree.precision {
			splitNodes(tree.roots, tree)
		}
	}
}

func splitParent(parent *node, tree *Tree) {
	copiedNetwork := subnetmath.DuplicateNetwork(parent.network)
	smallerNetwork := subnetmath.ShrinkNetwork(copiedNetwork)
	if smallerNetwork != nil {
		tree.insert([]*net.IPNet{
			smallerNetwork,
			subnetmath.NextNetwork(smallerNetwork),
		}, "ZZZZZZ", nil)
	}
}

func splitNodes(nodes []*node, tree *Tree) {
	first := nodes[0].network
	last := nodes[len(nodes)-1].network
	// BUG: v6 is unsupported/untested at the moment
	if first.IP.To4() == nil || last.IP.To4() == nil {
		return
	}
	lastAddr := subnetmath.BroadcastAddr(last)
	subnets := subnetmath.FindInbetweenSubnets(first.IP, lastAddr)
	tree.insert(subnets, "ZZZZZZ", nil)
}

func insertIntoSortedNodes(slc []*node, nd *node) []*node {
	index := sort.Search(len(slc), func(i int) bool {
		// BUG: is the NetworkComesBefore logic backwards?
		return subnetmath.NetworkComesBefore(nd.network, slc[i].network)
	})
	slc = append(slc, &node{})
	copy(slc[index+1:], slc[index:])
	slc[index] = nd
	return slc
}

// BUG: do we need to offset this index in case of an exact match?
func removeFromSortedNodes(slc []*node, nd *node) []*node {
	index := sort.Search(len(slc), func(i int) bool {
		// BUG: is the NetworkComesBefore logic backwards?
		return subnetmath.NetworkComesBefore(nd.network, slc[i].network)
	})
	if slc[index] == nd {
		copy(slc[index:], slc[index+1:])
		slc[len(slc)-1] = nil
		return slc[:len(slc)-1]
	}
	return slc
}

// BUG: need to remove this and use removeFromSortedNodes!
func removeNodeFromSlice(slc []*node, nd *node) []*node {
	for i := range slc {
		if slc[i] == nd {
			copy(slc[i:], slc[i+1:])
			slc[len(slc)-1] = nil
			return slc[:len(slc)-1]
		}
	}
	return slc
}

func findNetwork(address net.IP, nodes []*node) *node {
	for _, nd := range nodes {
		if nd.network.Contains(address) {
			if nd.children != nil && len(nd.children) > 0 {
				return findNetwork(address, nd.children)
			}
			return nd
		}
	}
	return nil
}

func findClosestSupernet(network *net.IPNet, nodes []*node) *node {
	idx := sort.Search(len(nodes), func(i int) bool {
		return subnetmath.NetworkComesBefore(network, nodes[i].network) ||
			subnetmath.NetworkContainsSubnet(nodes[i].network, network)
	})
	for idx < len(nodes) {
		if subnetmath.NetworkContainsSubnet(nodes[idx].network, network) {
			if nodes[idx].children != nil && len(nodes[idx].children) > 0 {
				canidateSupernet := findClosestSupernet(network, nodes[idx].children)
				if canidateSupernet != nil {
					return canidateSupernet
				}
			}
			return nodes[idx]
		}
		idx++
	}
	return nil
}
