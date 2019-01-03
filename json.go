package main

import (
	"encoding/json"
	"fmt"
)

type nodeJSON struct {
	Network   string     `json:"network"`
	Country   string     `json:"country"`
	Latitude  string     `json:"latitude"`
	Longitude string     `json:"longitude"`
	Children  []nodeJSON `json:"children"`
}

func (t *Tree) JSON() string {
	var treeJSON []nodeJSON
	for _, n := range t.roots {
		treeJSON = append(treeJSON, buildJSON(n))
	}
	b, _ := json.MarshalIndent(&treeJSON, "", "  ")
	return string(b)
}

func buildJSON(n *node) nodeJSON {
	result := nodeJSON{}
	result.Network = n.network.String()
	result.Country = n.country
	if n.position != nil {
		result.Latitude = fmt.Sprintf("%f", n.position.latitude)
		result.Longitude = fmt.Sprintf("%f", n.position.longitude)
	}
	for _, child := range n.children {
		result.Children = append(result.Children, buildJSON(child))
	}
	return result
}
