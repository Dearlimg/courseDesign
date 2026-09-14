// Package maze 提供迷宫的数据结构、校验、随机生成与题目示例。
// 迷宫约定：二维数组中 1 表示墙壁，0 表示通路，起点为左上角，终点为右下角。
package maze

import (
	"errors"
	"fmt"
	"math/rand"
)

// Point 表示迷宫中的一个格子坐标，X 为列、Y 为行。
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Maze 是矩形迷宫，Grid[y][x] 为 0（通路）或 1（墙壁）。
type Maze struct {
	Grid [][]int
}

// New 校验并构造迷宫：必须非空、矩形、且只含 0/1。
func New(grid [][]int) (*Maze, error) {
	if len(grid) == 0 {
		return nil, errors.New("迷宫不能为空")
	}
	cols := len(grid[0])
	if cols == 0 {
		return nil, errors.New("迷宫不能为空")
	}
	for y, row := range grid {
		if len(row) != cols {
			return nil, fmt.Errorf("迷宫第 %d 行长度 %d 与首行 %d 不一致，必须为矩形", y, len(row), cols)
		}
		for x, v := range row {
			if v != 0 && v != 1 {
				return nil, fmt.Errorf("迷宫 (%d,%d) 处的值 %d 非法，只能为 0（通路）或 1（墙）", x, y, v)
			}
		}
	}
	return &Maze{Grid: grid}, nil
}

// Rows 返回行数。
func (m *Maze) Rows() int { return len(m.Grid) }

// Cols 返回列数。
func (m *Maze) Cols() int { return len(m.Grid[0]) }

// Open 判断某格是否在迷宫内且为通路。
func (m *Maze) Open(p Point) bool {
	return p.Y >= 0 && p.Y < m.Rows() && p.X >= 0 && p.X < m.Cols() && m.Grid[p.Y][p.X] == 0
}

// Start 返回起点（左上角）。
func (m *Maze) Start() Point { return Point{X: 0, Y: 0} }

// End 返回终点（右下角）。
func (m *Maze) End() Point { return Point{X: m.Cols() - 1, Y: m.Rows() - 1} }

// Preset 返回课程设计指导书题目十二给出的 5x5 示例迷宫。
func Preset() *Maze {
	return &Maze{Grid: [][]int{
		{0, 1, 0, 0, 1},
		{0, 0, 0, 0, 0},
		{1, 0, 0, 0, 1},
		{0, 1, 0, 1, 1},
		{0, 1, 0, 0, 0},
	}}
}

// Random 生成随机迷宫：每个格子以 density 的概率成为墙，起点与终点强制为通路。
// seed 固定时结果确定，便于复现实验。
func Random(cols, rows int, density float64, seed int64) (*Maze, error) {
	if cols <= 0 || rows <= 0 {
		return nil, fmt.Errorf("迷宫尺寸必须为正数，实际 %dx%d", cols, rows)
	}
	if cols > 100 || rows > 100 {
		return nil, fmt.Errorf("迷宫尺寸过大（最大 100x100），实际 %dx%d", cols, rows)
	}
	if density < 0 || density > 1 {
		return nil, fmt.Errorf("墙密度必须在 [0,1] 之间，实际 %f", density)
	}
	rng := rand.New(rand.NewSource(seed))
	grid := make([][]int, rows)
	for y := range grid {
		grid[y] = make([]int, cols)
		for x := range grid[y] {
			if rng.Float64() < density {
				grid[y][x] = 1
			}
		}
	}
	grid[0][0] = 0
	grid[rows-1][cols-1] = 0
	return &Maze{Grid: grid}, nil
}
