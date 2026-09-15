package ga

import (
	"context"
	"math"
	"math/rand"
	"slices"
	"sort"
	"time"

	"simple_tuan/pkg/tsp"
)

func nearestTour(dm [][]float64, start int) []int {
	tour := []int{start}
	used := make([]bool, len(dm))
	used[start] = true
	for len(tour) < len(dm) {
		next := -1
		for i := range dm {
			if used[i] {
				continue
			}
			if next == -1 || dm[start][i] < dm[start][next] {
				next = i
			}
		}
		tour = append(tour, next)
		used[next] = true
		start = next
	}
	return tour
}

// decodeKeys sorts by key, breaking ties by the stable city identifier.
func decodeKeys(keys []float64) []int {
	tour := make([]int, len(keys))
	for i := range tour {
		tour[i] = i
	}
	sort.SliceStable(tour, func(i, j int) bool { return keys[tour[i]] < keys[tour[j]] })
	return tour
}

func encodeTour(tour []int) []float64 {
	keys := make([]float64, len(tour))
	for rank, id := range tour {
		keys[id] = float64(rank) / float64(len(tour))
	}
	return keys
}

func keyParent(dist []float64, p Params, rng *rand.Rand) int {
	if p.Selection == "roulette" {
		var total float64
		for _, d := range dist {
			if d == 0 {
				return slices.Index(dist, 0)
			}
			total += 1 / d
		}
		pick := rng.Float64() * total
		for i, d := range dist {
			pick -= 1 / d
			if pick <= 0 {
				return i
			}
		}
		return len(dist) - 1
	}
	best := rng.Intn(len(dist))
	for range 2 {
		i := rng.Intn(len(dist))
		if dist[i] < dist[best] {
			best = i
		}
	}
	return best
}

func evolveKeys(keys [][]float64, dist []float64, p Params, rng *rand.Rand) [][]float64 {
	order := make([]int, len(keys))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return dist[order[i]] < dist[order[j]] })
	next := make([][]float64, 0, len(keys))
	for _, id := range order[:p.Elitism] {
		next = append(next, slices.Clone(keys[id]))
	}
	for len(next) < len(keys) {
		a := keyParent(dist, p, rng)
		b := keyParent(dist, p, rng)
		child := slices.Clone(keys[a])
		if rng.Float64() < p.CrossoverRate {
			for i := range child {
				if rng.Intn(2) == 0 {
					child[i] = keys[b][i]
				}
			}
		}
		for i := range child {
			if rng.Float64() < p.MutationRate {
				child[i] = rng.Float64()
			}
		}
		next = append(next, child)
	}
	return next
}

func solveKeys(ctx context.Context, inst *tsp.Instance, p Params) (Result, error) {
	started := time.Now()
	rng := rand.New(rand.NewSource(p.Seed))
	dm := inst.DistanceMatrix()
	keys := make([][]float64, p.Population)
	for i := range keys {
		keys[i] = make([]float64, inst.Size())
		for j := range keys[i] {
			keys[i][j] = rng.Float64()
		}
		if p.Initialization == "mixed" && i < p.Population/2 {
			keys[i] = encodeTour(nearestTour(dm, rng.Intn(inst.Size())))
		}
	}
	result := Result{Generations: []Generation{}, BestTour: []int{}, BestDistance: math.Inf(1), Params: p}
	for gen := range p.Generations {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		tours := make([][]int, len(keys))
		distances := make([]float64, len(keys))
		var sum float64
		for i, k := range keys {
			tours[i] = decodeKeys(k)
			distances[i] = tourLength(dm, tours[i])
			sum += distances[i]
		}
		best := argMin(distances)
		if p.LocalSearch && TwoOptImprove(dm, tours[best]) {
			d := tourLength(dm, tours[best])
			sum += d - distances[best]
			distances[best] = d
			keys[best] = encodeTour(tours[best])
		}
		if distances[best] < result.BestDistance {
			result.BestDistance = distances[best]
			result.BestTour = slices.Clone(tours[best])
			result.ConvergedGen = gen
		}
		samples := [][]int{slices.Clone(tours[best])}
		for i := max(1, len(keys)/4); i < len(keys) && len(samples) < 4; i += max(1, len(keys)/4) {
			samples = append(samples, slices.Clone(tours[i]))
		}
		result.Generations = append(result.Generations, Generation{
			Gen: gen, Best: distances[best], Avg: sum / float64(len(keys)), Worst: slices.Max(distances),
			Diversity: uniqueRatio(tours), BestTour: slices.Clone(tours[best]), SampleTours: samples,
		})
		if gen+1 < p.Generations {
			keys = evolveKeys(keys, distances, p, rng)
		}
	}
	result.ElapsedMs = float64(time.Since(started).Microseconds()) / 1000
	return result, nil
}
