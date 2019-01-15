package main

import (
	"encoding/json"
	"fmt"
)

type nodeJSON struct {
	Network     string     `json:"network"`
	CityName    string     `json:"cityName"`
	SubdivName  string     `json:"subdivName"`
	CountryISO  string     `json:"countryISO"`
	CountryName string     `json:"countryName"`
	IsPartOfEU  string     `json:"isPartOfEU"`
	Latitude    string     `json:"latitude"`
	Longitude   string     `json:"longitude"`
	Children    []nodeJSON `json:"children"`
}

func (t *Tree) JSON() string {
	var treeJSON []nodeJSON
	for _, r := range [][]*Node{t.Roots, t.RootsV6} {
		for _, n := range r {
			treeJSON = append(treeJSON, buildJSON(n))
		}
	}
	b, _ := json.MarshalIndent(&treeJSON, "", "  ")
	return string(b)
}

func buildJSON(n *Node) nodeJSON {
	result := nodeJSON{}
	result.Network = n.Network.String()
	if n.GeoPosition != nil {
		if n.GeoPosition.Location != nil {
			result.CityName = n.GeoPosition.Location.CityName
			result.SubdivName = n.GeoPosition.Location.SubdivName
			result.CountryISO = n.GeoPosition.Location.CountryISO
			result.CountryName = n.GeoPosition.Location.CountryName
			result.IsPartOfEU = fmt.Sprintf("%t", n.GeoPosition.Location.IsPartOfEU)
		}
		result.Latitude = fmt.Sprintf("%f", n.GeoPosition.Latitude)
		result.Longitude = fmt.Sprintf("%f", n.GeoPosition.Longitude)
	}
	for _, child := range n.Children {
		result.Children = append(result.Children, buildJSON(child))
	}
	return result
}
