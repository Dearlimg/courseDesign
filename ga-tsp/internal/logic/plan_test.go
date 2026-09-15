package logic

import (
	"context"
	"math"
	"reflect"
	"testing"

	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
)

func planFixture() models.PlanRequest {
	return models.PlanRequest{
		Batch: models.OrderBatch{MapVersion: campus.Default().Version, Orders: []models.CampusOrder{}},
		Mode:  "business", CapacityGrams: 1000, MaxMinutes: 60, SpeedKPH: 12,
		CostCentsPerKM: 500, CostCentsPerMinute: 0, Seed: 7, Population: 30, Generations: 30,
	}
}

func TestBusinessEconomics(t *testing.T) {
	req := planFixture()
	req.Batch.Orders = []models.CampusOrder{
		{ID: "near", DestinationID: 1, WeightGrams: 500, DeliveryFeeCents: 600},
		{ID: "remote", DestinationID: 11, WeightGrams: 500, DeliveryFeeCents: 100},
	}
	r, err := Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Selected) != 1 || r.Selected[0].ID != "near" || math.Abs(r.Metrics.NetCents-418) > 0.01 {
		t.Fatalf("remote low-fee order should be excluded: %+v", r)
	}
	req.Batch.Orders = []models.CampusOrder{
		{ID: "a", DestinationID: 11, WeightGrams: 500, DeliveryFeeCents: 800},
		{ID: "b", DestinationID: 11, WeightGrams: 500, DeliveryFeeCents: 800},
	}
	r, err = Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Selected) != 2 || len(r.Stops) != 2 || r.Metrics.NetCents <= 0 {
		t.Fatal("same-place orders must share route cost")
	}
	req.Batch.Orders = req.Batch.Orders[:1]
	r, err = Plan(context.Background(), req)
	if err != nil || len(r.Selected) != 0 || r.Metrics.NetCents != 0 {
		t.Fatal("unprofitable trip must be empty")
	}
}

func TestPlanConstraintsAndReproducibility(t *testing.T) {
	req := planFixture()
	req.Batch, _ = GenerateBatch(models.BatchRequest{Count: 50, Seed: 123})
	req.MaxMinutes = 10
	a, err := Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	a.ElapsedMS, b.ElapsedMS = 0, 0
	if !reflect.DeepEqual(a, b) {
		t.Fatal("fixed-seed mismatch")
	}
	if !a.Metrics.Feasible || a.Metrics.WeightGrams > req.CapacityGrams || a.Metrics.Minutes > req.MaxMinutes {
		t.Fatal("invalid constraints")
	}
	if a.RoadPath[0] != 0 || a.RoadPath[len(a.RoadPath)-1] != 0 {
		t.Fatal("route is not closed")
	}
	m := campus.Default()
	var distance float64
	for i := 1; i < len(a.RoadPath); i++ {
		from, to := a.RoadPath[i-1], a.RoadPath[i]
		found := false
		for _, road := range m.Roads {
			forward := road.From == from && road.To == to
			reverse := road.To == from && road.From == to
			if forward || reverse {
				distance += road.Meters
				found = true
				break
			}
		}
		if !found {
			t.Fatal("path crosses missing road")
		}
	}
	if math.Abs(distance-a.Metrics.DistanceMeters) > 0.001 {
		t.Fatal("displayed path differs from optimization distance")
	}
	var legDistance float64
	for i, leg := range a.Legs {
		if leg.From != a.Stops[i] || leg.To != a.Stops[(i+1)%len(a.Stops)] {
			t.Fatal("leg sequence differs from stop sequence")
		}
		legDistance += leg.DistanceMeters
		if leg.RoadPath[0] != m.Places[leg.From].NodeID {
			t.Fatal("leg begins at the wrong entrance")
		}
	}
	if math.Abs(legDistance-distance) > 0.001 {
		t.Fatal("leg totals differ from trip distance")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Plan(ctx, req); err == nil {
		t.Fatal("ignored cancellation")
	}
	for i := range req.Batch.Orders {
		req.Batch.Orders[i].WeightGrams = 10000
	}
	r, err := Plan(context.Background(), req)
	if err != nil || len(r.Selected) != 0 {
		t.Fatal("overweight orders accepted")
	}
}

func TestRejectPreviousCampusVersion(t *testing.T) {
	req := planFixture()
	req.Batch.MapVersion = "xupt-simulation-v1"
	if _, err := Plan(context.Background(), req); err == nil {
		t.Fatal("old IDs silently remapped to new campus")
	}
}

func TestKnapsackAndRouteModes(t *testing.T) {
	req := planFixture()
	req.Mode = "knapsack"
	req.Batch.Orders = []models.CampusOrder{
		{ID: "a", DestinationID: 1, WeightGrams: 600, DeliveryFeeCents: 800},
		{ID: "b", DestinationID: 9, WeightGrams: 400, DeliveryFeeCents: 600},
		{ID: "c", DestinationID: 8, WeightGrams: 500, DeliveryFeeCents: 650},
	}
	r, err := Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !r.VerifiedOptimal || r.OptimalIncomeCents != 1400 {
		t.Fatalf("DP mismatch: %+v", r)
	}
	req.Mode = "route"
	r, err = Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Selected) != 3 || r.Metrics.DistanceMeters > r.Baseline.DistanceMeters {
		t.Fatal("route dropped orders or degraded baseline")
	}
}

func TestSmallBusinessExactOracle(t *testing.T) {
	req := planFixture()
	req.Batch.Orders = []models.CampusOrder{
		{ID: "a", DestinationID: 1, WeightGrams: 300, DeliveryFeeCents: 650},
		{ID: "b", DestinationID: 4, WeightGrams: 300, DeliveryFeeCents: 550},
		{ID: "c", DestinationID: 11, WeightGrams: 500, DeliveryFeeCents: 100},
		{ID: "d", DestinationID: 5, WeightGrams: 300, DeliveryFeeCents: 450},
	}
	r, err := Plan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	p := planner{req: req, m: campus.Default()}
	var optimal float64
	for mask := uint64(0); mask < 1<<len(req.Batch.Orders); mask++ {
		stops := []int{0}
		for i, o := range req.Batch.Orders {
			if mask&(1<<i) != 0 {
				stops = append(stops, o.DestinationID)
			}
		}
		var permute func(int)
		permute = func(at int) {
			if at == len(stops) {
				v := p.measure(mask, stops)
				if v.Feasible {
					optimal = math.Max(optimal, v.NetCents)
				}
				return
			}
			for i := at; i < len(stops); i++ {
				stops[at], stops[i] = stops[i], stops[at]
				permute(at + 1)
				stops[at], stops[i] = stops[i], stops[at]
			}
		}
		permute(1)
	}
	if math.Abs(r.Metrics.NetCents-optimal) > 0.001 {
		t.Fatalf("small oracle: GA=%v exact=%v", r.Metrics.NetCents, optimal)
	}
}
