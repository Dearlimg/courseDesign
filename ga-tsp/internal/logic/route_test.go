package logic

import (
	"context"
	"math"
	"testing"

	"simple_tuan/internal/models"
)

func TestRouteSmallAndMerge(t *testing.T) {
	r, err := Route(context.Background(), models.RouteRequest{
		Depot:  models.Point{Name: "站", X: 0, Y: 0},
		Points: []models.Point{{Name: "A", X: 3, Y: 4, OrderIDs: []string{"A"}}, {Name: "同址", X: 3, Y: 4, OrderIDs: []string{"B"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Distance != 10 || len(r.Stops) != 2 || len(r.Stops[1].OrderIDs) != 2 || r.Tour[0] != 0 {
		t.Fatalf("%+v", r)
	}
	r, err = Route(context.Background(), models.RouteRequest{Depot: models.Point{Name: "站"}, Points: []models.Point{{Name: "A", X: 3}, {Name: "B", Y: 4}}})
	if err != nil || r.Distance != 12 {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestRouteGAAndOrders(t *testing.T) {
	points := []models.Point{}
	for _, o := range exampleOrders() {
		points = append(points, models.Point{Name: o.Name, X: o.X, Y: o.Y, OrderIDs: []string{o.ID}})
	}
	r, err := Route(context.Background(), models.RouteRequest{Depot: models.Point{Name: "站", X: 10, Y: 10}, Points: points})
	if err != nil {
		t.Fatal(err)
	}
	if r.Distance > r.BaselineDistance || len(r.Tour) != 6 || r.Tour[0] != 0 {
		t.Fatalf("%+v", r)
	}
	seen := map[int]bool{}
	for _, i := range r.Tour {
		if seen[i] {
			t.Fatal("duplicate stop")
		}
		seen[i] = true
	}
	if _, err := Route(context.Background(), models.RouteRequest{Depot: models.Point{Name: "站"}, Points: []models.Point{{Name: "坏点", X: math.NaN()}}}); err == nil {
		t.Fatal("invalid point accepted")
	}
}
