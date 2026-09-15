package logic

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"strings"
	"time"

	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
	"simple_tuan/pkg/ga"
	"simple_tuan/pkg/tsp"
)

type candidate struct {
	mask    uint64
	stops   []int
	metrics models.PlanMetrics
}

type planner struct {
	ctx   context.Context
	req   models.PlanRequest
	m     campus.Map
	cache map[uint64]candidate
}

func ValidatePlan(req *models.PlanRequest) error {
	if req.Seed == 0 {
		req.Seed = 7
	}
	if req.Mode == "" {
		req.Mode = "business"
	}
	if !slices.Contains([]string{"business", "knapsack", "route"}, req.Mode) {
		return fmt.Errorf("未知规划模式")
	}
	if req.Population == 0 {
		req.Population = 60
	}
	if req.Generations == 0 {
		req.Generations = 100
	}
	badBudget := req.Population < 4 || req.Population > 200 || req.Generations < 1 || req.Generations > 500
	if badBudget || req.Population*req.Generations > 30000 {
		return fmt.Errorf("GA 参数超出预算（最多 30000 次个体评估）")
	}
	if req.Seed < 0 || req.Seed > 9007199254740991 {
		return fmt.Errorf("算法种子须为非负安全整数")
	}
	if req.CapacityGrams < 1 || req.CapacityGrams > 10000 {
		return fmt.Errorf("载重须为 1～10000 克")
	}
	checks := []struct{ value, low, high float64 }{
		{value: req.MaxMinutes, low: 1, high: 240}, {value: req.SpeedKPH, low: 1, high: 30},
		{value: req.CostCentsPerKM, low: 0, high: 10000}, {value: req.CostCentsPerMinute, low: 0, high: 10000},
	}
	for _, check := range checks {
		bad := math.IsNaN(check.value) || math.IsInf(check.value, 0) || check.value < check.low || check.value > check.high
		if bad {
			return fmt.Errorf("时长、速度或成本参数超出范围")
		}
	}
	m := campus.Default()
	if req.Batch.MapVersion != m.Version {
		return fmt.Errorf("地图版本不匹配，请重新载入订单")
	}
	if len(req.Batch.Orders) > 50 {
		return fmt.Errorf("最多支持 50 笔校园订单")
	}
	ids := map[string]bool{}
	for _, o := range req.Batch.Orders {
		badID := strings.TrimSpace(o.ID) == "" || len(o.ID) > 60 || ids[o.ID]
		badPlace := o.DestinationID < 0 || o.DestinationID >= len(m.Places)
		badWeight := o.WeightGrams < 1 || o.WeightGrams > 10000
		badFee := o.DeliveryFeeCents < 1 || o.DeliveryFeeCents > 100000
		if badID || badPlace || badWeight || badFee {
			return fmt.Errorf("订单编号、地址、重量或收入非法")
		}
		if m.DepotMeters[o.DestinationID] < 0 {
			return fmt.Errorf("订单 %s 的送达点不可达", o.ID)
		}
		if o.ServiceSeconds < 0 || o.ServiceSeconds > 600 {
			return fmt.Errorf("交付时间须为 0～600 秒")
		}
		ids[o.ID] = true
	}
	return nil
}

func (p *planner) measure(mask uint64, stops []int) models.PlanMetrics {
	v := models.PlanMetrics{}
	var seconds int
	for i, o := range p.req.Batch.Orders {
		if mask&(1<<i) == 0 {
			continue
		}
		v.IncomeCents += o.DeliveryFeeCents
		v.WeightGrams += o.WeightGrams
		seconds += o.ServiceSeconds
	}
	for i, stop := range stops {
		v.DistanceMeters += p.m.Distances[stop][stops[(i+1)%len(stops)]]
	}
	v.Minutes = v.DistanceMeters/(p.req.SpeedKPH*1000/60) + float64(seconds)/60
	v.CostCents = v.DistanceMeters/1000*p.req.CostCentsPerKM + v.Minutes*p.req.CostCentsPerMinute
	v.NetCents = float64(v.IncomeCents) - v.CostCents
	v.Feasible = v.WeightGrams <= p.req.CapacityGrams && v.Minutes <= p.req.MaxMinutes
	return v
}

