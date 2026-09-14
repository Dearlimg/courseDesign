package analysis

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"gatsp/internal/dispatch"
	"gatsp/internal/ga"
)

func TestSummary(t *testing.T) {
	s := summarize("x", []Run{{Value: 1, Curve: []float64{3, 1}}, {Value: 3, Curve: []float64{5, 3}}}, false)
	if s.Mean != 2 || math.Abs(s.StdDev-math.Sqrt(2)) > 1e-12 || s.Best != 1 || s.MeanCurve[0] != 4 {
		t.Fatal(s)
	}
}
func TestCompareReproducible(t *testing.T) {
	var p ga.Params
	if err := json.Unmarshal([]byte(`{"population":10,"generations":5,"mutationRate":0,"crossoverRate":0,"elitism":0}`), &p); err != nil {
		t.Fatal(err)
	}
	req := Request{
		Problem: "tsp", Seeds: []int64{7, 17},
		Route:  &dispatch.RouteRequest{Depot: dispatch.Point{Name: "站"}, Points: []dispatch.Point{{Name: "A", X: 3}, {Name: "B", Y: 4}, {Name: "C", X: 5, Y: 6}}},
		Groups: []Group{{Name: "zero", TSP: &p}},
	}
	result, err := Compare(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range result.Groups[0].Runs {
		if run.TSP.MutationRate != 0 || run.TSP.CrossoverRate != 0 || run.TSP.Elitism != 0 {
			t.Fatal("zero parameter replaced")
		}
		inst, _, _ := dispatch.RouteInstance(*req.Route)
		direct, err := ga.Solve(inst, *run.TSP)
		if err != nil || direct.BestDistance != run.Value {
			t.Fatal("run mismatch", err)
		}
	}
	req.Seeds = []int64{7, 7}
	if _, err := Compare(context.Background(), req); err == nil {
		t.Fatal("duplicate seeds accepted")
	}
}
func TestCompareBudgetAndCancel(t *testing.T) {
	p := dispatch.DefaultSelectionParams()
	p.Population = 500
	p.Generations = 1000
	req := Request{Problem: "knapsack", Selection: &dispatch.SelectionRequest{Capacity: 2, Orders: []dispatch.Order{{ID: "A", Name: "A", Load: 1, Income: 10}, {ID: "B", Name: "B", Load: 2, Income: 20}}}, Groups: []Group{{Name: "test", Knapsack: &p}}}
	if _, err := Compare(context.Background(), req); err == nil {
		t.Fatal("budget accepted")
	}
	p.Population = 10
	p.Generations = 5
	r, err := Compare(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Groups[0].Runs) != 5 || r.Unit != "cents" {
		t.Fatal(r)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Compare(ctx, req); err == nil {
		t.Fatal("cancel ignored")
	}
}
