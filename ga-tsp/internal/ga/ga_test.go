package ga

import (
	"math"
	"math/rand"
	"slices"
	"testing"

	"simple_tuan/internal/tsp"
)

// checkPerm 校验 tour 是 0..n-1 的合法排列。
func checkPerm(t *testing.T, n int, tour []int, ctx string) {
	t.Helper()
	if len(tour) != n {
		t.Fatalf("%s：长度应为 %d，实际 %d", ctx, n, len(tour))
	}
	seen := make([]bool, n)
	for _, v := range tour {
		if v < 0 || v >= n || seen[v] {
			t.Fatalf("%s：非法排列 %v", ctx, tour)
		}
		seen[v] = true
	}
}

func randomParents(rng *rand.Rand) ([]int, []int) {
	p1 := rng.Perm(10)
	p2 := rng.Perm(10)
	return p1, p2
}

func TestOXCrossoverProducesValidPermutation(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 50; i++ {
		p1, p2 := randomParents(rng)
		child := OXCrossover(p1, p2, rng)
		checkPerm(t, 10, child, "OX 交叉")
	}
}

func TestPMXCrossoverProducesValidPermutation(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 50; i++ {
		p1, p2 := randomParents(rng)
		child := PMXCrossover(p1, p2, rng)
		checkPerm(t, 10, child, "PMX 交叉")
	}
}

func TestMutationsPreservePermutation(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	mutators := map[string]func([]int, *rand.Rand){
		"inversion": InversionMutate,
		"swap":      SwapMutate,
		"insert":    InsertMutate,
	}
	for name, fn := range mutators {
		for i := 0; i < 50; i++ {
			tour := rng.Perm(12)
			fn(tour, rng)
			checkPerm(t, 12, tour, name+" 变异")
		}
	}
}

func TestInversionMutateReversesSegment(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	tour := []int{0, 1, 2, 3, 4}
	// 固定区间 [1,3)：变异后应恰好逆转下标 1..2
	inversionMutateRange(tour, 1, 3)
	want := []int{0, 2, 1, 3, 4}
	if !slices.Equal(tour, want) {
		t.Fatalf("逆转区间 [1,3) 后应为 %v，实际 %v", want, tour)
	}
	_ = rng
}

func TestTournamentSelectPrefersBest(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	pop := [][]int{
		{0, 1, 2, 3},
		{3, 2, 1, 0},
		{1, 0, 3, 2},
	}
	// 距离：个体 1 最短（最优）
	dist := []float64{10, 2, 5}
	counts := make([]int, len(pop))
	for i := 0; i < 300; i++ {
		got := TournamentSelect(pop, dist, 3, rng)
		for j, ind := range pop {
			if slices.Equal(got, ind) {
				counts[j]++
			}
		}
	}
	// 最优个体被选中概率理论值 1-(2/3)^3≈0.70，应显著多于其他个体
	if counts[1] <= counts[0] || counts[1] <= counts[2] {
		t.Fatalf("锦标赛应偏好最优个体，选中次数 %v", counts)
	}
	if counts[1] < 150 {
		t.Fatalf("最优个体选中次数 %d 远低于理论期望 ~210", counts[1])
	}
}

func TestElitismNeverWorsensBest(t *testing.T) {
	inst, _ := tsp.Random(15, 7)
	p := DefaultParams()
	p.Seed = 11
	p.Population = 60
	p.Generations = 80
	res, err := Solve(inst, p)
	if err != nil {
		t.Fatalf("求解失败: %v", err)
	}
	for g := 1; g < len(res.Generations); g++ {
		if res.Generations[g].Best > res.Generations[g-1].Best+1e-9 {
			t.Fatalf("精英保留下最优解不应变差：第 %d 代 %v > 第 %d 代 %v",
				g, res.Generations[g].Best, g-1, res.Generations[g-1].Best)
		}
	}
}

func TestSolveImprovesOverInitialPopulation(t *testing.T) {
	inst, _ := tsp.Random(25, 5)
	p := DefaultParams()
	p.Seed = 9
	p.Population = 100
	p.Generations = 300
	res, err := Solve(inst, p)
	if err != nil {
		t.Fatalf("求解失败: %v", err)
	}
	if len(res.Generations) != p.Generations {
		t.Fatalf("应记录 %d 代快照，实际 %d", p.Generations, len(res.Generations))
	}
	if res.BestDistance >= res.Generations[0].Best {
		t.Fatalf("GA 应改进初始种群：最终 %v，初始 %v",
			res.BestDistance, res.Generations[0].Best)
	}
	// 最终回路必须合法，且长度与统计一致
	checkPerm(t, 25, res.BestTour, "最优回路")
	gotLen := inst.TourLength(res.BestTour)
	if math.Abs(gotLen-res.BestDistance) > 1e-6 {
		t.Fatalf("最优回路长度 %v 与统计值 %v 不一致", gotLen, res.BestDistance)
	}
	if res.ConvergedGen < 0 || res.ConvergedGen >= p.Generations {
		t.Fatalf("收敛代数非法: %d", res.ConvergedGen)
	}
}

