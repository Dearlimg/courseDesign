package models

type CampusCompareRequest struct {
	Plan  PlanRequest `json:"plan"`
	Seeds []int64     `json:"seeds"`
}

type CampusRun struct {
	Seed      int64       `json:"seed"`
	Metrics   PlanMetrics `json:"metrics"`
	ElapsedMS float64     `json:"elapsedMs"`
	Curve     []float64   `json:"curve"`
	CurveUnit string      `json:"curveUnit"`
}

type CampusGroup struct {
	Name           string      `json:"name"`
	Runs           []CampusRun `json:"runs"`
	MeanNetCents   float64     `json:"meanNetCents"`
	StdDevNetCents float64     `json:"stdDevNetCents"`
	BestNetCents   float64     `json:"bestNetCents"`
	FeasibleRuns   int         `json:"feasibleRuns"`
}

type CampusComparison struct {
	Input         CampusCompareRequest `json:"input"`
	Groups        []CampusGroup        `json:"groups"`
	ExactNetCents *float64             `json:"exactNetCents,omitempty"`
}
