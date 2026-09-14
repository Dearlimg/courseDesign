package optimization

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
)

type Item struct {
	Weight int `json:"weight"`
	Value  int `json:"value"`
}

type Knapsack struct {
	Items    []Item `json:"items"`
	Capacity int    `json:"capacity"`
}

func (k Knapsack) Validate() error {
	if len(k.Items) < 2 || len(k.Items) > 100 {
		return fmt.Errorf("物品数量须在 2～100 之间")
	}
	if k.Capacity < 1 || k.Capacity > 10000 {
		return fmt.Errorf("容量须在 1～10000 之间")
	}
	for _, item := range k.Items {
		if item.Weight < 1 || item.Weight > 10000 {
			return fmt.Errorf("重量须为 1～10000 的整数")
		}
		if item.Value < 1 || item.Value > 100000 {
			return fmt.Errorf("价值须为 1～100000 的整数")
		}
	}
	return nil
}

func RandomKnapsack(n int, seed int64) (Knapsack, error) {
	if n < 2 || n > 100 {
		return Knapsack{}, fmt.Errorf("物品数量须在 2～100 之间")
	}
	rng := rand.New(rand.NewSource(seed))
	k := Knapsack{Items: make([]Item, n)}
	var total int
	for i := range k.Items {
		k.Items[i] = Item{Weight: 2 + rng.Intn(28), Value: 5 + rng.Intn(95)}
		total += k.Items[i].Weight
	}
	k.Capacity = max(1, total*35/100)
	return k, nil
}

// ExactValue 使用容量倒序动态规划，确保每件物品只能使用一次。
func (k Knapsack) ExactValue() int {
	dp := make([]int, k.Capacity+1)
	for _, item := range k.Items {
		for c := k.Capacity; c >= item.Weight; c-- {
			dp[c] = max(dp[c], dp[c-item.Weight]+item.Value)
		}
	}
	return dp[k.Capacity]
}

func (k Knapsack) value(genes []float64) float64 {
	var total int
	for i, v := range genes {
		if v == 1 {
			total += k.Items[i].Value
		}
	}
	return float64(total)
}

func SolveKnapsack(ctx context.Context, k Knapsack, params Params) (Result, error) {
	if err := k.Validate(); err != nil {
		return Result{}, err
	}
	// 按价值密度升序剔除超重物品。修复仅保证可行性，不注入动态规划最优解。
	order := make([]int, len(k.Items))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := k.Items[order[i]], k.Items[order[j]]
		return a.Value*b.Weight < b.Value*a.Weight
	})
	repair := func(genes []float64) {
		var weight int
		for i, g := range genes {
			if g == 1 {
				weight += k.Items[i].Weight
			}
		}
		for _, i := range order {
			if weight <= k.Capacity {
				break
			}
			if genes[i] == 1 {
				genes[i] = 0
				weight -= k.Items[i].Weight
			}
		}
	}
	task := problem{kind: "knapsack", size: len(k.Items), lower: 0, upper: 1,
		optimal: float64(k.ExactValue()), evaluate: k.value, repair: repair}
	return run(ctx, task, params)
}
