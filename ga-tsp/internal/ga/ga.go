// Package ga 实现求解 TSP 的遗传算法引擎：
// 排列编码 + 精英保留 + 可切换的选择/交叉/变异算子，逐代记录进化快照。
package ga

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"time"

	"simple_tuan/internal/tsp"
)

// Params 是 GA 求解参数。
type Params struct {
	crossSet       bool
	mutationSet    bool
	elitismSet     bool
	seedSet        bool
	Encoding       string  `json:"encoding"`       // permutation / random-key
	Initialization string  `json:"initialization"` // random / mixed
	Population     int     `json:"population"`     // 种群规模
	Generations    int     `json:"generations"`    // 进化代数上限
	CrossoverRate  float64 `json:"crossoverRate"`  // 交叉概率
	MutationRate   float64 `json:"mutationRate"`   // 变异概率
	Elitism        int     `json:"elitism"`        // 精英保留个数
	Selection      string  `json:"selection"`      // tournament / roulette
	Crossover      string  `json:"crossover"`      // ox / pmx
	Mutation       string  `json:"mutation"`       // inversion / swap / insert
	LocalSearch    bool    `json:"localSearch"`    // 每代对最优个体做 2-opt 局部搜索（模因算法）
	Seed           int64   `json:"seed"`           // 随机种子（0 表示按时间随机）
}

// DefaultParams 返回推荐的默认参数（种子除外）。
func DefaultParams() Params {
	return Params{
		crossSet: true, mutationSet: true, elitismSet: true,
		Population:    200,
		Generations:   500,
		CrossoverRate: 0.9,
		MutationRate:  0.02,
		Elitism:       2,
		Selection:     "tournament",
		Crossover:     "ox",
		Mutation:      "inversion",
	}
}

// Normalize 填充零值字段为默认值并校验取值范围。
func (p *Params) Normalize() error {
	d := DefaultParams()
	if p.Encoding == "" {
		p.Encoding = "permutation"
	}
	if p.Initialization == "" {
		p.Initialization = "random"
	}
	if !slices.Contains([]string{"permutation", "random-key"}, p.Encoding) {
		return fmt.Errorf("不支持的编码方法")
	}
	if !slices.Contains([]string{"random", "mixed"}, p.Initialization) {
		return fmt.Errorf("不支持的初始化方法")
	}
	if p.Encoding == "random-key" {
		if p.Crossover == "" {
			p.Crossover = "uniform"
		}
		if p.Mutation == "" {
			p.Mutation = "reset"
		}
	}
	if p.Population == 0 {
		p.Population = d.Population
	}
	if p.Generations == 0 {
		p.Generations = d.Generations
	}
	if p.CrossoverRate == 0 && !p.crossSet {
		p.CrossoverRate = d.CrossoverRate
	}
	if p.MutationRate == 0 && !p.mutationSet {
		p.MutationRate = d.MutationRate
	}
	if p.Elitism == 0 && !p.elitismSet {
		p.Elitism = d.Elitism
	}
	if p.Selection == "" {
		p.Selection = d.Selection
	}
	if p.Crossover == "" {
		p.Crossover = d.Crossover
	}
	if p.Mutation == "" {
		p.Mutation = d.Mutation
	}
	if p.Seed == 0 && !p.seedSet {
		p.Seed = time.Now().UnixNano()
	}

	switch {
	case p.Population < 2 || p.Population > 2000:
		return fmt.Errorf("种群规模须在 [2,2000]，实际 %d", p.Population)
	case p.Generations < 1 || p.Generations > 10000:
		return fmt.Errorf("进化代数须在 [1,10000]，实际 %d", p.Generations)
	case p.CrossoverRate < 0 || p.CrossoverRate > 1:
		return fmt.Errorf("交叉概率须在 [0,1]，实际 %v", p.CrossoverRate)
	case p.MutationRate < 0 || p.MutationRate > 1:
		return fmt.Errorf("变异概率须在 [0,1]，实际 %v", p.MutationRate)
	case p.Elitism < 0 || p.Elitism >= p.Population:
		return fmt.Errorf("精英数须在 [0,种群规模)，实际 %d", p.Elitism)
	}
	if !slices.Contains([]string{"tournament", "roulette"}, p.Selection) {
		return fmt.Errorf("未知选择算子 %q，支持 tournament / roulette", p.Selection)
	}
	if p.Encoding == "random-key" {
		if p.Crossover != "uniform" || p.Mutation != "reset" {
			return fmt.Errorf("随机键编码须使用均匀交叉与随机重置变异")
		}
		return nil
	}
	if !slices.Contains([]string{"ox", "pmx"}, p.Crossover) {
		return fmt.Errorf("未知交叉算子 %q，支持 ox / pmx", p.Crossover)
	}
	if !slices.Contains([]string{"inversion", "swap", "insert"}, p.Mutation) {
		return fmt.Errorf("未知变异算子 %q，支持 inversion / swap / insert", p.Mutation)
	}
	return nil
}

