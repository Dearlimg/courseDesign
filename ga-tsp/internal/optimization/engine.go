// Package optimization 提供背包与 CEC 连续优化的遗传算法实验。
package optimization

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"time"
)

type Params struct {
	Population     int     `json:"population"`
	Generations    int     `json:"generations"`
	CrossoverRate  float64 `json:"crossoverRate"`
	MutationRate   float64 `json:"mutationRate"`
	Elitism        int     `json:"elitism"`
	Seed           int64   `json:"seed"`
	Selection      string  `json:"selection"`
	Crossover      string  `json:"crossover"`
	Mutation       string  `json:"mutation"`
	Initialization string  `json:"initialization"`
}

func (p Params) Validate(kind string) error {
	if p.Population < 4 || p.Population > 500 {
		return fmt.Errorf("种群规模须在 4～500 之间")
	}
	if p.Generations < 1 || p.Generations > 2000 {
		return fmt.Errorf("进化代数须在 1～2000 之间")
	}
	if p.Population*p.Generations > 500000 {
		return fmt.Errorf("种群规模乘以代数不能超过 500000")
	}
	for _, v := range []float64{p.CrossoverRate, p.MutationRate} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("概率必须为有限数")
		}
		if v < 0 || v > 1 {
			return fmt.Errorf("交叉和变异概率须在 0～1 之间")
		}
	}
	if p.Elitism < 0 || p.Elitism >= p.Population {
		return fmt.Errorf("精英数须小于种群规模且不为负数")
	}
	if !slices.Contains([]string{"tournament", "rank"}, p.Selection) {
		return fmt.Errorf("不支持的选择方法")
	}
	if !slices.Contains([]string{"random", "stratified"}, p.Initialization) {
		return fmt.Errorf("不支持的初始种群方法")
	}
	cross, mutation := []string{"onepoint", "uniform"}, []string{"bitflip", "swap"}
	if kind == "cec" {
		cross, mutation = []string{"arithmetic", "blend"}, []string{"gaussian", "reset"}
	}
	if !slices.Contains(cross, p.Crossover) || !slices.Contains(mutation, p.Mutation) {
		return fmt.Errorf("交叉或变异方法与当前问题不匹配")
	}
	return nil
}

type Generation struct {
	Gen       int         `json:"gen"`
	Best      float64     `json:"best"`
	Average   float64     `json:"average"`
	BestSoFar float64     `json:"bestSoFar"`
	Diversity float64     `json:"diversity"`
	Genes     []float64   `json:"genes"`
	Samples   [][]float64 `json:"samples"`
}

type Result struct {
	Generations  []Generation `json:"generations"`
	Best         float64      `json:"best"`
	Genes        []float64    `json:"genes"`
	Optimal      float64      `json:"optimal"`
	ConvergedGen int          `json:"convergedGen"`
	ElapsedMs    float64      `json:"elapsedMs"`
	Evaluations  int          `json:"evaluations"`
	Params       Params       `json:"params"`
}

type problem struct {
	kind     string
	size     int
	lower    float64
	upper    float64
	optimal  float64
	evaluate func([]float64) float64
	repair   func([]float64)
}

type engine struct {
	problem problem
	params  Params
	rng     *rand.Rand
}