// evaluate deterministically prices a set using nearest-neighbor and bounded 2-opt.
func (p *planner) evaluate(mask uint64) candidate {
	if c, ok := p.cache[mask]; ok {
		return c
	}
	remaining := map[int]bool{}
	for i, o := range p.req.Batch.Orders {
		if mask&(1<<i) != 0 && o.DestinationID != 0 {
			remaining[o.DestinationID] = true
		}
	}
	stops := []int{0}
	for len(remaining) > 0 {
		best, distance := -1, math.Inf(1)
		for id := 1; id < len(p.m.Places); id++ {
			if remaining[id] && p.m.Distances[stops[len(stops)-1]][id] < distance {
				best, distance = id, p.m.Distances[stops[len(stops)-1]][id]
			}
		}
		stops = append(stops, best)
		delete(remaining, best)
	}
	for range 4 {
		if !ga.TwoOptImprove(p.m.Distances, stops) {
			break
		}
	}
	stops = rotateDepot(stops)
	c := candidate{mask: mask, stops: stops, metrics: p.measure(mask, stops)}
	if len(p.cache) < 50000 {
		p.cache[mask] = c
	}
	return c
}

func (p *planner) score(c candidate) float64 {
	if p.req.Mode == "knapsack" {
		if c.metrics.WeightGrams > p.req.CapacityGrams {
			return math.Inf(-1)
		}
		return float64(c.metrics.IncomeCents)
	}
	if !c.metrics.Feasible {
		return math.Inf(-1)
	}
	return c.metrics.NetCents
}

// repair removes the least costly order per unit of constraint relief.
func (p *planner) repair(mask uint64) candidate {
	c := p.evaluate(mask)
	for mask != 0 && math.IsInf(p.score(c), -1) {
		best, loss := c, math.Inf(1)
		for i := range p.req.Batch.Orders {
			if mask&(1<<i) == 0 {
				continue
			}
			next := p.evaluate(mask &^ (1 << i))
			relief := float64(c.metrics.WeightGrams-next.metrics.WeightGrams) / float64(p.req.CapacityGrams)
			if p.req.Mode == "business" {
				relief += math.Max(0, c.metrics.Minutes-next.metrics.Minutes) / p.req.MaxMinutes
			}
			cost := float64(c.metrics.IncomeCents - next.metrics.IncomeCents)
			if p.req.Mode == "business" {
				cost = c.metrics.NetCents - next.metrics.NetCents
			}
			if ratio := cost / math.Max(relief, 0.0001); ratio < loss {
				best, loss = next, ratio
			}
		}
		c, mask = best, best.mask
	}
	return c
}

func (p *planner) refine(c candidate) (candidate, error) {
	if len(c.stops) < 4 {
		return c, nil
	}
	inst := &tsp.Instance{Name: "校园道路", EdgeType: "matrix", Cities: []tsp.City{}, Matrix: [][]float64{}}
	for i, id := range c.stops {
		inst.Cities = append(inst.Cities, tsp.City{ID: i, X: p.m.Places[id].X, Y: p.m.Places[id].Y})
		row := make([]float64, len(c.stops))
		for j, to := range c.stops {
			row[j] = p.m.Distances[id][to]
		}
		inst.Matrix = append(inst.Matrix, row)
	}
	params := ga.DefaultParams()
	params.Seed, params.Population, params.Generations = p.req.Seed, 60, 100
	params.Initialization, params.LocalSearch = "mixed", true
	r, err := ga.SolveContext(p.ctx, inst, params)
	if err != nil {
		return c, err
	}
	if err := inst.ValidateTour(r.BestTour); err != nil {
		return c, err
	}
	if r.BestDistance < c.metrics.DistanceMeters {
		stops := make([]int, len(c.stops))
		for i, index := range r.BestTour {
			stops[i] = c.stops[index]
		}
		c.stops = rotateDepot(stops)
		c.metrics = p.measure(c.mask, c.stops)
	}
	return c, nil
}

