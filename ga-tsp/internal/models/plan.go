package models

type PlanRequest struct {
	Batch              OrderBatch `json:"batch"`
	Mode               string     `json:"mode"` // business, knapsack, route
	CapacityGrams      int        `json:"capacityGrams"`
	MaxMinutes         float64    `json:"maxMinutes"`
	SpeedKPH           float64    `json:"speedKph"`
	CostCentsPerKM     float64    `json:"costCentsPerKm"`
	CostCentsPerMinute float64    `json:"costCentsPerMinute"`
	Seed               int64      `json:"seed"`
	Population         int        `json:"population"`
	Generations        int        `json:"generations"`
}

type PlanMetrics struct {
	IncomeCents    int     `json:"incomeCents"`
	WeightGrams    int     `json:"weightGrams"`
	DistanceMeters float64 `json:"distanceMeters"`
	Minutes        float64 `json:"minutes"`
	CostCents      float64 `json:"costCents"`
	NetCents       float64 `json:"netCents"`
	Feasible       bool    `json:"feasible"`
}

type RouteFrame struct {
	Gen            int         `json:"gen"`
	Phase          string      `json:"phase"`
	DistanceMeters float64     `json:"distanceMeters"`
	Fitness        float64     `json:"fitness"`
	Stops          []int       `json:"stops"`
	Legs           []CampusLeg `json:"legs"`
}

type PlanResult struct {
	RouteFrames        []RouteFrame    `json:"routeFrames"`
	Legs               []CampusLeg     `json:"legs"`
	Input              PlanRequest     `json:"input"`
	Selected           []CampusOrder   `json:"selected"`
	Excluded           []ExcludedOrder `json:"excluded"`
	Stops              []int           `json:"stops"`    // Depot first, without duplicate terminal depot.
	RoadPath           []int           `json:"roadPath"` // Includes terminal depot and intermediate road nodes.
	Metrics            PlanMetrics     `json:"metrics"`
	Curve              []float64       `json:"curve"`
	CurveUnit          string          `json:"curveUnit"`
	OptimalIncomeCents int             `json:"optimalIncomeCents,omitempty"`
	VerifiedOptimal    bool            `json:"verifiedOptimal"`
	Baseline           PlanMetrics     `json:"baseline"`
	BaselineName       string          `json:"baselineName"`
	ElapsedMS          float64         `json:"elapsedMs"`
}

type CampusLeg struct {
	From           int     `json:"from"` // Place ID, not road node ID.
	To             int     `json:"to"`
	DistanceMeters float64 `json:"distanceMeters"`
	RoadPath       []int   `json:"roadPath"` // Indices into map.nodes.
}
