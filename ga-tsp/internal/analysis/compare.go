// Package analysis runs reproducible, bounded optimization comparisons.
package analysis

import (
	"context"
	"fmt"
	"math"

	"gatsp/internal/dispatch"
	"gatsp/internal/ga"
	"gatsp/internal/optimization"
)

type Group struct {
	Name     string               `json:"name"`
	TSP      *ga.Params           `json:"tsp,omitempty"`
	Knapsack *optimization.Params `json:"knapsack,omitempty"`
}
type Request struct {
	Problem   string                     `json:"problem"`
	Route     *dispatch.RouteRequest     `json:"route,omitempty"`
	Selection *dispatch.SelectionRequest `json:"selection,omitempty"`
	Groups    []Group                    `json:"groups"`
	Seeds     []int64                    `json:"seeds"`
}
type Run struct {
	Seed      int64                `json:"seed"`
	Value     float64              `json:"value"`
	ElapsedMs float64              `json:"elapsedMs"`
	Curve     []float64            `json:"curve"`
	TSP       *ga.Params           `json:"tsp,omitempty"`
	Knapsack  *optimization.Params `json:"knapsack,omitempty"`
}
type Summary struct {
	Name          string    `json:"name"`
	Best          float64   `json:"best"`
	Mean          float64   `json:"mean"`
	StdDev        float64   `json:"stdDev"`
	MeanElapsedMs float64   `json:"meanElapsedMs"`
	MeanCurve     []float64 `json:"meanCurve"`
	Runs          []Run     `json:"runs"`
}
type Result struct {
	Input  Request   `json:"input"`
	Unit   string    `json:"unit"`
	Groups []Summary `json:"groups"`
}

func validate(req *Request) error {
	if req.Problem != "tsp" && req.Problem != "knapsack" {
		return fmt.Errorf("问题须为 tsp 或 knapsack")
	}
	if len(req.Groups) < 1 || len(req.Groups) > 8 {
		return fmt.Errorf("配置组数须为 1～8")
	}
	if len(req.Seeds) == 0 {
		req.Seeds = []int64{7, 17, 27, 37, 47}
	}
	if len(req.Seeds) > 10 {
		return fmt.Errorf("每组最多 10 个随机种子")
	}
	seen := make(map[int64]bool)
	for _, s := range req.Seeds {
		if s <= 0 || s > 9007199254740991 || seen[s] {
			return fmt.Errorf("随机种子须为不重复的正安全整数")
		}
		seen[s] = true
	}
	size := 0
	if req.Problem == "tsp" {
		if req.Route == nil {
			return fmt.Errorf("缺少配送点输入")
		}
		inst, _, err := dispatch.RouteInstance(*req.Route)
		if err != nil {
			return err
		}
		if inst.Size() < 4 {
			return fmt.Errorf("TSP 实验至少需要 3 个不同送达点（取餐点之外）")
		}
		size = inst.Size()
	} else {
		if req.Selection == nil {
			return fmt.Errorf("缺少订单输入")
		}
		if err := dispatch.ValidateOrders(req.Selection.Orders); err != nil {
			return err
		}
		if req.Selection.Capacity < 1 || req.Selection.Capacity > 10000 {
			return fmt.Errorf("餐箱容量须为 1～10000")
		}
		size = len(req.Selection.Orders)
		if size < 2 {
			return fmt.Errorf("背包实验至少需要 2 笔订单")
		}
	}
	var evaluations, work int64
	names := make(map[string]bool)
	for i, g := range req.Groups {
		if g.Name == "" || len([]rune(g.Name)) > 80 || names[g.Name] {
			return fmt.Errorf("配置组名称为空、重复或过长")
		}
		names[g.Name] = true
		var count int
		if req.Problem == "tsp" {
			if g.TSP == nil || g.Knapsack != nil {
				return fmt.Errorf("TSP 配置组须提供 tsp 参数")
			}
			p := *g.TSP
			if err := p.Normalize(); err != nil {
				return err
			}
			if p.LocalSearch {
				return fmt.Errorf("重复实验关闭 2-opt，以控制计算量及比较条件；原始实验页可单独研究 2-opt")
			}
			req.Groups[i].TSP = &p
			count = p.Population * p.Generations
		} else {
			if g.Knapsack == nil || g.TSP != nil {
				return fmt.Errorf("背包配置组须提供 knapsack 参数")
			}
			if err := g.Knapsack.Validate("knapsack"); err != nil {
				return err
			}
			count = g.Knapsack.Population * g.Knapsack.Generations
		}
		n := int64(count) * int64(len(req.Seeds))
		evaluations += n
		work += n * int64(size)
	}
	if evaluations > 1500000 || work > 30000000 {
		return fmt.Errorf("实验超过总预算（150 万次评估或 3000 万基因评估），请减少种群、代数或重复次数")
	}
	return nil
}

func Compare(ctx context.Context, req Request) (Result, error) {
	result := Result{Groups: []Summary{}}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := validate(&req); err != nil {
		return result, err
	}
	result.Input = req
	result.Unit = "distance"
	if req.Problem == "knapsack" {
		result.Unit = "cents"
	}
	for _, g := range req.Groups {
		runs := []Run{}
		for _, seed := range req.Seeds {
			if err := ctx.Err(); err != nil {
				return Result{}, err
			}
			run := Run{Seed: seed, Curve: []float64{}}
			if req.Problem == "tsp" {
				inst, _, err := dispatch.RouteInstance(*req.Route)
				if err != nil {
					return Result{}, err
				}
				p := *g.TSP
				p.Seed = seed
				r, err := ga.SolveContext(ctx, inst, p)
				if err != nil {
					return Result{}, err
				}
				run.Value = r.BestDistance
				run.ElapsedMs = r.ElapsedMs
				run.TSP = &r.Params
				best := math.Inf(1)
				for _, gen := range r.Generations {
					best = min(best, gen.Best)
					run.Curve = append(run.Curve, best)
				}
			} else {
				input := *req.Selection
				p := *g.Knapsack
				p.Seed = seed
				input.Params = &p
				r, err := dispatch.Select(ctx, input)
				if err != nil {
					return Result{}, err
				}
				run.Value = float64(r.Income)
				run.ElapsedMs = r.Evolution.ElapsedMs
				run.Knapsack = &r.Evolution.Params
				for _, gen := range r.Evolution.Generations {
					run.Curve = append(run.Curve, gen.BestSoFar)
				}
				if len(run.Curve) == 0 {
					run.Curve = []float64{run.Value}
				}
			}
			runs = append(runs, run)
		}
		result.Groups = append(result.Groups, summarize(g.Name, runs, req.Problem == "knapsack"))
	}
	return result, nil
}
func summarize(name string, runs []Run, maximize bool) Summary {
	s := Summary{Name: name, Runs: runs, MeanCurve: []float64{}, Best: runs[0].Value}
	for _, r := range runs {
		s.Mean += r.Value
		s.MeanElapsedMs += r.ElapsedMs
		if maximize {
			s.Best = max(s.Best, r.Value)
		} else {
			s.Best = min(s.Best, r.Value)
		}
	}
	n := float64(len(runs))
	s.Mean /= n
	s.MeanElapsedMs /= n
	for _, r := range runs {
		s.StdDev += (r.Value - s.Mean) * (r.Value - s.Mean)
	}
	if len(runs) > 1 {
		s.StdDev = math.Sqrt(s.StdDev / (n - 1))
	}
	for i := range runs[0].Curve {
		var value float64
		for _, r := range runs {
			value += r.Curve[i]
		}
		s.MeanCurve = append(s.MeanCurve, value/n)
	}
	return s
}
