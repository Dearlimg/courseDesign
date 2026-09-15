package campus

import (
	"fmt"
	"math"
	"testing"
)

func TestRoadDetourAndPath(t *testing.T) {
	m := Default()
	path, err := m.Path(2, 6)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(m.Distances[2][6]-280) > 0.001 || len(path) != 5 {
		t.Fatalf("missing road must detour: %v %v", path, m.Distances[2][6])
	}
	for i := range m.Places {
		if math.IsInf(m.Distances[0][i], 1) {
			t.Fatal("unreachable default place")
		}
	}
	m.Roads = []Road{}
	if err := m.Build(); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Path(0, 1); err == nil {
		t.Fatal("disconnected path accepted")
	}
}

func TestRoadsDoNotCrossRectangularBuildings(t *testing.T) {
	m := Default()
	var checked int
	for _, area := range m.Areas {
		if area.Kind != "building" && area.Kind != "dorm" && area.Kind != "food" {
			continue
		}
		var x, y, w, h, back float64
		count, _ := fmt.Sscanf(
			area.Path,
			"M%f %fh%fv%fh%fZ",
			&x, &y, &w, &h, &back,
		)
		if count != 5 {
			continue
		}
		checked++
		for _, road := range m.Roads {
			a, b := m.Nodes[road.From], m.Nodes[road.To]
			for i := 1; i < 100; i++ {
				fraction := float64(i) / 100
				px, py := a.X+(b.X-a.X)*fraction, a.Y+(b.Y-a.Y)*fraction
				inside := px > x && px < x+w && py > y && py < y+h
				if inside {
					t.Fatalf("road %d→%d crosses building %s", a.ID, b.ID, area.Path)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("building geometry was not checked")
	}
}

func TestEntrancesDifferFromJunctions(t *testing.T) {
	m := Map{
		Places: []Place{{ID: 0, NodeID: 1, X: 0, Y: 0}, {ID: 1, NodeID: 3, X: 20, Y: 0}},
		Nodes:  []Node{{ID: 0, X: 0, Y: 10}, {ID: 1, X: 0, Y: 0}, {ID: 2, X: 20, Y: 10}, {ID: 3, X: 20, Y: 0}},
		Roads:  []Road{{From: 1, To: 0, Meters: 10}, {From: 0, To: 2, Meters: 20}, {From: 2, To: 3, Meters: 10}},
	}
	if err := m.Build(); err != nil {
		t.Fatal(err)
	}
	path, err := m.Path(0, 1)
	if err != nil || len(path) != 4 || path[0] != 1 || path[3] != 3 || m.Distances[0][1] != 40 {
		t.Fatalf("entrances and junction indices mixed: %v %v", path, err)
	}
	m.Places[1].NodeID = 20
	if err := m.Build(); err == nil {
		t.Fatal("unknown entrance accepted")
	}
}

func TestWestCampusRouteGeometry(t *testing.T) {
	m := Default()
	if len(m.Nodes) <= len(m.Places) || len(m.Areas) == 0 {
		t.Fatal("missing detailed campus dataset")
	}
	// A direct gym-to-track chord crosses the sports ground; roads must go around it.
	gym, track := m.Places[8], m.Places[21]
	chord := math.Hypot(gym.X-track.X, gym.Y-track.Y) * m.MetersPerUnit
	if m.Distances[gym.ID][track.ID] <= chord*1.4 {
		t.Fatal("route cuts across the sports ground")
	}
	for _, from := range m.Places {
		for _, to := range m.Places {
			path, err := m.Path(from.ID, to.ID)
			if err != nil {
				t.Fatal(err)
			}
			var meters float64
			for i := 1; i < len(path); i++ {
				a, b := m.Nodes[path[i-1]], m.Nodes[path[i]]
				meters += math.Hypot(a.X-b.X, a.Y-b.Y) * m.MetersPerUnit
			}
			if math.Abs(meters-m.Distances[from.ID][to.ID]) > 0.00001 {
				t.Fatal("geometry and cost disagree")
			}
		}
	}
}
