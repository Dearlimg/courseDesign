package logic

import (
	"context"
	"testing"

	"simple_tuan/internal/models"
)

func exampleOrders() []models.Order {
	return []models.Order{
		{ID: "A", Name: "宿舍一", X: 20, Y: 25, Load: 4, Income: 1200},
		{ID: "B", Name: "宿舍二", X: 65, Y: 20, Load: 3, Income: 1000},
		{ID: "C", Name: "图书馆", X: 80, Y: 70, Load: 2, Income: 700},
		{ID: "D", Name: "实验楼", X: 30, Y: 85, Load: 5, Income: 1400},
		{ID: "E", Name: "教学楼", X: 50, Y: 50, Load: 3, Income: 800},
	}
}

func TestSelection(t *testing.T) {
	r, err := Select(context.Background(), models.SelectionRequest{Orders: exampleOrders(), Capacity: 10})
	if err != nil {
		t.Fatal(err)
	}
	if r.OptimalIncome != 3100 || r.Income != 3100 || r.Load != 10 || !r.VerifiedOptimal {
		t.Fatalf("unexpected result: %+v", r)
	}
	tests := []struct {
		name             string
		orders           []models.Order
		capacity, income int
	}{
		{name: "single cents", orders: []models.Order{{ID: "A", Name: "点", Load: 1, Income: 101}}, capacity: 1, income: 101},
		{name: "oversized", orders: []models.Order{{ID: "A", Name: "点", Load: 2, Income: 101}}, capacity: 1},
		{name: "empty", orders: []models.Order{}, capacity: 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := Select(context.Background(), models.SelectionRequest{Orders: tc.orders, Capacity: tc.capacity})
			if err != nil || r.Income != tc.income {
				t.Fatalf("%+v %v", r, err)
			}
		})
	}
}
func TestSelectionRejectsInvalid(t *testing.T) {
	orders := exampleOrders()
	orders[1].ID = orders[0].ID
	if _, err := Select(context.Background(), models.SelectionRequest{Orders: orders, Capacity: 10}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, err := Select(context.Background(), models.SelectionRequest{Capacity: 0}); err == nil {
		t.Fatal("zero capacity accepted")
	}
	orders = exampleOrders()
	orders[0].Income = -1
	if err := ValidateOrders(orders); err == nil {
		t.Fatal("negative income accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Select(ctx, models.SelectionRequest{Orders: exampleOrders(), Capacity: 10}); err == nil {
		t.Fatal("canceled request accepted")
	}
}
