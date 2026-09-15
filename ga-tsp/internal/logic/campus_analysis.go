package logic

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
)

func CompareCampus(ctx context.Context, req models.CampusCompareRequest) (models.CampusComparison, error) {
	result := models.CampusComparison{Groups: []models.CampusGroup{}}
	if err := ValidatePlan(&req.Plan); err != nil {
		return result, err
	}
	if len(req.Seeds) == 0 {
		req.Seeds = []int64{7, 17, 27, 37, 47}
	}
	if len(req.Seeds) > 5 {
		return result, fmt.Errorf("最多比较五个种子")
	}
	if req.Plan.Population*req.Plan.Generations*len(req.Seeds)*2 > 100000 {
		return result, fmt.Errorf("比较总预算超过 100000 次个体评估")
	}
	seen := map[int64]bool{}
	for _, seed := range req.Seeds {
		if seed <= 0 || seed > 9007199254740991 || seen[seed] {
			return result, fmt.Errorf("比较种子须为互异的正安全整数")
		}
		seen[seed] = true
	}
	result.Input = req
	for _, mode := range []string{"knapsack", "greedy", "business"} {
		name := map[string]string{"knapsack": "纯背包 GA + 路线优化", "greedy": "收入重量比贪心 + 路线优化", "business": "考虑整趟成本的业务 GA"}[mode]
		group := models.CampusGroup{Name: name, Runs: []models.CampusRun{}, BestNetCents: math.Inf(-1)}
		for _, seed := range req.Seeds {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			input := req.Plan
			input.Seed = seed
			var r models.PlanResult
			var err error
			if mode == "greedy" {
				r, err = greedyCampus(ctx, input)
			} else {
				input.Mode = mode
				r, err = Plan(ctx, input)
			}
			if err != nil {
				return result, err
			}
			group.Runs = append(group.Runs, models.CampusRun{
				Seed: seed, Metrics: r.Metrics, ElapsedMS: r.ElapsedMS, Curve: r.Curve, CurveUnit: r.CurveUnit,
			})
			group.MeanNetCents += r.Metrics.NetCents
			group.BestNetCents = math.Max(group.BestNetCents, r.Metrics.NetCents)
			if r.Metrics.Feasible {
				group.FeasibleRuns++
			}
		}
		group.MeanNetCents /= float64(len(group.Runs))
		for _, r := range group.Runs {
			group.StdDevNetCents += math.Pow(r.Metrics.NetCents-group.MeanNetCents, 2)
		}
		if len(group.Runs) > 1 {
			group.StdDevNetCents = math.Sqrt(group.StdDevNetCents / float64(len(group.Runs)-1))
		}
		result.Groups = append(result.Groups, group)
	}
	if len(req.Plan.Batch.Orders) <= 10 {
		exact, err := exactCampus(ctx, req.Plan)
		if err != nil {
			return result, err
		}
		result.ExactNetCents = &exact
	}
	return result, nil
}

func greedyCampus(ctx context.Context, req models.PlanRequest) (models.PlanResult, error) {
	start := time.Now()
	indices := make([]int, len(req.Batch.Orders))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool {
		a, b := req.Batch.Orders[indices[i]], req.Batch.Orders[indices[j]]
		return a.DeliveryFeeCents*b.WeightGrams > b.DeliveryFeeCents*a.WeightGrams
	})
	var mask uint64
	var weight int
	for _, i := range indices {
		o := req.Batch.Orders[i]
		if weight+o.WeightGrams <= req.CapacityGrams {
			mask |= 1 << i
			weight += o.WeightGrams
		}
	}
	p := planner{ctx: ctx, req: req, m: campus.Default(), cache: map[uint64]candidate{}}
	r, err := p.refine(p.evaluate(mask))
	return models.PlanResult{Metrics: r.metrics, Curve: []float64{}, CurveUnit: "分", ElapsedMS: float64(time.Since(start).Microseconds()) / 1000}, err
}

// exactCampus enumerates order sets and uses Held-Karp road tour costs, limited to ten orders.
func exactCampus(ctx context.Context, req models.PlanRequest) (float64, error) {
	if len(req.Batch.Orders) > 10 {
		return 0, fmt.Errorf("精确验证最多支持十单")
	}
	m := campus.Default()
	places := []int{}
	indices := map[int]int{}
	for _, o := range req.Batch.Orders {
		if _, ok := indices[o.DestinationID]; !ok && o.DestinationID != 0 {
			indices[o.DestinationID] = len(places)
			places = append(places, o.DestinationID)
		}
	}
	n := len(places)
	dp := make([][]float64, 1<<n)
	cost := make([]float64, 1<<n)
	for mask := range dp {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		dp[mask] = make([]float64, n)
		if mask > 0 {
			cost[mask] = math.Inf(1)
		}
		for j := range n {
			dp[mask][j] = math.Inf(1)
			if mask&(1<<j) == 0 {
				continue
			}
			previous := mask &^ (1 << j)
			if previous == 0 {
				dp[mask][j] = m.Distances[0][places[j]]
			}
			for k := range n {
				if previous&(1<<k) != 0 {
					dp[mask][j] = math.Min(dp[mask][j], dp[previous][k]+m.Distances[places[k]][places[j]])
				}
			}
			cost[mask] = math.Min(cost[mask], dp[mask][j]+m.Distances[places[j]][0])
		}
	}
	var best float64
	for mask := 0; mask < 1<<len(req.Batch.Orders); mask++ {
		var weight, income, seconds, nodes int
		for i, o := range req.Batch.Orders {
			if mask&(1<<i) != 0 {
				weight += o.WeightGrams
				income += o.DeliveryFeeCents
				seconds += o.ServiceSeconds
				if o.DestinationID != 0 {
					nodes |= 1 << indices[o.DestinationID]
				}
			}
		}
		minutes := cost[nodes]/(req.SpeedKPH*1000/60) + float64(seconds)/60
		if weight > req.CapacityGrams || minutes > req.MaxMinutes {
			continue
		}
		net := float64(income) - cost[nodes]/1000*req.CostCentsPerKM - minutes*req.CostCentsPerMinute
		best = math.Max(best, net)
	}
	return best, nil
}
