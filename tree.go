package main

import (
	"net"
	"sort"

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
	roots []*node
	size  int
}

// NewTree creates a new Tree object
func NewTree() *Tree {
	return &Tree{
		roots: make([]*node, 0),
	}
}

func (tree *Tree) Length() int {
	return len(tree.roots)
}

func (tree *Tree) Size() int {
	return tree.size
}

func (tree *Tree) insertIPv4(subnets []*net.IPNet, country string, position *Position) {
	for _, network := range subnets {
		tree.size++
		newNode := &node{
			network:  network,
			country:  country,
			position: position,
			parent:   getDeepestParent(network, tree.roots),
		}
		if newNode.parent != nil {
			insertWithParent(newNode, tree)
		} else {
			insertWithoutParent(newNode, tree)
		}
	}
}

func insertWithParent(newNode *node, tree *Tree) {
	// deletions will need to occur outside the upcoming loops to avoid corruption
	var relocatedNodes []*node
	// sweep through adjacent children
	for _, sibling := range newNode.parent.children {
		// see if we should be their parent
		if newNode.network.Contains(sibling.network.IP) {
			// remove child from previous parent
			relocatedNodes = append(relocatedNodes, sibling)
			// make ourselves the parent
			sibling.parent = newNode
			newNode.children = insertIntoSortedNodes(newNode.children, sibling)
		}
	}
	// remove any nodes that were moved away from their original parent
	if relocatedNodes != nil {
		for _, sibling := range relocatedNodes {
			newNode.parent.children = removeNodeFromSlice(newNode.parent.children, sibling)
		}
	}
	// add ourselves to the parent we found
	newNode.parent.children = insertIntoSortedNodes(newNode.parent.children, newNode)
}

func insertWithoutParent(newNode *node, tree *Tree) {
	// deletions will need to occur outside the upcoming loops to avoid corruption
	var relocatedNodes []*node
	// sweep through existing subnets without a parent
	for _, otherNode := range tree.roots {
		// check if this other node should be our child
		if newNode.network.Contains(otherNode.network.IP) {
			// remove this node from the base of the tree
			relocatedNodes = append(relocatedNodes, otherNode)
			// make ourselves the parent
			otherNode.parent = newNode
			newNode.children = insertIntoSortedNodes(newNode.children, otherNode)
		}
	}
	// remove any nodes that were moved out from the base of the tree
	if relocatedNodes != nil {
		for _, otherNode := range relocatedNodes {
			tree.roots = removeNodeFromSlice(tree.roots, otherNode)
		}
	}
	// add ourselves to the base of the tree
	tree.roots = insertIntoSortedNodes(tree.roots, newNode)
}

func getDeepestParent(orig *net.IPNet, parents []*node) (parent *node) {
	for _, nd := range parents {
		snMask, _ := nd.network.Mask.Size()
		origMask, _ := orig.Mask.Size()
		if snMask < origMask && nd.network.Contains(orig.IP) {
			deeper := getDeepestParent(orig, nd.children)
			if deeper != nil {
				return deeper
			}
			return nd
		}
	}
	return nil
}

func insertIntoSortedNodes(slc []*node, nd *node) []*node {
	index := sort.Search(len(slc), func(i int) bool {
		return subnetmath.NetworkComesBefore(slc[i].network, nd.network)
	})
	slc = append(slc, &node{})
	copy(slc[index+1:], slc[index:])
	slc[index] = nd
	return slc
}

// BUG: do we need to offset this index in case of an exact match?
func removeFromSortedNodes(slc []*node, nd *node) []*node {
	index := sort.Search(len(slc), func(i int) bool {
		return subnetmath.NetworkComesBefore(slc[i].network, nd.network)
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
