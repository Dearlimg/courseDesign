// Package solver 用深度优先（DFS）与广度优先（BFS）算法求解迷宫最短通路，
// 并记录逐步探索过程供前端动画回放。
package solver

import (
	"fmt"
	"time"

	"mazeweb/internal/maze"
)

// 邻格扩展顺序：上、下、左、右（固定顺序保证结果可复现）。
var directions = [4]maze.Point{
	{X: 0, Y: -1},
	{X: 0, Y: 1},
	{X: -1, Y: 0},
	{X: 1, Y: 0},
}

// Step 是探索过程中的一步：visit 表示访问新格子，backtrack 表示回溯。
type Step struct {
	Cell   maze.Point `json:"cell"`
	Action string     `json:"action"`
}

// Result 是一次求解的完整结果。
type Result struct {
	Found     bool         `json:"found"`
	Path      []maze.Point `json:"path"`
	Steps     []Step       `json:"steps"`
	Visited   int          `json:"visitedCount"`
	PathLen   int          `json:"pathLength"`
	ElapsedMs float64      `json:"elapsedMs"`
	Algorithm string       `json:"algorithm"`
}

// Solve 按算法名称分发求解。algorithm 取 "bfs" 或 "dfs"。
func Solve(m *maze.Maze, algorithm string) (Result, error) {
	if !m.Open(m.Start()) {
		return Result{}, fmt.Errorf("起点 (0,0) 是墙，无解")
	}
	if !m.Open(m.End()) {
		return Result{}, fmt.Errorf("终点 (%d,%d) 是墙，无解", m.End().X, m.End().Y)
	}
	switch algorithm {
	case "bfs":
		return BFS(m), nil
	case "dfs":
		return DFS(m), nil
	default:
		return Result{}, fmt.Errorf("未知算法 %q，仅支持 bfs / dfs", algorithm)
	}
}

// reconstruct 沿 parent 链从终点回溯出路径。
func reconstruct(parent map[maze.Point]maze.Point, m *maze.Maze) []maze.Point {
	path := []maze.Point{}
	for p, ok := m.End(), true; ok; p, ok = parent[p] {
		path = append(path, p)
		if p == m.Start() {
			break
		}
	}
	// 反转得到 起点→终点
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// finalize 组装结果：若 found 则回溯路径并计算统计。
func finalize(m *maze.Maze, steps []Step, visited int, parent map[maze.Point]maze.Point, found bool, algorithm string, start time.Time) Result {
	res := Result{
		Found:     found,
		Steps:     steps,
		Visited:   visited,
		Algorithm: algorithm,
		ElapsedMs: float64(time.Since(start).Microseconds()) / 1000.0,
	}
	if found {
		res.Path = reconstruct(parent, m)
		res.PathLen = len(res.Path) - 1
	} else {
		res.Path = []maze.Point{}
	}
	return res
}
