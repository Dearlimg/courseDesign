package maze

import "testing"

// 题目十二给出的示例迷宫：1 表示墙，0 表示通路
var presetGrid = [][]int{
	{0, 1, 0, 0, 1},
	{0, 0, 0, 0, 0},
	{1, 0, 0, 0, 1},
	{0, 1, 0, 1, 1},
	{0, 1, 0, 0, 0},
}

func TestNewRejectsEmptyGrid(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("空迷宫应返回错误")
	}
	if _, err := New([][]int{}); err == nil {
		t.Fatal("空迷宫应返回错误")
	}
}

func TestNewRejectsNonRectangularGrid(t *testing.T) {
	grid := [][]int{
		{0, 1, 0},
		{0, 0},
	}
	if _, err := New(grid); err == nil {
		t.Fatal("非矩形迷宫应返回错误")
	}
}

func TestNewRejectsInvalidValues(t *testing.T) {
	grid := [][]int{
		{0, 2},
		{0, 0},
	}
	if _, err := New(grid); err == nil {
		t.Fatal("包含非 0/1 值的迷宫应返回错误")
	}
}

func TestPresetMatchesAssignment(t *testing.T) {
	m := Preset()
	if m.Rows() != 5 || m.Cols() != 5 {
		t.Fatalf("预置迷宫应为 5x5，实际 %dx%d", m.Rows(), m.Cols())
	}
	for y, row := range presetGrid {
		for x, v := range row {
			if got := m.Grid[y][x]; got != v {
				t.Fatalf("预置迷宫 (%d,%d) 处应为 %d，实际 %d", x, y, v, got)
			}
		}
	}
	if !m.Open(m.Start()) || !m.Open(m.End()) {
		t.Fatal("预置迷宫的起点和终点必须是通路")
	}
}

func TestRandomDimensionsAndEndpoints(t *testing.T) {
	m, err := Random(12, 8, 0.3, 42)
	if err != nil {
		t.Fatalf("生成迷宫失败: %v", err)
	}
	if m.Rows() != 8 || m.Cols() != 12 {
		t.Fatalf("期望 12x8，实际 %dx%d", m.Cols(), m.Rows())
	}
	if !m.Open(m.Start()) {
		t.Fatal("随机迷宫起点 (0,0) 必须是通路")
	}
	if !m.Open(m.End()) {
		t.Fatal("随机迷宫终点必须是通路")
	}
}

func TestRandomRejectsInvalidSize(t *testing.T) {
	if _, err := Random(0, 5, 0.3, 1); err == nil {
		t.Fatal("宽度为 0 应返回错误")
	}
	if _, err := Random(5, 0, 0.3, 1); err == nil {
		t.Fatal("高度为 0 应返回错误")
	}
}

func TestRandomDensityZeroAllOpen(t *testing.T) {
	m, err := Random(6, 6, 0, 7)
	if err != nil {
		t.Fatalf("生成迷宫失败: %v", err)
	}
	for y, row := range m.Grid {
		for x, v := range row {
			if v != 0 {
				t.Fatalf("密度为 0 时 (%d,%d) 应全为通路", x, y)
			}
		}
	}
}

func TestRandomIsDeterministicBySeed(t *testing.T) {
	a, _ := Random(10, 10, 0.4, 99)
	b, _ := Random(10, 10, 0.4, 99)
	for y := range a.Grid {
		for x := range a.Grid[y] {
			if a.Grid[y][x] != b.Grid[y][x] {
				t.Fatalf("相同种子应生成相同迷宫，(%d,%d) 处不一致", x, y)
			}
		}
	}
}
