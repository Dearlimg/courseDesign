package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"simple_tuan/internal/optimization"
)

func TestExperimentEndpoints(t *testing.T) {
	mux := NewMux()
	for _, url := range []string{"/api/knapsack/instance?n=12&seed=0", "/api/cec/functions",
		"/api/cec/landscape?function=f2&dimension=10"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", url, w.Code, w.Body.String())
		}
	}
	k, _ := optimization.RandomKnapsack(8, 42)
	p := optimization.Params{Population: 20, Generations: 10, CrossoverRate: .9, MutationRate: .1, Elitism: 2,
		Seed: 0, Selection: "rank", Initialization: "stratified", Crossover: "uniform", Mutation: "bitflip"}
	req := experimentRequest{Problem: "knapsack", Knapsack: k, Params: p, ScanParam: "mutationRate", Values: []float64{0, .1}}
	for _, endpoint := range []string{"solve", "scan"} {
		body, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/experiments/"+endpoint, bytes.NewReader(body)))
		if w.Code != 200 {
			t.Fatalf("%s: %s", endpoint, w.Body.String())
		}
	}
	req.Problem = "cec"
	req.Function = "f1"
	req.Dimension = 2
	req.Params.Crossover = "blend"
	req.Params.Mutation = "gaussian"
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/experiments/solve", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	req.Params.Population = -1
	body, _ = json.Marshal(req)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/experiments/solve", bytes.NewReader(body)))
	if w.Code != 400 {
		t.Fatal("未拒绝非法参数")
	}
}
