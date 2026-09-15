// Package campus models campus entrances and road junctions separately.
package campus

import (
	"fmt"
	"math"
)

type Place struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	NodeID   int     `json:"nodeId"`
	Category string  `json:"category"`
	LabelX   float64 `json:"labelX"`
	LabelY   float64 `json:"labelY"`
}

type Node struct {
	ID int     `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

type Area struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type Label struct {
	Text   string  `json:"text"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Rotate float64 `json:"rotate,omitempty"`
	Kind   string  `json:"kind"`
}

type Road struct {
	From   int     `json:"from"`
	To     int     `json:"to"`
	Meters float64 `json:"meters"`
	Kind   string  `json:"kind"`
}

type Map struct {
	Version       string      `json:"version"`
	Name          string      `json:"name"`
	Notice        string      `json:"notice"`
	Places        []Place     `json:"places"`
	Roads         []Road      `json:"roads"`
	Nodes         []Node      `json:"nodes"`
	Areas         []Area      `json:"areas"`
	Labels        []Label     `json:"labels"`
	Width         float64     `json:"width"`
	Height        float64     `json:"height"`
	MetersPerUnit float64     `json:"metersPerUnit"`
	SourceURL     string      `json:"sourceUrl"`
	Distances     [][]float64 `json:"-"`
	// DepotMeters 是各地点沿道路到取餐点（0 号地点）的距离（米），不可达记 -1；
	// 供前端在订单卡片上标注单笔配送距离，不参与任何算法计算。
	DepotMeters []float64 `json:"depotMeters"`
	next        [][]int
}

// Default reconstructs the west campus layout from the referenced campus map.
// Edge lengths are estimates, and do not assert current access permissions.
func Default() Map {
	m := westCampus()
	if err := m.Build(); err != nil {
		panic(err)
	}
	return m
}

// Build validates the undirected road graph and computes shortest paths.
func (m *Map) Build() error {
	n := len(m.Nodes)
	if n == 0 || n > 200 || len(m.Places) == 0 {
		return fmt.Errorf("道路节点数量须为 1～200，配送地点不能为空")
	}
	distances := make([][]float64, n)
	m.next = make([][]int, n)
	for i, p := range m.Nodes {
		if p.ID != i || !finite(p.X) || !finite(p.Y) {
			return fmt.Errorf("节点 ID 须连续且坐标有限")
		}
		distances[i] = make([]float64, n)
		m.next[i] = make([]int, n)
		for j := range n {
			distances[i][j], m.next[i][j] = math.Inf(1), -1
		}
		distances[i][i], m.next[i][i] = 0, i
	}
	for i, p := range m.Places {
		badNode := p.NodeID < 0 || p.NodeID >= n
		if p.ID != i || badNode {
			return fmt.Errorf("地点 ID 或入口节点非法")
		}
		entry := m.Nodes[p.NodeID]
		if p.X != entry.X || p.Y != entry.Y {
			return fmt.Errorf("地点须位于入口节点上")
		}
	}
	for _, r := range m.Roads {
		badNode := r.From < 0 || r.From >= n || r.To < 0 || r.To >= n
		badLength := r.Meters <= 0 || math.IsNaN(r.Meters) || math.IsInf(r.Meters, 0)
		if badNode || badLength {
			return fmt.Errorf("非法道路")
		}
		if r.Meters < distances[r.From][r.To] {
			distances[r.From][r.To], distances[r.To][r.From] = r.Meters, r.Meters
			m.next[r.From][r.To], m.next[r.To][r.From] = r.To, r.From
		}
	}
	for k := range n {
		for i := range n {
			for j := range n {
				if d := distances[i][k] + distances[k][j]; d < distances[i][j] {
					distances[i][j], m.next[i][j] = d, m.next[i][k]
				}
			}
		}
	}
	m.Distances = make([][]float64, len(m.Places))
	for i, from := range m.Places {
		m.Distances[i] = make([]float64, len(m.Places))
		for j, to := range m.Places {
			m.Distances[i][j] = distances[from.NodeID][to.NodeID]
		}
	}
	m.DepotMeters = make([]float64, len(m.Places))
	for i, d := range m.Distances[0] {
		if math.IsInf(d, 0) {
			m.DepotMeters[i] = -1
			continue
		}
		m.DepotMeters[i] = d
	}
	return nil
}

func (m Map) Path(from, to int) ([]int, error) {
	n := len(m.Places)
	if from < 0 || to < 0 || from >= n || to >= n {
		return []int{}, fmt.Errorf("未知地点")
	}
	from, to = m.Places[from].NodeID, m.Places[to].NodeID
	if len(m.next) != len(m.Nodes) {
		return []int{}, fmt.Errorf("路网尚未构建")
	}
	if m.next[from][to] < 0 {
		return []int{}, fmt.Errorf("地点不可达")
	}
	path := []int{from}
	for from != to {
		from = m.next[from][to]
		path = append(path, from)
	}
	return path, nil
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
