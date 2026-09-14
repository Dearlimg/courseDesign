package solver

import (
	"testing"

	"mazeweb/internal/maze"
)

// 题目十二给出的示例迷宫：1 表示墙，0 表示通路
var presetGrid = [][]int{
	{0, 1, 0, 0, 1},
	{0, 0, 0, 0, 0},
	{1, 0, 0, 0, 1},
	{0, 1, 0, 1, 1},
	{0, 1, 0, 0, 0},
}

// 起点在左上、终点在右下，且存在一条 8 步最优路径的迷宫（曼哈顿距离为 8）
var craftedGrid = [][]int{
	{0, 0, 0, 0, 0, 0},
	{0, 1, 1, 1, 1, 0},
	{0, 1, 0, 0, 0, 0},
	{0, 0, 0, 1, 1, 0},
}

func mustMaze(t *testing.T, grid [][]int) *maze.Maze {
	t.Helper()
	m, err := maze.New(grid)
	if err != nil {
		t.Fatalf("构造迷宫失败: %v", err)
	}
	return m
}

// checkValidPath 校验路径：起终点正确、逐格相邻且均为通路、无重复格子。
func checkValidPath(t *testing.T, m *maze.Maze, path []maze.Point) {
	t.Helper()
	if len(path) == 0 {
		t.Fatal("路径不应为空")
	}
	start, end := m.Start(), m.End()
	if path[0] != start {
		t.Fatalf("路径应以起点 %v 开始，实际 %v", start, path[0])
	}
	if path[len(path)-1] != end {
		t.Fatalf("路径应以终点 %v 结束，实际 %v", end, path[len(path)-1])
	}
	seen := map[maze.Point]bool{}
	for i, p := range path {
		if !m.Open(p) {
			t.Fatalf("路径第 %d 格 %v 是墙", i, p)
		}
		if seen[p] {
			t.Fatalf("路径重复经过 %v", p)
		}
		seen[p] = true
		if i == 0 {
			continue
		}
		prev := path[i-1]
		dx, dy := p.X-prev.X, p.Y-prev.Y
		if dx*dx+dy*dy != 1 {
			t.Fatalf("路径第 %d 格 %v 与前一格 %v 不相邻", i, p, prev)
		}
	}
}

func TestBFSFindsShortestPathOnPreset(t *testing.T) {
	m := mustMaze(t, presetGrid)
	res := BFS(m)
	if !res.Found {
		t.Fatal("BFS 应在题目示例迷宫中找到通路")
	}
	if len(res.Path) != 9 {
		t.Fatalf("最短路径应经过 9 个格子，实际 %d", len(res.Path))
	}
	if res.PathLen != 8 {
		t.Fatalf("最短路径应为 8 步，实际 %d", res.PathLen)
	}
	checkValidPath(t, m, res.Path)
	if res.Visited < len(res.Path) {
		t.Fatalf("访问格子数 (%d) 不应少于路径长度 (%d)", res.Visited, len(res.Path))
	}
	if len(res.Steps) == 0 {
		t.Fatal("应记录探索过程步骤")
	}
}

func TestBFSPathIsOptimalOnCraftedMaze(t *testing.T) {
	m := mustMaze(t, craftedGrid)
	res := BFS(m)
	if !res.Found {
		t.Fatal("BFS 应找到通路")
	}
	// 曼哈顿距离为 3+5=8，存在单调最短路径，因此最优解恰为 8 步
	if res.PathLen != 8 {
		t.Fatalf("BFS 应给出 8 步最优路径，实际 %d", res.PathLen)
	}
}

func TestDFSFindsValidPathOnPreset(t *testing.T) {
	m := mustMaze(t, presetGrid)
	res := DFS(m)
	if !res.Found {
		t.Fatal("DFS 应在题目示例迷宫中找到通路")
	}
	checkValidPath(t, m, res.Path)
	if len(res.Steps) == 0 {
		t.Fatal("应记录探索过程步骤")
	}
}

func TestBFSNeverLongerThanDFS(t *testing.T) {
	// 多个种子下的随机迷宫，BFS 路径长度不应超过 DFS
	for seed := int64(1); seed <= 10; seed++ {
		m, err := maze.Random(9, 9, 0.3, seed)
		if err != nil {
			t.Fatalf("生成迷宫失败: %v", err)
		}
		bfsRes, dfsRes := BFS(m), DFS(m)
		if bfsRes.Found != dfsRes.Found {
			t.Fatalf("种子 %d：BFS 与 DFS 的可达性应一致", seed)
		}
		if bfsRes.Found && bfsRes.PathLen > dfsRes.PathLen {
			t.Fatalf("种子 %d：BFS 路径 (%d 步) 不应长于 DFS (%d 步)",
				seed, bfsRes.PathLen, dfsRes.PathLen)
		}
	}
}

func TestNoPathWhenBlocked(t *testing.T) {
	grid := [][]int{
		{0, 1},
		{1, 0},
	}
	m := mustMaze(t, grid)
	for name, fn := range map[string]func(*maze.Maze) Result{"BFS": BFS, "DFS": DFS} {
		res := fn(m)
		if res.Found {
			t.Fatalf("%s 不应在无解迷宫中找到路径", name)
		}
		if len(res.Path) != 0 {
			t.Fatalf("%s 无解时路径应为空", name)
		}
	}
}

func TestErrorWhenStartOrEndIsWall(t *testing.T) {
	grid := [][]int{
		{1, 0},
		{0, 0},
	}
	m := mustMaze(t, grid)
	if _, err := Solve(m, "bfs"); err == nil {
		t.Fatal("起点为墙时应返回错误")
	}
}

func TestSolveDispatchesAlgorithm(t *testing.T) {
	m := mustMaze(t, presetGrid)
	res, err := Solve(m, "bfs")
	if err != nil || !res.Found {
		t.Fatalf("Solve(bfs) 失败: found=%v err=%v", res.Found, err)
	}
	res, err = Solve(m, "dfs")
	if err != nil || !res.Found {
		t.Fatalf("Solve(dfs) 失败: found=%v err=%v", res.Found, err)
	}
	if _, err := Solve(m, "astar"); err == nil {
		t.Fatal("未知算法应返回错误")
	}
}
