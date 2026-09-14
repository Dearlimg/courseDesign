package tsp

import (
	"math"
	"testing"
)

// 正方形四城：环游 0→1→2→3→0 周长恰为 4.0
func squareInstance() *Instance {
	return &Instance{
		Name:     "square",
		EdgeType: "euclid",
		Cities: []City{
			{ID: 0, X: 0, Y: 0},
			{ID: 1, X: 0, Y: 1},
			{ID: 2, X: 1, Y: 1},
			{ID: 3, X: 1, Y: 0},
		},
	}
}

func TestTourLengthOnSquare(t *testing.T) {
	inst := squareInstance()
	got := inst.TourLength([]int{0, 1, 2, 3})
	if math.Abs(got-4.0) > 1e-9 {
		t.Fatalf("正方形环游长度应为 4.0，实际 %v", got)
	}
}

func TestAtt48HasOptimalTourLength(t *testing.T) {
	inst := Att48()
	if inst.Name != "att48" {
		t.Fatalf("实例名应为 att48，实际 %q", inst.Name)
	}
	if len(inst.Cities) != 48 {
		t.Fatalf("att48 应有 48 座城市，实际 %d", len(inst.Cities))
	}
	if inst.EdgeType != "att" {
		t.Fatalf("att48 应使用 ATT 伪欧氏距离，实际 %q", inst.EdgeType)
	}
	// 指导书附件 1 给出的最优回路（0 基），已知最优长度 10628
	optimalTour := []int{
		0, 7, 37, 30, 43, 17, 6, 27, 5, 36, 18, 26, 16, 42, 29, 35,
		45, 32, 19, 46, 20, 31, 38, 47, 4, 41, 23, 9, 44, 34, 3, 25,
		1, 28, 33, 40, 15, 21, 2, 22, 13, 24, 12, 10, 11, 14, 39, 8,
	}
	got := inst.TourLength(optimalTour)
	if got != 10628 {
		t.Fatalf("att48 最优回路长度应为 10628，实际 %v（坐标或距离函数有误）", got)
	}
}

func TestAttDistanceRoundsPseudoEuclidean(t *testing.T) {
	// 城市间 dx=531, dy=-185：r=sqrt((531²+185²)/10)≈177.81，四舍五入为 178
	a, b := City{ID: 0, X: 6734, Y: 1453}, City{ID: 1, X: 7265, Y: 1268}
	if got := attDistance(a, b); got != 178 {
		t.Fatalf("ATT 距离应为 178，实际 %v", got)
	}
}

func TestValidateTour(t *testing.T) {
	inst := squareInstance()
	if err := inst.ValidateTour([]int{0, 3, 2, 1}); err != nil {
		t.Fatalf("合法环游不应报错: %v", err)
	}
	if err := inst.ValidateTour([]int{0, 1, 1, 2}); err == nil {
		t.Fatal("含重复城市的环游应报错")
	}
	if err := inst.ValidateTour([]int{0, 1, 2}); err == nil {
		t.Fatal("缺城市的环游应报错")
	}
	if err := inst.ValidateTour(nil); err == nil {
		t.Fatal("空环游应报错")
	}
}

func TestRandomInstance(t *testing.T) {
	inst, err := Random(20, 42)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if len(inst.Cities) != 20 {
		t.Fatalf("应有 20 城，实际 %d", len(inst.Cities))
	}
	for i, c := range inst.Cities {
		if c.ID != i {
			t.Fatalf("城市编号应从 0 递增，第 %d 个为 %d", i, c.ID)
		}
		if c.X < 0 || c.X > 100 || c.Y < 0 || c.Y > 100 {
			t.Fatalf("城市 %d 坐标超出 [0,100]: %v", i, c)
		}
	}
	again, _ := Random(20, 42)
	for i := range inst.Cities {
		if inst.Cities[i] != again.Cities[i] {
			t.Fatalf("相同种子应生成相同实例，城市 %d 不一致", i)
		}
	}
}

func TestRandomInstanceRejectsBadSize(t *testing.T) {
	if _, err := Random(3, 1); err == nil {
		t.Fatal("少于 4 城应报错")
	}
	if _, err := Random(501, 1); err == nil {
		t.Fatal("超过 500 城应报错")
	}
}