func (p *planner) search() (candidate, []float64, error) {
	n := len(p.req.Batch.Orders)
	rng := rand.New(rand.NewSource(p.req.Seed))
	pop := make([]candidate, p.req.Population)
	pop[0] = p.evaluate(0)
	for i := 1; i < len(pop); i++ {
		var mask uint64
		for j := range n {
			if rng.Intn(2) == 1 {
				mask |= 1 << j
			}
		}
		pop[i] = p.repair(mask)
	}
	best := pop[0]
	curve := make([]float64, 0, p.req.Generations)
	for range p.req.Generations {
		if err := p.ctx.Err(); err != nil {
			return best, curve, err
		}
		sort.SliceStable(pop, func(i, j int) bool { return p.score(pop[i]) > p.score(pop[j]) })
		if p.score(pop[0]) > p.score(best) {
			best = pop[0]
		}
		curve = append(curve, p.score(best))
		next := make([]candidate, len(pop))
		next[0], next[1] = best, pop[1]
		pick := func() candidate {
			a, b := pop[rng.Intn(len(pop))], pop[rng.Intn(len(pop))]
			if p.score(a) > p.score(b) {
				return a
			}
			return b
		}
		for i := 2; i < len(next); i++ {
			if err := p.ctx.Err(); err != nil {
				return best, curve, err
			}
			a, b := pick(), pick()
			mask := a.mask
			if rng.Float64() < 0.9 {
				cut := rng.Intn(n + 1)
				lower := uint64(1)<<cut - 1
				mask = a.mask&lower | b.mask&^lower
			}
			for j := range n {
				if rng.Float64() < 0.03 {
					mask ^= 1 << j
				}
			}
			next[i] = p.repair(mask)
		}
		pop = next
	}
	// Include the last offspring, then refine several distinct strong sets.
	pool := append(pop, best)
	sort.SliceStable(pool, func(i, j int) bool { return p.score(pool[i]) > p.score(pool[j]) })
	seen := map[uint64]bool{}
	for _, c := range pool {
		if seen[c.mask] {
			continue
		}
		seen[c.mask] = true
		r, err := p.refine(c)
		if err != nil {
			return best, curve, err
		}
		if p.score(r) > p.score(best) || (r.mask == best.mask && r.metrics.DistanceMeters < best.metrics.DistanceMeters) {
			best = r
		}
		if len(seen) == 5 {
			break
		}
	}
	curve = append(curve, p.score(best))
	return best, curve, nil
}

// Plan computes a complete immutable selection and route snapshot.
func Plan(ctx context.Context, req models.PlanRequest) (models.PlanResult, error) {
	start := time.Now()
	result := models.PlanResult{Selected: []models.CampusOrder{}, Excluded: []models.ExcludedOrder{}, Stops: []int{}, RoadPath: []int{}, Curve: []float64{}}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := ValidatePlan(&req); err != nil {
		return result, err
	}
	p := planner{ctx: ctx, req: req, m: campus.Default(), cache: map[uint64]candidate{}}
	result.Input = req
	result.CurveUnit = "分"
	var best candidate
	var err error
	if req.Mode == "route" {
		mask := uint64(1)<<len(req.Batch.Orders) - 1
		base := p.evaluate(mask)
		result.Baseline, result.BaselineName = base.metrics, "最近邻 + 2-opt"
		best, err = p.refine(base)
		result.CurveUnit = "米"
		result.Curve = []float64{base.metrics.DistanceMeters, best.metrics.DistanceMeters}
	} else {
		best, result.Curve, err = p.search()
	}
	if err != nil {
		return result, err
	}
	if req.Mode == "business" && !best.metrics.Feasible {
		return result, fmt.Errorf("未获得可行方案")
	}
	result.Stops, result.Metrics = best.stops, p.measure(best.mask, best.stops)
	for i, o := range req.Batch.Orders {
		if best.mask&(1<<i) != 0 {
			result.Selected = append(result.Selected, o)
			continue
		}
		reason := "未入选当前方案"
		if o.WeightGrams > req.CapacityGrams {
			reason = "超过本趟载重上限"
		}
		result.Excluded = append(result.Excluded, models.ExcludedOrder{ID: o.ID, Reason: reason})
	}
	result.RoadPath = []int{p.m.Places[0].NodeID}
	result.Legs = []models.CampusLeg{}
	for i, from := range best.stops {
		to := best.stops[(i+1)%len(best.stops)]
		path, err := p.m.Path(from, to)
		if err != nil {
			return result, err
		}
		result.RoadPath = append(result.RoadPath, path[1:]...)
		if from != to {
			result.Legs = append(result.Legs, models.CampusLeg{
				From: from, To: to, DistanceMeters: p.m.Distances[from][to], RoadPath: path,
			})
		}
	}
	if req.Mode == "knapsack" {
		bag := make([]int, req.CapacityGrams+1)
		for _, o := range req.Batch.Orders {
			for c := req.CapacityGrams; c >= o.WeightGrams; c-- {
				bag[c] = max(bag[c], bag[c-o.WeightGrams]+o.DeliveryFeeCents)
			}
		}
		result.OptimalIncomeCents = bag[req.CapacityGrams]
		result.VerifiedOptimal = result.Metrics.IncomeCents == result.OptimalIncomeCents
	}
	result.ElapsedMS = float64(time.Since(start).Microseconds()) / 1000
	return result, nil
}
