package models

import (
	"simple_tuan/pkg/ga"
	"simple_tuan/pkg/optimization"
)

// Group 是重复实验的一组参数配置。
type Group struct {
	Name     string               `json:"name"`
	TSP      *ga.Params           `json:"tsp,omitempty"`
	Knapsack *optimization.Params `json:"knapsack,omitempty"`
}

// Request 是重复实验请求。
type Request struct {
	Problem   string            `json:"problem"`
	Route     *RouteRequest     `json:"route,omitempty"`
	Selection *SelectionRequest `json:"selection,omitempty"`
	Groups    []Group           `json:"groups"`
	Seeds     []int64           `json:"seeds"`
}

// Run 是一次重复实验的执行记录。
type Run struct {
	Seed      int64                `json:"seed"`
	Value     float64              `json:"value"`
	ElapsedMs float64              `json:"elapsedMs"`
	Curve     []float64            `json:"curve"`
	TSP       *ga.Params           `json:"tsp,omitempty"`
	Knapsack  *optimization.Params `json:"knapsack,omitempty"`
}

// Summary 是一组的统计汇总。
type Summary struct {
	Name          string    `json:"name"`
	Best          float64   `json:"best"`
	Mean          float64   `json:"mean"`
	StdDev        float64   `json:"stdDev"`
	MeanElapsedMs float64   `json:"meanElapsedMs"`
	MeanCurve     []float64 `json:"meanCurve"`
	Runs          []Run     `json:"runs"`
}

// Result 是重复实验结果。
type Result struct {
	Input  Request   `json:"input"`
	Unit   string    `json:"unit"`
	Groups []Summary `json:"groups"`
}