// Generation 是一代进化的快照。
type Generation struct {
	Gen         int     `json:"gen"`
	Best        float64 `json:"best"`        // 本代最短环游距离
	Avg         float64 `json:"avg"`         // 本代平均距离
	Worst       float64 `json:"worst"`       // 本代最长距离
	Diversity   float64 `json:"diversity"`   // 种群多样性（0~1，独特个体占比）
	BestTour    []int   `json:"bestTour"`    // 本代最优环游
	SampleTours [][]int `json:"sampleTours"` // 种群抽样（最优+若干个体），用于可视化种群分布
}

// Result 是一次完整进化的结果。
type Result struct {
	Generations  []Generation `json:"generations"`
	BestTour     []int        `json:"bestTour"`
	BestDistance float64      `json:"bestDistance"`
	ConvergedGen int          `json:"convergedGen"` // 最优解最后一次被改进的代数
	ElapsedMs    float64      `json:"elapsedMs"`
	Params       Params       `json:"params"`
}

// Solve 运行遗传算法并返回逐代快照与最终结果。
func Solve(inst *tsp.Instance, p Params) (Result, error) {
	return SolveContext(context.Background(), inst, p)
}

// SolveContext permits cancellation between generations.
func SolveContext(ctx context.Context, inst *tsp.Instance, p Params) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if err := p.Normalize(); err != nil {
		return Result{}, err
	}
	if inst.Size() < 4 {
		return Result{}, fmt.Errorf("城市数至少为 4，实际 %d", inst.Size())
	}
	if p.Encoding == "random-key" {
		return solveKeys(ctx, inst, p)
	}
	start := time.Now()
	rng := rand.New(rand.NewSource(p.Seed))
	n := inst.Size()
	dm := inst.DistanceMatrix()

	// 初始种群：随机排列
	pop := make([][]int, p.Population)
	for i := range pop {
		pop[i] = rng.Perm(n)
		if p.Initialization == "mixed" && i < p.Population/2 {
			pop[i] = nearestTour(dm, rng.Intn(n))
		}
	}

	result := Result{
		Generations:  make([]Generation, 0, p.Generations),
		BestDistance: math.Inf(1),
		Params:       p,
	}

	for gen := 0; gen < p.Generations; gen++ {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		dist := make([]float64, p.Population)
		sum := 0.0
		for i, tour := range pop {
			dist[i] = tourLength(dm, tour)
			sum += dist[i]
		}
		bestIdx := argMin(dist)

		// 模因算法：每代对本代最优个体做 2-opt 局部搜索（拉马克式改进，
		// 改进结果直接写回种群，随精英保留传递给下一代）
		if p.LocalSearch && TwoOptImprove(dm, pop[bestIdx]) {
			newLen := tourLength(dm, pop[bestIdx])
			sum += newLen - dist[bestIdx]
			dist[bestIdx] = newLen
		}

		// 记录本代快照与历史最优
		if dist[bestIdx] < result.BestDistance {
			result.BestDistance = dist[bestIdx]
			result.BestTour = slices.Clone(pop[bestIdx])
			result.ConvergedGen = gen
		}
		// 种群抽样：最优个体 + 均匀采样 3 个（不消耗随机数流，保持进化确定性）
		samples := make([][]int, 0, 4)
		samples = append(samples, slices.Clone(pop[bestIdx]))
		step := p.Population / 4
		if step < 1 {
			step = 1
		}
		for k := step; k < p.Population && len(samples) < 4; k += step {
			samples = append(samples, slices.Clone(pop[k]))
		}
		result.Generations = append(result.Generations, Generation{
			Gen:         gen,
			Best:        dist[bestIdx],
			Avg:         sum / float64(p.Population),
			Worst:       slices.Max(dist),
			Diversity:   uniqueRatio(pop),
			BestTour:    slices.Clone(pop[bestIdx]),
			SampleTours: samples,
		})

		if gen == p.Generations-1 {
			break // 最后一代不再繁殖
		}
		pop = evolve(pop, dist, &p, rng)
	}
	result.ElapsedMs = float64(time.Since(start).Microseconds()) / 1000.0
	return result, nil
}

