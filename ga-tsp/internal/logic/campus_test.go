package logic

import (
	"reflect"
	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
	"testing"
)

func TestBatchReproducible(t *testing.T) {
	m := campus.Default()
	for _, scenario := range []string{"uniform", "clustered", "near", "far", "outlier", "same-place"} {
		req := models.BatchRequest{Seed: 42, Scenario: scenario}
		a, err := GenerateBatch(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := GenerateBatch(req)
		if !reflect.DeepEqual(a, b) || len(a.Orders) < 20 || len(a.Orders) > 50 {
			t.Fatal("batch reproducibility")
		}
		for _, o := range a.Orders {
			if o.DestinationID < 1 || o.DestinationID >= len(m.Places) {
				t.Fatal("invalid destination")
			}
		}
	}
	if _, err := GenerateBatch(models.BatchRequest{Count: 51}); err == nil {
		t.Fatal("accepted oversized batch")
	}
}

func TestCampusScenariosUseRoadDistances(t *testing.T) {
	m := campus.Default()
	near, err := GenerateBatch(models.BatchRequest{Count: 50, Seed: 7, Scenario: "near"})
	if err != nil {
		t.Fatal(err)
	}
	far, err := GenerateBatch(models.BatchRequest{Count: 50, Seed: 7, Scenario: "far"})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range near.Orders {
		for _, b := range far.Orders {
			if m.DepotMeters[a.DestinationID] >= m.DepotMeters[b.DestinationID] {
				t.Fatal("near/far classification must use road distances")
			}
		}
	}
	dorms, err := GenerateBatch(models.BatchRequest{Count: 50, Seed: 7, Scenario: "clustered"})
	if err != nil {
		t.Fatal(err)
	}
	for _, order := range dorms.Orders {
		if m.Places[order.DestinationID].Category != "dorm" {
			t.Fatal("dorm scenario includes a road junction or teaching building")
		}
	}
}
