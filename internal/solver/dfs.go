package solver

import (
	"time"

	"mazeweb/internal/maze"
)

// DFS 用深度优先搜索（显式栈实现回溯）求解迷宫。
// 注意：DFS 找到的是一条可行通路，不保证最短，用于与 BFS 对比。
func DFS(m *maze.Maze) Result {
	const algorithm = "dfs"
	start := time.Now()
	startPt, endPt := m.Start(), m.End()

	steps := []Step{}
	parent := map[maze.Point]maze.Point{}
	visited := map[maze.Point]bool{startPt: true}
	stack := []maze.Point{startPt}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		steps = append(steps, Step{Cell: cur, Action: "visit"})

		if cur == endPt {
			return finalize(m, steps, len(visited), parent, true, algorithm, start)
		}

		// 依次尝试四个方向的未访问邻格；全部失败则该步视为回溯（死路）。
		advanced := false
		for _, d := range directions {
			next := maze.Point{X: cur.X + d.X, Y: cur.Y + d.Y}
			if !m.Open(next) || visited[next] {
				continue
			}
			visited[next] = true
			parent[next] = cur
			stack = append(stack, next)
			advanced = true
		}
		if !advanced && len(stack) > 0 {
			steps = append(steps, Step{Cell: cur, Action: "backtrack"})
		}
	}
	return finalize(m, steps, len(visited), parent, false, algorithm, start)
}
