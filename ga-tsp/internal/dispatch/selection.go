// Package dispatch maps delivery decisions to optimization models.
package dispatch

import (
	"context"
	"fmt"
	"math"
	"strings"

	"simple_tuan/pkg/optimization"
)

type Order struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Load   int     `json:"load"`
	Income int     `json:"income"`
}

func ValidateOrders(orders []Order) error {
	if len(orders) > 100 {
		return fmt.Errorf("最多支持 100 笔订单")
	}
	ids := make(map[string]bool)
	for _, o := range orders {
		invalidID := strings.TrimSpace(o.ID) == "" || len([]rune(o.ID)) > 60 || ids[o.ID]
		if invalidID {
			return fmt.Errorf("订单编号为空、重复或过长")
		}
		ids[o.ID] = true
		if strings.TrimSpace(o.Name) == "" || len([]rune(o.Name)) > 100 {
			return fmt.Errorf("送达点名称不能为空或超过 100 字符")
		}
		if !validCoordinate(o.X) || !validCoordinate(o.Y) {
			return fmt.Errorf("坐标须在 0～1000 之间")
		}
		if o.Load < 1 || o.Load > 10000 {
			return fmt.Errorf("容量占用须为 1～10000 的整数")
		}
		if o.Income < 1 || o.Income > 100000 {
			return fmt.Errorf("配送收入须为 1～100000 分的整数")
		}
	}
	return nil
}

func validCoordinate(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1000
}

func DefaultSelectionParams() optimization.Params {
	return optimization.Params{
		Population: 100, Generations: 200, CrossoverRate: 0.9, MutationRate: 0.03,
		Elitism: 2, Seed: 7, Selection: "tournament", Crossover: "onepoint",
		Mutation: "bitflip", Initialization: "random", GreedyRepair: true,
	}
}

type SelectionRequest struct {
	Orders   []Order              `json:"orders"`
	Capacity int                  `json:"capacity"`
	Params   *optimization.Params `json:"params,omitempty"`
}
type ExcludedOrder struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}
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

func Select(ctx context.Context, req SelectionRequest) (SelectionResult, error) {
	result := SelectionResult{
		Selected: []Order{}, Eligible: []Order{}, Excluded: []ExcludedOrder{},
		Evolution: optimization.Result{Generations: []optimization.Generation{}, Genes: []float64{}},
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := ValidateOrders(req.Orders); err != nil {
		return result, err
	}
	if req.Capacity < 1 || req.Capacity > 10000 {
		return result, fmt.Errorf("餐箱容量须为 1～10000")
	}
	params := DefaultSelectionParams()
	if req.Params != nil {
		params = *req.Params
	}
	if err := params.Validate("knapsack"); err != nil {
		return result, err
	}
	result.Evolution.Params = params
	bag := optimization.Knapsack{Capacity: req.Capacity, Items: []optimization.Item{}}
	for _, o := range req.Orders {
		if o.Load > req.Capacity {
			result.Excluded = append(result.Excluded, ExcludedOrder{ID: o.ID, Reason: "该订单超过本趟容量"})
			continue
		}
		result.Eligible = append(result.Eligible, o)
		bag.Items = append(bag.Items, optimization.Item{Weight: o.Load, Value: o.Income})
	}
	result.OptimalIncome = bag.ExactValue()
	if len(bag.Items) < 2 {
		result.Method = "direct"
		for _, o := range result.Eligible {
			result.Selected = append(result.Selected, o)
			result.Income += o.Income
			result.Load += o.Load
			result.Evolution.Genes = append(result.Evolution.Genes, 1)
		}
		result.Evolution.Best = float64(result.Income)
		result.Evolution.Optimal = float64(result.OptimalIncome)
	} else {
		result.Method = "ga"
		evolution, err := optimization.SolveKnapsack(ctx, bag, params)
		if err != nil {
			return result, err
		}
		result.Evolution = evolution
		if len(evolution.Genes) != len(result.Eligible) {
			return result, fmt.Errorf("算法结果不完整")
		}
		for i, g := range evolution.Genes {
			if g != 0 && g != 1 {
				return result, fmt.Errorf("算法返回非法接单编码")
			}
			o := result.Eligible[i]
			if g == 1 {
				result.Selected = append(result.Selected, o)
				result.Load += o.Load
				result.Income += o.Income
			} else {
				result.Excluded = append(result.Excluded, ExcludedOrder{ID: o.ID, Reason: "未入选当前推荐组合"})
			}
		}
	}
	if result.Load > req.Capacity {
		return result, fmt.Errorf("本次未得到有效接单建议：算法结果超出容量，请重新计算")
	}
	result.VerifiedOptimal = result.Income == result.OptimalIncome
	return result, nil
}
