package logic

import (
	"fmt"
	"math/rand"
	"slices"
	"sort"
	"time"

	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
)

func GenerateBatch(req models.BatchRequest) (models.OrderBatch, error) {
	batch := models.OrderBatch{Orders: []models.CampusOrder{}}
	if req.Seed == 0 {
		req.Seed = time.Now().UnixMilli()
	}
	if req.Seed < 0 || req.Seed > 9007199254740991 {
		return batch, fmt.Errorf("种子须为非负安全整数")
	}
	rng := rand.New(rand.NewSource(req.Seed))
	if req.Count == 0 {
		req.Count = 20 + rng.Intn(31)
	}
	if req.Count < 20 || req.Count > 50 {
		return batch, fmt.Errorf("随机订单数量须为 20～50")
	}
	if req.Scenario == "" {
		req.Scenario = "uniform"
	}
	if !slices.Contains([]string{"uniform", "clustered", "near", "far", "outlier", "same-place"}, req.Scenario) {
		return batch, fmt.Errorf("未知订单场景")
	}
	m := campus.Default()
	eligible := []int{}
	dorms := []int{}
	for _, place := range m.Places {
		if place.ID == 0 || m.DepotMeters[place.ID] < 0 {
			continue
		}
		eligible = append(eligible, place.ID)
		if place.Category == "dorm" {
			dorms = append(dorms, place.ID)
		}
	}
	if len(eligible) == 0 {
		return batch, fmt.Errorf("没有可达配送点")
	}
	sort.SliceStable(eligible, func(i, j int) bool { return m.DepotMeters[eligible[i]] < m.DepotMeters[eligible[j]] })
	near := eligible[:min(4, len(eligible))]
	far := eligible[max(0, len(eligible)-4):]
	if len(dorms) == 0 {
		dorms = near
	}
	samePlace := dorms[rng.Intn(len(dorms))]
	batch.MapVersion, batch.Seed, batch.Scenario = m.Version, req.Seed, req.Scenario
	for i := range req.Count {
		destination, fee := eligible[rng.Intn(len(eligible))], 300+rng.Intn(1001)
		switch req.Scenario {
		case "clustered":
			destination = dorms[rng.Intn(len(dorms))]
		case "near":
			destination, fee = near[rng.Intn(len(near))], 200+rng.Intn(301)
		case "far":
			destination, fee = far[rng.Intn(len(far))], 800+rng.Intn(1001)
		case "outlier":
			destination = near[rng.Intn(len(near))]
			if i%5 == 0 {
				destination, fee = far[len(far)-1], 100
			}
		case "same-place":
			destination = samePlace
		}
		batch.Orders = append(batch.Orders, models.CampusOrder{
			ID: fmt.Sprintf("O%03d", i+1), DestinationID: destination,
			WeightGrams: 200 + 100*rng.Intn(14), DeliveryFeeCents: fee, ServiceSeconds: 30 + rng.Intn(61),
		})
	}
	return batch, nil
}
