package campus

import (
	"math"
	"testing"
)

func TestRoadDetourAndPath(t *testing.T) {
	m := Default()
	path, err := m.Path(2, 6)
	if err != nil {
		t.Fatal(err)
	}
	if m.Distances[2][6] != 780 || len(path) != 4 {
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
