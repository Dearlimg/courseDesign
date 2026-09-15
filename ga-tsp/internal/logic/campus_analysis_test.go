package logic

import (
	"context"
	"math"
	"simple_tuan/internal/models"
	"testing"
)

func TestCampusComparison(t *testing.T) {
	req := planFixture()
	req.Batch.Orders = []models.CampusOrder{
		{ID: "near", DestinationID: 1, WeightGrams: 500, DeliveryFeeCents: 600},
		{ID: "remote", DestinationID: 11, WeightGrams: 500, DeliveryFeeCents: 100},
	}
	r, err := CompareCampus(context.Background(), models.CampusCompareRequest{Plan: req, Seeds: []int64{7, 17}})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Groups) != 3 || r.ExactNetCents == nil || math.Abs(*r.ExactNetCents-418) > 0.001 {
		t.Fatal("exact comparison mismatch")
	}
	if r.Groups[2].MeanNetCents <= r.Groups[0].MeanNetCents {
		t.Fatal("business should avoid remote detour")
	}
	for _, g := range r.Groups {
		mean := (g.Runs[0].Metrics.NetCents + g.Runs[1].Metrics.NetCents) / 2
		std := math.Abs(g.Runs[0].Metrics.NetCents-g.Runs[1].Metrics.NetCents) / math.Sqrt(2)
		if mean != g.MeanNetCents || math.Abs(std-g.StdDevNetCents) > 0.001 {
			t.Fatal("invalid statistics")
		}
	}
	if _, err := CompareCampus(context.Background(), models.CampusCompareRequest{Plan: req, Seeds: []int64{7, 7}}); err == nil {
		t.Fatal("duplicate seeds")
	}
}
