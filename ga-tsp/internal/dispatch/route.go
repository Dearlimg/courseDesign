package dispatch

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"simple_tuan/pkg/ga"
	"simple_tuan/pkg/tsp"
)

type Point struct {
	Name     string   `json:"name"`
	X        float64  `json:"x"`
	Y        float64  `json:"y"`
	OrderIDs []string `json:"orderIds,omitempty"`
}
type RouteRequest struct {
	Depot  Point      `json:"depot"`
	Points []Point    `json:"points"`
	Params *ga.Params `json:"params,omitempty"`
}
type RouteResult struct {
	Stops            []Point   `json:"stops"`
	Tour             []int     `json:"tour"`
	Distance         float64   `json:"distance"`
	BaselineDistance float64   `json:"baselineDistance"`
	Improvement      float64   `json:"improvement"`
	UsedBaseline     bool      `json:"usedBaseline"`
	Method           string    `json:"method"`
	Evolution        ga.Result `json:"evolution"`
}

func DefaultRouteParams() ga.Params {
	p := ga.DefaultParams()
	p.Population = 100
	p.Generations = 200
	p.Seed = 7
	return p
}

// RouteInstance merges equal coordinates, including deliveries at the depot.
func RouteInstance(req RouteRequest) (*tsp.Instance, []Point, error) {
	if len(req.Points) == 0 || len(req.Points) > 100 {
		return nil, []Point{}, fmt.Errorf("送达点数量须为 1～100")
	}
	stops := []Point{}
	indices := make(map[[2]float64]int)
	points := append([]Point{req.Depot}, req.Points...)
	for _, p := range points {
		if strings.TrimSpace(p.Name) == "" || len([]rune(p.Name)) > 100 {
			return nil, stops, fmt.Errorf("地点名称不能为空或过长")
		}
		if !validCoordinate(p.X) || !validCoordinate(p.Y) {
			return nil, stops, fmt.Errorf("坐标须在 0～1000 之间")
		}
		if len(p.OrderIDs) > 100 {
			return nil, stops, fmt.Errorf("地点关联订单过多")
		}
		key := [2]float64{p.X, p.Y}
		if i, ok := indices[key]; ok {
			stops[i].OrderIDs = append(stops[i].OrderIDs, p.OrderIDs...)
			continue
		}
		indices[key] = len(stops)
		p.OrderIDs = append([]string{}, p.OrderIDs...)
		stops = append(stops, p)
	}
	cities := make([]tsp.City, len(stops))
	for i, p := range stops {
		cities[i] = tsp.City{ID: i, X: p.X, Y: p.Y}
	}
	return &tsp.Instance{Name: "园区配送", EdgeType: "euclid", Cities: cities}, stops, nil
}

func Route(ctx context.Context, req RouteRequest) (RouteResult, error) {
	result := RouteResult{Stops: []Point{}, Tour: []int{}, Evolution: ga.Result{Generations: []ga.Generation{}, BestTour: []int{}}}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	inst, stops, err := RouteInstance(req)
	if err != nil {
		return result, err
	}
	params := DefaultRouteParams()
	if req.Params != nil {
		params = *req.Params
	}
	if err := params.Normalize(); err != nil {
		return result, err
	}
	if params.Population*params.Generations > 500000 {
		return result, fmt.Errorf("单次路线评估次数不能超过 500000")
	}
	result.Stops = stops
	baseline := make([]int, inst.Size())
	for i := range baseline {
		baseline[i] = i
	}
	result.BaselineDistance = inst.TourLength(baseline)
	result.Tour = slices.Clone(baseline)
	result.Distance = result.BaselineDistance
	result.Method = "direct"
	if inst.Size() >= 4 {
		result.Method = "ga"
		evolution, err := ga.SolveContext(ctx, inst, params)
		if err != nil {
			return result, err
		}
		result.Evolution = evolution
		if err := inst.ValidateTour(evolution.BestTour); err != nil {
			return result, err
		}
		actual := inst.TourLength(evolution.BestTour)
		if actual <= result.Distance {
			result.Tour = rotateDepot(evolution.BestTour)
			result.Distance = actual
		} else {
			result.UsedBaseline = true
		}
	}
	if result.BaselineDistance > 0 {
		result.Improvement = (result.BaselineDistance - result.Distance) / result.BaselineDistance
	}
	return result, nil
}
func rotateDepot(tour []int) []int {
	i := slices.Index(tour, 0)
	result := append([]int{}, tour[i:]...)
	return append(result, tour[:i]...)
}