// evolve 产生下一代种群：精英保留 + 选择/交叉/变异。
func evolve(pop [][]int, dist []float64, p *Params, rng *rand.Rand) [][]int {
	order := make([]int, len(pop))
	for i := range order {
		order[i] = i
	}
	// 按距离升序排列个体下标，前 Elitism 个为精英
	sort.Slice(order, func(a, b int) bool { return dist[order[a]] < dist[order[b]] })

	next := make([][]int, 0, len(pop))
	for i := 0; i < p.Elitism; i++ {
		next = append(next, slices.Clone(pop[order[i]]))
	}

	selectParent := func() []int {
		if p.Selection == "roulette" {
			return RouletteSelect(pop, dist, rng)
		}
		return TournamentSelect(pop, dist, 3, rng)
	}
	crossover := OXCrossover
	if p.Crossover == "pmx" {
		crossover = PMXCrossover
	}
	var mutate func([]int, *rand.Rand)
	switch p.Mutation {
	case "swap":
		mutate = SwapMutate
	case "insert":
		mutate = InsertMutate
	default:
		mutate = InversionMutate
	}

	for len(next) < len(pop) {
		child := slices.Clone(selectParent())
		if rng.Float64() < p.CrossoverRate {
			child = crossover(selectParent(), selectParent(), rng)
		}
		if rng.Float64() < p.MutationRate {
			mutate(child, rng)
		}
		next = append(next, child)
	}
	return next
}

// ---------- 选择算子 ----------

// TournamentSelect 锦标赛选择：随机取 k 个个体，返回其中最优者。
func TournamentSelect(pop [][]int, dist []float64, k int, rng *rand.Rand) []int {
	best := rng.Intn(len(pop))
	for i := 1; i < k; i++ {
		cand := rng.Intn(len(pop))
		if dist[cand] < dist[best] {
			best = cand
		}
	}
	return pop[best]
}

// RouletteSelect 轮盘赌选择：适应度取 1/距离，距离越短被选中概率越大。
func RouletteSelect(pop [][]int, dist []float64, rng *rand.Rand) []int {
	fitness := make([]float64, len(pop))
	total := 0.0
	for i, d := range dist {
		fitness[i] = 1.0 / d
		total += fitness[i]
	}
	pick := rng.Float64() * total
	for i, f := range fitness {
		pick -= f
		if pick <= 0 {
			return pop[i]
		}
	}
	return pop[len(pop)-1]
}

// ---------- 交叉算子 ----------

// OXCrossover 顺序交叉：保留 p1 中 [i,j) 片段，其余城市按 p2 顺序填充。
func OXCrossover(p1, p2 []int, rng *rand.Rand) []int {
	n := len(p1)
	i, j := twoCutPoints(n, rng)

	child := make([]int, n)
	for k := range child {
		child[k] = -1
	}
	inSegment := make([]bool, n)
	for k := i; k < j; k++ {
		child[k] = p1[k]
		inSegment[p1[k]] = true
	}
	// 从 p2 的 j 之后起，按 p2 环形顺序填入未出现城市
	pos := j
	for k := 0; k < n; k++ {
		c := p2[(j+k)%n]
		if inSegment[c] {
			continue
		}
		child[pos] = c
		pos = (pos + 1) % n
	}
	return child
}

