package campus

import (
	_ "embed" // Embed the versioned campus dataset alongside its graph implementation.
	"encoding/json"
	"math"
)

//go:embed west.json
var westData []byte

type dataset struct {
	Map
	// RoadChains reference node IDs; every bend is a routable graph junction.
	RoadChains [][]int `json:"roadChains"`
}

func westCampus() Map {
	var data dataset
	if err := json.Unmarshal(westData, &data); err != nil {
		panic(err)
	}
	m := data.Map
	m.Roads = []Road{}
	for _, chain := range data.RoadChains {
		for i := 1; i < len(chain); i++ {
			a, b := m.Nodes[chain[i-1]], m.Nodes[chain[i]]
			m.Roads = append(m.Roads, Road{
				From: a.ID, To: b.ID, Kind: "campus",
				Meters: math.Hypot(a.X-b.X, a.Y-b.Y) * m.MetersPerUnit,
			})
		}
	}
	return m
}