func run(ctx context.Context, task problem, params Params) (Result, error) {
	if err := params.Validate(task.kind); err != nil {
		return Result{}, err
	}
	started := time.Now()
	e := engine{problem: task, params: params, rng: rand.New(rand.NewSource(params.Seed))}
	pop := e.initialize()
	result := Result{Generations: make([]Generation, 0, params.Generations), Genes: []float64{},
		Best: math.Inf(1), Optimal: task.optimal, Params: params}
	if task.kind == "knapsack" {
		result.Best = math.Inf(-1)
	}
	for gen := range params.Generations {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		scores := make([]float64, len(pop))
		order := make([]int, len(pop))
		seen := make(map[string]bool, len(pop))
		var sum float64
		for i, genes := range pop {
			scores[i] = task.evaluate(genes)
			sum += scores[i]
			order[i] = i
			seen[fmt.Sprint(genes)] = true
		}
		sort.SliceStable(order, func(i, j int) bool { return e.better(scores[order[i]], scores[order[j]]) })
		best := order[0]
		if e.better(scores[best], result.Best) {
			result.Best = scores[best]
			result.Genes = slices.Clone(pop[best])
			result.ConvergedGen = gen
		}
		samples := make([][]float64, 0, min(40, len(pop)))
		for i := 0; i < len(pop) && len(samples) < 40; i += max(1, len(pop)/40) {
			samples = append(samples, slices.Clone(pop[i][:min(2, task.size)]))
		}
		result.Generations = append(result.Generations, Generation{Gen: gen, Best: scores[best],
			Average: sum / float64(len(pop)), BestSoFar: result.Best, Diversity: float64(len(seen)) / float64(len(pop)),
			Genes: slices.Clone(pop[best]), Samples: samples})
		result.Evaluations += len(pop)
		if gen+1 < params.Generations {
			pop = e.evolve(pop, order)
		}
	}
	result.ElapsedMs = float64(time.Since(started).Microseconds()) / 1000
	return result, nil
}

func (e engine) better(a, b float64) bool {
	if e.problem.kind == "knapsack" {
		return a > b
	}
	return a < b
}

func (e engine) initialize() [][]float64 {
	p := e.params
	pop := make([][]float64, p.Population)
	for i := range pop {
		pop[i] = make([]float64, e.problem.size)
	}
	for j := range e.problem.size {
		perm := e.rng.Perm(p.Population)
		for i := range pop {
			u := e.rng.Float64()
			if p.Initialization == "stratified" {
				u = (float64(perm[i]) + u) / float64(p.Population)
			}
			pop[i][j] = e.problem.lower + u*(e.problem.upper-e.problem.lower)
			if e.problem.kind == "knapsack" {
				pop[i][j] = 0
				if u > 0.5 {
					pop[i][j] = 1
				}
			}
		}
	}
	for _, genes := range pop {
		e.problem.repair(genes)
	}
	return pop
}

func (e engine) parent(order []int) int {
	n := len(order)
	if e.params.Selection == "rank" {
		pick := e.rng.Intn(n * (n + 1) / 2)
		for rank, id := range order {
			pick -= n - rank
			if pick < 0 {
				return id
			}
		}
	}
	rank := e.rng.Intn(n)
	for range 2 {
		rank = min(rank, e.rng.Intn(n))
	}
	return order[rank]
}

func (e engine) evolve(pop [][]float64, order []int) [][]float64 {
	next := make([][]float64, 0, len(pop))
	for _, id := range order[:e.params.Elitism] {
		next = append(next, slices.Clone(pop[id]))
	}
	for len(next) < len(pop) {
		a, b := pop[e.parent(order)], pop[e.parent(order)]
		child := slices.Clone(a)
		if e.rng.Float64() < e.params.CrossoverRate {
			e.cross(child, b)
		}
		e.mutate(child)
		e.problem.repair(child)
		next = append(next, child)
	}
	return next
}

func (e engine) cross(child, other []float64) {
	cut := 1 + e.rng.Intn(len(child)-1)
	for i, a := range child {
		b := other[i]
		switch e.params.Crossover {
		case "onepoint":
			if i >= cut {
				child[i] = b
			}
		case "uniform":
			if e.rng.Intn(2) == 0 {
				child[i] = b
			}
		case "arithmetic":
			alpha := e.rng.Float64()
			child[i] = alpha*a + (1-alpha)*b
		case "blend":
			lo, hi := min(a, b), max(a, b)
			child[i] = lo + (e.rng.Float64()*1.5-0.25)*(hi-lo)
		}
	}
}

func (e engine) mutate(child []float64) {
	span := e.problem.upper - e.problem.lower
	for i := range child {
		if e.rng.Float64() >= e.params.MutationRate {
			continue
		}
		switch e.params.Mutation {
		case "bitflip":
			child[i] = 1 - child[i]
		case "swap":
			j := e.rng.Intn(len(child))
			child[i], child[j] = child[j], child[i]
		case "gaussian":
			child[i] += e.rng.NormFloat64() * span * 0.04
		case "reset":
			child[i] = e.problem.lower + e.rng.Float64()*span
		}
	}
}
