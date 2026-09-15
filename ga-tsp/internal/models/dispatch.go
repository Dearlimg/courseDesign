package models

import (
	"simple_tuan/pkg/ga"
	"simple_tuan/pkg/optimization"
)

// Order 是一趟配送的候选订单。
type Order struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Load   int     `json:"load"`
	Income int     `json:"income"`
}

// ExcludedOrder 说明订单未入选的原因。
type ExcludedOrder struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// SelectionRequest 是智能接单请求。
type SelectionRequest struct {
	Orders   []Order              `json:"orders"`
	Capacity int                  `json:"capacity"`
	Params   *optimization.Params `json:"params,omitempty"`
}

// SelectionResult 是智能接单结果。
type SelectionResult struct {
	Selected        []Order             `json:"selected"`
	Eligible        []Order             `json:"eligible"`
	Excluded        []ExcludedOrder     `json:"excluded"`
	Load            int                 `json:"load"`
	Income          int                 `json:"income"`
	OptimalIncome   int                 `json:"optimalIncome"`
	VerifiedOptimal bool                `json:"verifiedOptimal"`
	Method          string              `json:"method"`
	Evolution       optimization.Result `json:"evolution"`
}

// Point 是配送路线上的停靠点（同址订单合并后）。
type Point struct {
	Name     string   `json:"name"`
	X        float64  `json:"x"`
	Y        float64  `json:"y"`
	OrderIDs []string `json:"orderIds,omitempty"`
}

// RouteRequest 是配送路线规划请求。
type RouteRequest struct {
	Depot  Point      `json:"depot"`
	Points []Point    `json:"points"`
	Params *ga.Params `json:"params,omitempty"`
}

// RouteResult 是配送路线规划结果。
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
