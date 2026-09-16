package logic

import (
	"context"
	"math"
	"reflect"
	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
	"testing"
)

func TestRouteReplayMatchesRoadsAndFinalPlan(t *testing.T) {
	for _, mode := range []string{"route", "business", "knapsack"} {
		t.Run(mode, func(t *testing.T) {
			req := planFixture()
			req.Mode, req.CapacityGrams, req.MaxMinutes = mode, 10000, 240
			req.Batch, _ = GenerateBatch(models.BatchRequest{Count: 30, Seed: 123})
			result, err := Plan(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.RouteFrames) < 2 {
				t.Fatal("missing generation history")
			}
			previous := math.Inf(1)
			for _, frame := range result.RouteFrames {
				if frame.DistanceMeters > previous+1e-7 {
					t.Fatal("incumbent regressed")
				}
				previous = frame.DistanceMeters
				if frame.Stops[0] != 0 {
					t.Fatal("route must start at depot")
				}
				seen := map[int]bool{}
				for _, stop := range frame.Stops {
					if seen[stop] {
						t.Fatal("duplicate stop")
					}
					seen[stop] = true
				}
				for _, order := range result.Selected {
					if !seen[order.DestinationID] {
						t.Fatal("selected destination omitted")
					}
				}
				distance := 0.0
				for i, leg := range frame.Legs {
					if leg.From != frame.Stops[i] || leg.To != frame.Stops[(i+1)%len(frame.Stops)] {
						t.Fatal("leg order mismatch")
					}
					path, err := campus.Default().Path(leg.From, leg.To)
					if err != nil || !reflect.DeepEqual(path, leg.RoadPath) {
						t.Fatal("incorrect road geometry")
					}
					distance += leg.DistanceMeters
				}
				if math.Abs(distance-frame.DistanceMeters) > 1e-7 {
					t.Fatal("distance mismatch")
				}
				if math.Abs(frame.Fitness-1000/(1+distance)) > 1e-7 {
					t.Fatal("fitness mismatch")
				}
			}
			last := result.RouteFrames[len(result.RouteFrames)-1]
			if !reflect.DeepEqual(last.Stops, result.Stops) || math.Abs(last.DistanceMeters-result.Metrics.DistanceMeters) > 1e-7 {
				t.Fatal("final frame differs from result")
			}
		})
	}
}

func TestRouteReplaySmallRoutes(t *testing.T) {
	for _, destination := range []int{0, 1} {
		req := planFixture()
		req.Mode = "route"
		req.Batch.Orders = []models.CampusOrder{{ID: "one", DestinationID: destination, WeightGrams: 1, DeliveryFeeCents: 100}}
		result, err := Plan(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.RouteFrames) != 1 {
			t.Fatal("small routes need a single truthful snapshot")
		}
	}
}
