// Package tsp 提供旅行商问题的城市数据、距离计算与内置测试实例。
// 支持两种边类型：att（ATT 伪欧氏距离，att48 基准）与 euclid（欧氏距离，随机实例）。
package tsp

import (
	"fmt"
	"math"
	"math/rand"
)

// City 是平面上的一座城市。
type City struct {
	ID int     `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

// Instance 是一个 TSP 实例。
type Instance struct {
	Name        string  `json:"name"`
	EdgeType    string  `json:"edgeType"`              // "att" 或 "euclid"
	Cities      []City  `json:"cities"`
	Optimal     float64 `json:"optimal"`               // 已知最优环游长度；0 表示未知
	OptimalTour []int   `json:"optimalTour,omitempty"` // 已知最优回路（用于可视化对比）
}

// Size 返回城市数量。
func (in *Instance) Size() int { return len(in.Cities) }

// DistanceMatrix 计算全量距离矩阵。
func (in *Instance) DistanceMatrix() [][]float64 {
	n := len(in.Cities)
	dm := make([][]float64, n)
	for i := range dm {
		dm[i] = make([]float64, n)
		for j := range dm[i] {
			dm[i][j] = in.edgeDistance(in.Cities[i], in.Cities[j])
		}
	}
	return dm
}

// TourLength 计算环游长度（含回到起点的闭合边）。
func (in *Instance) TourLength(tour []int) float64 {
	if err := in.ValidateTour(tour); err != nil {
		return math.NaN()
	}
	dm := in.DistanceMatrix()
	total := 0.0
	for i := 0; i < len(tour); i++ {
		total += dm[tour[i]][tour[(i+1)%len(tour)]]
	}
	return total
}

// ValidateTour 校验环游是恰好经过每座城市一次的排列。
func (in *Instance) ValidateTour(tour []int) error {
	n := len(in.Cities)
	if len(tour) != n {
		return fmt.Errorf("环游长度 %d 与城市数 %d 不一致", len(tour), n)
	}
	seen := make([]bool, n)
	for _, v := range tour {
		if v < 0 || v >= n {
			return fmt.Errorf("环游包含非法城市编号 %d", v)
		}
		if seen[v] {
			return fmt.Errorf("环游重复经过城市 %d", v)
		}
		seen[v] = true
	}
	return nil
}

// edgeDistance 按实例边类型计算两城距离。
func (in *Instance) edgeDistance(a, b City) float64 {
	if in.EdgeType == "att" {
		return attDistance(a, b)
	}
	return euclidDistance(a, b)
}

// attDistance 计算 TSPLIB ATT 伪欧氏距离：r=√((dx²+dy²)/10)，
// 四舍五入为 t，若 t<r 则取 t+1（向上取整效果）。
func attDistance(a, b City) float64 {
	dx, dy := a.X-b.X, a.Y-b.Y
	r := math.Sqrt((dx*dx + dy*dy) / 10.0)
	t := math.Round(r)
	if t < r {
		return t + 1
	}
	return t
}

// euclidDistance 计算平面欧氏距离。
func euclidDistance(a, b City) float64 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Random 生成随机欧氏实例：n 座城市均匀分布在 [0,100]² 内，种子固定结果可复现。
func Random(n int, seed int64) (*Instance, error) {
	if n < 4 {
		return nil, fmt.Errorf("城市数至少为 4，实际 %d", n)
	}
	if n > 500 {
		return nil, fmt.Errorf("城市数最多 500，实际 %d", n)
	}
	rng := rand.New(rand.NewSource(seed))
	cities := make([]City, n)
	for i := range cities {
		cities[i] = City{
			ID: i,
			X:  rng.Float64() * 100,
			Y:  rng.Float64() * 100,
		}
	}
	return &Instance{
		Name:     fmt.Sprintf("random-%d-%d", n, seed),
		EdgeType: "euclid",
		Cities:   cities,
	}, nil
}
