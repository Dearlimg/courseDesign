package solver

import (
	"time"

	"mazeweb/internal/maze"
)

// BFS 用广度优先搜索求解迷宫：按层扩展，首次到达终点即为最短路径。
func BFS(m *maze.Maze) Result {
	const algorithm = "bfs"
	start := time.Now()
	startPt, endPt := m.Start(), m.End()

	steps := []Step{}
	parent := map[maze.Point]maze.Point{}
	visited := map[maze.Point]bool{startPt: true}
	queue := []maze.Point{startPt}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		steps = append(steps, Step{Cell: cur, Action: "visit"})

		if cur == endPt {
			return finalize(m, steps, len(visited), parent, true, algorithm, start)
		}
		for _, d := range directions {
			next := maze.Point{X: cur.X + d.X, Y: cur.Y + d.Y}
			if !m.Open(next) || visited[next] {
				continue
			}
			visited[next] = true
			parent[next] = cur
			queue = append(queue, next)
		}
	}
	return finalize(m, steps, len(visited), parent, false, algorithm, start)
}
