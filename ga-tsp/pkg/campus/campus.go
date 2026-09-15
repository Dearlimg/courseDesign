// Package campus provides a versioned simulation road network, not real navigation.
package campus

import (
	"fmt"
	"math"
)

type Place struct {
	ID   int     `json:"id"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type Road struct {
	From   int     `json:"from"`
	To     int     `json:"to"`
	Meters float64 `json:"meters"`
}

type Map struct {
	Version   string      `json:"version"`
	Name      string      `json:"name"`
	Notice    string      `json:"notice"`
	Places    []Place     `json:"places"`
	Roads     []Road      `json:"roads"`
	Distances [][]float64 `json:"-"`
	next      [][]int
}

// Default uses fictional locations and measured-in-meters simulation edges.
// A verified campus survey can replace the data without changing the algorithms.
func Default() Map {
	m := Map{
		Version: "xupt-simulation-v1", Name: "西安邮电大学 · 校园配送仿真",
		Notice: "长安校区场景示意；地点、道路与长度为仿真数据，未经实地核实，不用于真实导航。",
		Places: []Place{}, Roads: []Road{},
	}
	names := []string{"统一取餐点", "宿舍区 A", "宿舍区 B", "宿舍区 C", "教学区 A", "图书馆配送点",
		"教学区 B", "实验区 A", "宿舍区 D", "宿舍区 E", "活动区配送点", "实验区 B"}
	for i, name := range names {
		m.Places = append(m.Places, Place{ID: i, Name: name, X: float64(i%4)*240 + 80, Y: float64(i/4)*300 + 80})
	}
	for i := range names {
		if i%4 < 3 {
			m.Roads = append(m.Roads, Road{From: i, To: i + 1, Meters: 240})
		}
		if i < 8 && i != 2 {
			m.Roads = append(m.Roads, Road{From: i, To: i + 4, Meters: 300})
		}
	}
	if err := m.Build(); err != nil {
		panic(err)
	}
	return m
}

// Build validates the undirected road graph and computes shortest paths.
func (m *Map) Build() error {
	n := len(m.Places)
	if n == 0 || n > 100 {
		return fmt.Errorf("地图地点数量须为 1～100")
	}
	m.Distances = make([][]float64, n)
	m.next = make([][]int, n)
	for i, p := range m.Places {
		if p.ID != i {
			return fmt.Errorf("地点 ID 须连续")
		}
		m.Distances[i] = make([]float64, n)
		m.next[i] = make([]int, n)
		for j := range n {
			m.Distances[i][j], m.next[i][j] = math.Inf(1), -1
		}
		m.Distances[i][i], m.next[i][i] = 0, i
	}
	for _, r := range m.Roads {
		badNode := r.From < 0 || r.From >= n || r.To < 0 || r.To >= n
		badLength := r.Meters <= 0 || math.IsNaN(r.Meters) || math.IsInf(r.Meters, 0)
		if badNode || badLength {
			return fmt.Errorf("非法道路")
		}
		if r.Meters < m.Distances[r.From][r.To] {
			m.Distances[r.From][r.To], m.Distances[r.To][r.From] = r.Meters, r.Meters
			m.next[r.From][r.To], m.next[r.To][r.From] = r.To, r.From
		}
	}
	for k := range n {
		for i := range n {
			for j := range n {
				if d := m.Distances[i][k] + m.Distances[k][j]; d < m.Distances[i][j] {
					m.Distances[i][j], m.next[i][j] = d, m.next[i][k]
				}
			}
		}
	}
	return nil
}

func (m Map) Path(from, to int) ([]int, error) {
	n := len(m.Places)
	if from < 0 || to < 0 || from >= n || to >= n {
		return []int{}, fmt.Errorf("未知地点")
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
