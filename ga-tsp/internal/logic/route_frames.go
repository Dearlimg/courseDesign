package logic

import (
	"simple_tuan/internal/models"
	"slices"
)

// recordRoute captures actual incumbents on the same road network as the final plan.
func (p *planner) recordRoute(c candidate, gen int, phase string) {
	frame := models.RouteFrame{
		Gen: gen, Phase: phase, Stops: slices.Clone(c.stops),
		DistanceMeters: c.metrics.DistanceMeters,
		Fitness:        1000 / (1 + c.metrics.DistanceMeters), Legs: []models.CampusLeg{},
	}
	for i, from := range c.stops {
		to := c.stops[(i+1)%len(c.stops)]
		if from == to {
			continue
		}
		path, err := p.m.Path(from, to)
		if err != nil {
			continue
		} // Validated reachable places share this connected road network.
		frame.Legs = append(frame.Legs, models.CampusLeg{
			From: from, To: to, DistanceMeters: p.m.Distances[from][to], RoadPath: path,
		})
	}
	p.frames = append(p.frames, frame)
}