func TestSolveDeterministicWithSameSeed(t *testing.T) {
	inst, _ := tsp.Random(20, 3)
	p := DefaultParams()
	p.Seed = 77
	p.Generations = 60
	r1, _ := Solve(inst, p)
	r2, _ := Solve(inst, p)
	if r1.BestDistance != r2.BestDistance {
		t.Fatalf("相同种子结果应一致：%v vs %v", r1.BestDistance, r2.BestDistance)
	}
	if !slices.Equal(r1.BestTour, r2.BestTour) {
		t.Fatal("相同种子最优回路应完全一致")
	}
}

func TestTwoOptImprovesOrKeepsLength(t *testing.T) {
	inst, _ := tsp.Random(18, 21)
	dm := inst.DistanceMatrix()
	rng := rand.New(rand.NewSource(6))
	for i := 0; i < 20; i++ {
		tour := rng.Perm(18)
		before := tourLength(dm, tour)
		TwoOptImprove(dm, tour)
		checkPerm(t, 18, tour, "2-opt 局部搜索")
		after := tourLength(dm, tour)
		if after > before+1e-9 {
			t.Fatalf("2-opt 不应使路径变长：%v -> %v", before, after)
		}
	}
}

func TestTwoOptUncrossesSquareTour(t *testing.T) {
	inst := &tsp.Instance{
		Name:     "square",
		EdgeType: "euclid",
		Cities: []tsp.City{
			{ID: 0, X: 0, Y: 0},
			{ID: 1, X: 0, Y: 1},
			{ID: 2, X: 1, Y: 1},
			{ID: 3, X: 1, Y: 0},
		},
	}
	dm := inst.DistanceMatrix()
	tour := []int{0, 2, 1, 3} // 交叉的劣解，长度 2√2+2≈4.83
	TwoOptImprove(dm, tour)
	if got := tourLength(dm, tour); math.Abs(got-4.0) > 1e-9 {
		t.Fatalf("2-opt 应消除交叉得到最优环游 4.0，实际 %v (%v)", got, tour)
	}
}

func TestLocalSearchReachesNearOptimalOnAtt48(t *testing.T) {
	p := DefaultParams()
	p.Seed = 2024
	p.Population = 60
	p.Generations = 120
	p.LocalSearch = true
	res, err := Solve(tsp.Att48(), p)
	if err != nil {
		t.Fatalf("求解失败: %v", err)
	}
	// 纯 GA（同规模）通常停在 13000 左右；GA+2-opt 应显著逼近 10628
	if res.BestDistance >= 11500 {
		t.Fatalf("GA+2-opt 应逼近 att48 最优 10628，实际 %v (Gap %.2f%%)",
			res.BestDistance, (res.BestDistance-10628)/10628*100)
	}
	if err := tsp.Att48().ValidateTour(res.BestTour); err != nil {
		t.Fatalf("最优回路非法: %v", err)
	}
}

func TestParamsNormalizeAndValidate(t *testing.T) {
	p := Params{} // 全零值
	if err := p.Normalize(); err != nil {
		t.Fatalf("零值参数应能填充默认值: %v", err)
	}
	if p.Population == 0 || p.Generations == 0 || p.CrossoverRate == 0 || p.MutationRate == 0 {
		t.Fatal("默认参数未填充完整")
	}

	bad := []Params{
		{Population: 1, Generations: 10},
		{Population: 2001, Generations: 10},
		{Population: 50, Generations: 10001},
		{Population: 50, Generations: 10, CrossoverRate: 1.5},
		{Population: 50, Generations: 10, MutationRate: -0.1},
		{Population: 50, Generations: 10, Elitism: 50},
		{Population: 50, Generations: 10, Selection: "unknown"},
		{Population: 50, Generations: 10, Crossover: "unknown"},
		{Population: 50, Generations: 10, Mutation: "unknown"},
	}
	for i, bp := range bad {
		if err := bp.Normalize(); err == nil {
			t.Fatalf("第 %d 组非法参数应报错: %+v", i, bp)
		}
	}
}
