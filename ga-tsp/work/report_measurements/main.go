package main

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
	"simple_tuan/pkg/ga"
	"simple_tuan/pkg/tsp"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func main() {
	output := map[string]any{"date": time.Now().Format(time.RFC3339), "go": runtime.Version(), "map": campus.Default()}
	seeds := []int64{7, 17, 27, 37, 47, 57, 67, 77, 87, 97}
	inst := tsp.Att48()
	runs := []map[string]any{}
	for _, mode := range []string{"baseline", "mixed", "mixed_2opt"} {
		for _, seed := range seeds {
			p := ga.DefaultParams()
			p.Population = 100
			p.Generations = 200
			p.Seed = seed
			p.Initialization = "random"
			if mode != "baseline" {
				p.Initialization = "mixed"
			}
			p.LocalSearch = mode == "mixed_2opt"
			r, err := ga.Solve(inst, p)
			must(err)
			runs = append(runs, map[string]any{"mode": mode, "seed": seed, "distance": r.BestDistance, "ms": r.ElapsedMs, "convergedGen": r.ConvergedGen, "params": r.Params})
			if seed == 7 {
				output["trace_"+mode] = r
			}
		}
	}
	output["tspRuns"] = runs
	output["instance"] = inst
	sensitivity := []map[string]any{}
	for _, factor := range []string{"population", "mutation", "generations"} {
		values := map[string][]float64{"population": {30, 60, 100, 200}, "mutation": {0.01, 0.02, 0.05, 0.1}, "generations": {50, 100, 200, 400}}[factor]
		for _, value := range values {
			for _, seed := range seeds[:5] {
				p := ga.DefaultParams()
				p.Population = 100
				p.Generations = 200
				p.Seed = seed
				p.Initialization = "random"
				switch factor {
				case "population":
					p.Population = int(value)
				case "mutation":
					p.MutationRate = value
				case "generations":
					p.Generations = int(value)
				}
				r, err := ga.Solve(inst, p)
				must(err)
				sensitivity = append(sensitivity, map[string]any{"factor": factor, "value": value, "seed": seed, "distance": r.BestDistance, "ms": r.ElapsedMs})
			}
		}
	}
	output["sensitivity"] = sensitivity
	batch, err := logic.GenerateBatch(models.BatchRequest{Count: 30, Seed: 123, Scenario: "uniform"})
	must(err)
	req := models.PlanRequest{Batch: batch, Mode: "business", CapacityGrams: 6000, MaxMinutes: 45, SpeedKPH: 12, CostCentsPerKM: 100, CostCentsPerMinute: 30, Seed: 7, Population: 60, Generations: 100}
	comparison, err := logic.CompareCampus(context.Background(), models.CampusCompareRequest{Plan: req, Seeds: seeds[:5]})
	must(err)
	output["comparison"] = comparison
	for _, mode := range []string{"business", "knapsack", "route"} {
		req.Mode = mode
		r, err := logic.Plan(context.Background(), req)
		must(err)
		output["campus_"+mode] = r
	}
	small := req
	small.Mode = "business"
	small.Batch.Orders = small.Batch.Orders[:8]
	exact, err := logic.CompareCampus(context.Background(), models.CampusCompareRequest{Plan: small, Seeds: seeds[:5]})
	must(err)
	output["smallComparison"] = exact
	f, err := os.Create(os.Args[1])
	must(err)
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	must(enc.Encode(output))
}
