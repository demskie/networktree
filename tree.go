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
	sbuf      *subnetmath.Buffer
	precision int
	size      int
}

// NewTree creates a new Tree object
func NewTree(precision int) *Tree {
	return &Tree{
		roots:     make([]*node, 0),
		sbuf:      subnetmath.NewBuffer(),
		precision: precision,
	}
}

func (tree *Tree) Size() int {
	return tree.size
}

func (tree *Tree) insert(networks []*net.IPNet, country string, position *Position) {
	for _, network := range networks {
		tree.size++
		parent := tree.findClosestSupernet(network, tree.roots)
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
			insertNode(tree, &node{network, country, position, parent, nil})
		}
	}
}

func insertNode(tree *Tree, newNode *node) {
	if newNode.parent != nil {
		for _, sibling := range newNode.parent.children {
			if tree.sbuf.NetworkContainsSubnet(newNode.network, sibling.network) {
				newNode.children = append(newNode.children, sibling)
			}
		}
		for _, newChild := range newNode.children {
			newChild.parent = newNode
			newNode.parent.children = tree.removeFromSortedNodes(newNode.parent.children, newChild)
		}
		newNode.parent.children = tree.insertIntoSortedNodes(newNode.parent.children, newNode)
		for len(newNode.parent.children) > tree.precision {
			splitParent(newNode.parent, tree)
		}
	} else {
		for _, sibling := range tree.roots {
			if tree.sbuf.NetworkContainsSubnet(newNode.network, sibling.network) {
				newNode.children = append(newNode.children, sibling)
			}
		}
		for _, newChild := range newNode.children {
			newChild.parent = newNode
			tree.roots = tree.removeFromSortedNodes(tree.roots, newChild)
		}
		tree.roots = tree.insertIntoSortedNodes(tree.roots, newNode)
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
		}, "ZZ", nil)
	}
}

func splitNodes(nodes []*node, tree *Tree) {
	lastAddr := subnetmath.BroadcastAddr(nodes[len(nodes)-1].network)
	subnets := subnetmath.FindInbetweenSubnets(nodes[0].network.IP, lastAddr)
	tree.insert(subnets, "ZZ", nil)
}

func (tree *Tree) insertIntoSortedNodes(slc []*node, nd *node) []*node {
	index := sort.Search(len(slc), func(i int) bool {
		return tree.sbuf.NetworkComesBefore(nd.network, slc[i].network)
	})
	slc = append(slc, &node{})
	copy(slc[index+1:], slc[index:])
	slc[index] = nd
	return slc
}

func (tree *Tree) removeFromSortedNodes(slc []*node, nd *node) []*node {
	index := sort.Search(len(slc), func(i int) bool {
		return slc[i].network.Contains(nd.network.IP) || tree.sbuf.NetworkComesBefore(nd.network, slc[i].network)
	})
	if slc[index] == nd {
		copy(slc[index:], slc[index+1:])
		slc[len(slc)-1] = nil
		return slc[:len(slc)-1]
	}
	return slc
}

func (tree *Tree) findNetwork(address net.IP, nodes []*node) *node {
	idx := sort.Search(len(nodes), func(i int) bool {
		return nodes[i].network.Contains(address) || tree.sbuf.AddressComesBefore(address, nodes[i].network.IP)
	})
	if idx < len(nodes) {
		if nodes[idx].children != nil && len(nodes[idx].children) > 0 {
			return tree.findNetwork(address, nodes[idx].children)
		}
		return nodes[idx]
	}
	return nil
}

func (tree *Tree) findClosestSupernet(network *net.IPNet, nodes []*node) *node {
	idx := sort.Search(len(nodes), func(i int) bool {
		return nodes[i].network.Contains(network.IP) || tree.sbuf.NetworkComesBefore(network, nodes[i].network)
	})
	if idx < len(nodes) && tree.sbuf.NetworkContainsSubnet(nodes[idx].network, network) {
		if nodes[idx].children != nil && len(nodes[idx].children) > 0 {
			canidateSupernet := tree.findClosestSupernet(network, nodes[idx].children)
			if canidateSupernet != nil {
				return canidateSupernet
			}
		}
		return nodes[idx]
	}
	return nil
}
