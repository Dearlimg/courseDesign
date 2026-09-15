package logic

import (
	"reflect"
	"simple_tuan/internal/models"
	"testing"
)

func TestBatchReproducible(t *testing.T) {
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
			if o.DestinationID < 1 || o.DestinationID > 11 {
				t.Fatal("invalid destination")
			}
		}
	}
	if _, err := GenerateBatch(models.BatchRequest{Count: 51}); err == nil {
		t.Fatal("accepted oversized batch")
	}
}
