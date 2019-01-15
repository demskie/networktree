package main

import (
	"net"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/demskie/subnetmath"
)

type Node struct {
	Network     *net.IPNet
	GeoPosition *GeoPosition
	Parent      *Node
	Children    []*Node
}

// Tree contains the root nodes
type Tree struct {
	mtx       *sync.RWMutex
	sbuf      *subnetmath.Buffer
	Roots     []*Node
	RootsV6   []*Node
	Precision int
	Size      int
}

// NewTree creates a new Tree object
func NewTree(precision int) *Tree {
	return &Tree{
		mtx:       &sync.RWMutex{},
		sbuf:      subnetmath.NewBuffer(),
		Roots:     make([]*Node, 0),
		RootsV6:   make([]*Node, 0),
		Precision: precision,
		Size:      0,
	}
}

func (tree *Tree) insert(geoPosition *GeoPosition, networks ...*net.IPNet) {
	for _, network := range networks {
		var parent *Node
		if network.IP.To4() != nil {
			parent = tree.findClosestSupernet(network, tree.Roots)
		} else {
			parent = tree.findClosestSupernet(network, tree.RootsV6)
		}
		if parent != nil && subnetmath.NetworksAreIdentical(network, parent.Network) {
			atomic.AddUint64(&counters.parentRate, 1)
			if parent.GeoPosition == nil {
				parent.GeoPosition = geoPosition
			}
		} else {
			if parent != nil {
				atomic.AddUint64(&counters.parentRate, 1)
			} else {
				atomic.AddUint64(&counters.noParentRate, 1)
			}
			insertNode(tree, &Node{Network: network, GeoPosition: geoPosition, Parent: parent, Children: nil})
			tree.Size++
		}
	}
}

func insertNode(tree *Tree, newNode *Node) {
	if newNode.Parent != nil {
		for _, sibling := range newNode.Parent.Children {
			if tree.sbuf.NetworkContainsSubnet(newNode.Network, sibling.Network) {
				newNode.Children = append(newNode.Children, sibling)
			}
		}
		for _, newChild := range newNode.Children {
			newChild.Parent = newNode
			newNode.Parent.Children = tree.removeFromSortedNodes(newNode.Parent.Children, newChild)
		}
		newNode.Parent.Children = tree.insertIntoSortedNodes(newNode.Parent.Children, newNode)
		for len(newNode.Parent.Children) > tree.Precision {
			splitParent(newNode.Parent, tree)
		}
	} else if newNode.Network.IP.To4() != nil {
		for _, sibling := range tree.Roots {
			if tree.sbuf.NetworkContainsSubnet(newNode.Network, sibling.Network) {
				newNode.Children = append(newNode.Children, sibling)
			}
		}
		for _, newChild := range newNode.Children {
			newChild.Parent = newNode
			tree.Roots = tree.removeFromSortedNodes(tree.Roots, newChild)
		}
		tree.Roots = tree.insertIntoSortedNodes(tree.Roots, newNode)
		if len(tree.Roots) > tree.Precision {
			divideNodes(tree.Roots, tree)
		}
	} else {
		for _, sibling := range tree.RootsV6 {
			if tree.sbuf.NetworkContainsSubnet(newNode.Network, sibling.Network) {
				newNode.Children = append(newNode.Children, sibling)
			}
		}
		for _, newChild := range newNode.Children {
			newChild.Parent = newNode
			tree.RootsV6 = tree.removeFromSortedNodes(tree.RootsV6, newChild)
		}
		tree.RootsV6 = tree.insertIntoSortedNodes(tree.RootsV6, newNode)
		if len(tree.RootsV6) > tree.Precision {
			divideNodes(tree.RootsV6, tree)
		}
	}
}

func splitParent(parent *Node, tree *Tree) {
	copiedNetwork := subnetmath.DuplicateNetwork(parent.Network)
	smallerNetwork := subnetmath.ShrinkNetwork(copiedNetwork)
	if smallerNetwork != nil {
		tree.insert(nil, smallerNetwork, subnetmath.NextNetwork(smallerNetwork))
	}
}

func divideNodes(nodes []*Node, tree *Tree) {
	lastAddr := subnetmath.BroadcastAddr(nodes[len(nodes)-1].Network)
	subnets := tree.sbuf.FindInbetweenSubnets(nodes[0].Network.IP, lastAddr)
	tree.insert(nil, subnets...)
}

func (tree *Tree) insertIntoSortedNodes(slc []*Node, nd *Node) []*Node {
	idx := sort.Search(len(slc), func(i int) bool {
		return tree.sbuf.NetworkComesBefore(nd.Network, slc[i].Network)
	})
	slc = append(slc, &Node{})
	copy(slc[idx+1:], slc[idx:])
	slc[idx] = nd
	return slc
}

func (tree *Tree) removeFromSortedNodes(slc []*Node, nd *Node) []*Node {
	idx := sort.Search(len(slc), func(i int) bool {
		return slc[i].Network.Contains(nd.Network.IP) || tree.sbuf.NetworkComesBefore(nd.Network, slc[i].Network)
	})
	if slc[idx] == nd {
		copy(slc[idx:], slc[idx+1:])
		slc[len(slc)-1] = nil
		return slc[:len(slc)-1]
	}
	return slc
}

func (tree *Tree) findNetwork(address net.IP, nodes []*Node) *Node {
	idx := sort.Search(len(nodes), func(i int) bool {
		return nodes[i].Network.Contains(address) || tree.sbuf.AddressComesBefore(address, nodes[i].Network.IP)
	})
	if idx < len(nodes) {
		if nodes[idx].Children != nil && len(nodes[idx].Children) > 0 {
			return tree.findNetwork(address, nodes[idx].Children)
		}
		return nodes[idx]
	}
	return nil
}

func (tree *Tree) findClosestSupernet(network *net.IPNet, nodes []*Node) *Node {
	idx := sort.Search(len(nodes), func(i int) bool {
		return nodes[i].Network.Contains(network.IP) || tree.sbuf.NetworkComesBefore(network, nodes[i].Network)
	})
	if idx < len(nodes) && tree.sbuf.NetworkContainsSubnet(nodes[idx].Network, network) {
		if nodes[idx].Children != nil && len(nodes[idx].Children) > 0 {
			canidateSupernet := tree.findClosestSupernet(network, nodes[idx].Children)
			if canidateSupernet != nil {
				return canidateSupernet
			}
		}
		return nodes[idx]
	}
	return nil
}
