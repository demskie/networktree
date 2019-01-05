package main

import (
	"fmt"
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
		if len(newNode.parent.children) > tree.precision {
			splitNodes(newNode.parent.children, tree)
		}
	} else {
		tree.roots = insertIntoSortedNodes(tree.roots, newNode)
		if len(tree.roots) > tree.precision {
			splitNodes(tree.roots, tree)
		}
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

// searchedIndex: 11	 actualIndex: 7	 length: 11
// >>>204.29.8.0/23<<< 3.0.0.0/8 4.0.0.0/6 8.0.0.0/5 16.0.0.0/4 32.0.0.0/3 64.0.0.0/2 128.0.0.0/2 192.0.0.0/4 208.0.0.0/5 216.0.0.0/8 217.147.184.0/21

func findClosestSupernet(network *net.IPNet, nodes []*node) *node {
	searchedIndex := sort.Search(len(nodes), func(i int) bool {
		return subnetmath.NetworkComesBefore(network, nodes[i].network) ||
			subnetmath.NetworkContainsSubnet(nodes[i].network, network)
	})
	actualIndex := 0
	for _, nd := range nodes {
		if subnetmath.NetworkContainsSubnet(nd.network, network) {
			if searchedIndex != actualIndex {
				fmt.Printf("searchedIndex: %v\t actualIndex: %v\t length: %v\n",
					searchedIndex, actualIndex, len(nodes))
				s := ">>>" + network.String() + "<<<"
				for _, val := range nodes {
					s += " " + val.network.String()
				}
				fmt.Println(s + "\n")
			}
			if nd.children != nil && len(nd.children) > 0 {
				canidateSupernet := findClosestSupernet(network, nd.children)
				if canidateSupernet != nil {
					return canidateSupernet
				}
				return nd
			}
			return nd
		}
		actualIndex++
	}
	return nil
}