// PMXCrossover 部分匹配交叉：交换片段并用位置映射修复冲突。
func PMXCrossover(p1, p2 []int, rng *rand.Rand) []int {
	n := len(p1)
	i, j := twoCutPoints(n, rng)

	child := slices.Clone(p2)
	// 段内取自 p1
	for k := i; k < j; k++ {
		child[k] = p1[k]
	}
	// 修复段外冲突：沿 p1↔p2 的段内映射链替换
	for k := 0; k < n; k++ {
		if k >= i && k < j {
			continue
		}
		for slices.Contains(child[i:j], child[k]) {
			// 找到冲突城市在 p1 段中的位置，用 p2 对应位置的值替换
			idx := slices.Index(p1[i:j], child[k]) + i
			child[k] = p2[idx]
		}
	}
	return child
}

// twoCutPoints 返回两个随机切割点 0 <= i < j <= n。
func twoCutPoints(n int, rng *rand.Rand) (int, int) {
	i := rng.Intn(n)
	j := rng.Intn(n)
	if i == j {
		j = (j + 1) % n
	}
	if i > j {
		i, j = j, i
	}
	return i, j
}

// ---------- 变异算子（原地修改） ----------

// InversionMutate 逆转变异：随机选一段子路径并反转（对 TSP 收敛效果好）。
func InversionMutate(tour []int, rng *rand.Rand) {
	n := len(tour)
	i, j := twoCutPoints(n, rng)
	inversionMutateRange(tour, i, j)
}

// inversionMutateRange 反转 tour 的 [i,j) 区间。
func inversionMutateRange(tour []int, i, j int) {
	for a, b := i, j-1; a < b; a, b = a+1, b-1 {
		tour[a], tour[b] = tour[b], tour[a]
	}
}

// SwapMutate 交换变异：随机交换两座城市的位置。
func SwapMutate(tour []int, rng *rand.Rand) {
	i, j := twoCutPoints(len(tour), rng)
	tour[i], tour[j] = tour[j], tour[i]
}

// InsertMutate 插入变异：随机取一座城市插入到另一位置。
func InsertMutate(tour []int, rng *rand.Rand) {
	n := len(tour)
	i, j := twoCutPoints(n, rng)
	city := tour[i]
	// 删除 i 后插入 j（j 的语义在删除后仍有效：j > i）
	copy(tour[i:j], tour[i+1:j+1])
	tour[j] = city
}

// ---------- 辅助函数 ----------

// TwoOptImprove 对环游执行 2-opt 局部搜索：反复寻找可消除交叉的边交换，
// 直到无改进为止。成功改进返回 true（tor 原地修改）。
func TwoOptImprove(dm [][]float64, tour []int) bool {
	n := len(tour)
	improved := false
	for {
		gain := false
		for i := 0; i < n-1; i++ {
			for j := i + 2; j < n; j++ {
				if i == 0 && j == n-1 {
					continue // 整环反转无意义
				}
				a, b := tour[i], tour[i+1]
				c, d := tour[j], tour[(j+1)%n]
				delta := dm[a][c] + dm[b][d] - dm[a][b] - dm[c][d]
				if delta < -1e-9 {
					inversionMutateRange(tour, i+1, j+1)
					gain = true
					improved = true
				}
			}
		}
		if !gain {
			return improved
		}
	}
}

// tourLength 用预计算距离矩阵求环游长度（闭合）。
func tourLength(dm [][]float64, tour []int) float64 {
	total := 0.0
	for i := 0; i < len(tour); i++ {
		total += dm[tour[i]][tour[(i+1)%len(tour)]]
	}
	return total
}

// argMin 返回最小值的下标。
func argMin(xs []float64) int {
	best := 0
	for i, x := range xs {
		if x < xs[best] {
			best = i
		}
	}
	return best
}

// uniqueRatio 计算种群多样性：旋转归一化后独特个体的占比（1=完全多样，0=全部相同）。
func uniqueRatio(pop [][]int) float64 {
	seen := make(map[string]struct{}, len(pop))
	for _, tour := range pop {
		seen[normalizeTour(tour)] = struct{}{}
	}
	return float64(len(seen)) / float64(len(pop))
}

// normalizeTour 把环游旋转到以城市 0 开头，作为等价类的代表。
func normalizeTour(tour []int) string {
	start := slices.Index(tour, 0)
	n := len(tour)
	buf := make([]byte, 0, n*4)
	for k := 0; k < n; k++ {
		buf = append(buf, byte(tour[(start+k)%n]))
		buf = append(buf, ',')
	}
	return string(buf)
}
