package logic

import (
	"fmt"
	"math/rand"
	"slices"
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
	batch.MapVersion, batch.Seed, batch.Scenario = m.Version, req.Seed, req.Scenario
	for i := range req.Count {
		destination, fee := 1+rng.Intn(len(m.Places)-1), 300+rng.Intn(1001)
		switch req.Scenario {
		case "clustered":
			destination = 1 + rng.Intn(3)
		case "near":
			destination, fee = []int{1, 4, 5}[rng.Intn(3)], 200+rng.Intn(301)
		case "far":
			destination, fee = 8+rng.Intn(4), 800+rng.Intn(1001)
		case "outlier":
			destination = 1 + rng.Intn(2)
			if i%5 == 0 {
				destination, fee = 11, 100
			}
		case "same-place":
			destination = 9
		}
		batch.Orders = append(batch.Orders, models.CampusOrder{
			ID: fmt.Sprintf("O%03d", i+1), DestinationID: destination,
			WeightGrams: 200 + 100*rng.Intn(14), DeliveryFeeCents: fee, ServiceSeconds: 30 + rng.Intn(61),
		})
	}
	return batch, nil
}
