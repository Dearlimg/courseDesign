package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"simple_tuan/pkg/optimization"
)

type experimentRequest struct {
	Problem   string                `json:"problem"`
	Knapsack  optimization.Knapsack `json:"knapsack"`
	Function  string                `json:"function"`
	Dimension int                   `json:"dimension"`
	Params    optimization.Params   `json:"params"`
	ScanParam string                `json:"scanParam"`
	Values    []float64             `json:"values"`
}

func registerExperiments(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/knapsack/instance", func(w http.ResponseWriter, r *http.Request) {
		n := 24
		if raw := r.URL.Query().Get("n"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, 400, "物品数量须为整数")
				return
			}
			n = parsed
		}
		seed := int64(42)
		if raw := r.URL.Query().Get("seed"); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				writeError(w, 400, "种子须为整数")
				return
			}
			seed = parsed
		}
		k, err := optimization.RandomKnapsack(n, seed)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, k)
	})
	mux.HandleFunc("GET /api/cec/functions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, optimization.Benchmarks())
	})
	mux.HandleFunc("GET /api/cec/landscape", func(w http.ResponseWriter, r *http.Request) {
		dimension, err := strconv.Atoi(r.URL.Query().Get("dimension"))
		if err != nil {
			writeError(w, 400, "维数须为整数")
			return
		}
		c, err := optimization.NewCEC(r.URL.Query().Get("function"), dimension)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"instance": c, "grid": c.Landscape()})
	})
	mux.HandleFunc("POST /api/experiments/solve", handleExperimentSolve)
	mux.HandleFunc("POST /api/experiments/scan", handleExperimentScan)
}

func decodeExperiment(w http.ResponseWriter, r *http.Request) (experimentRequest, error) {
	var req experimentRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return req, fmt.Errorf("请求格式错误: %w", err)
	}
	return req, nil
}

func solveExperiment(ctx context.Context, req experimentRequest) (optimization.Result, error) {
	switch req.Problem {
	case "knapsack":
		return optimization.SolveKnapsack(ctx, req.Knapsack, req.Params)
	case "cec":
		c, err := optimization.NewCEC(req.Function, req.Dimension)
		if err != nil {
			return optimization.Result{}, err
		}
		return optimization.SolveCEC(ctx, c, req.Params)
	default:
		return optimization.Result{}, fmt.Errorf("未知问题类型")
	}
}

func handleExperimentSolve(w http.ResponseWriter, r *http.Request) {
	req, err := decodeExperiment(w, r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	result, err := solveExperiment(ctx, req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

type scanResult struct {
	Value        float64   `json:"value"`
	Best         float64   `json:"best"`
	Error        float64   `json:"error"`
	ConvergedGen int       `json:"convergedGen"`
	ElapsedMs    float64   `json:"elapsedMs"`
	Curve        []float64 `json:"curve"`
}

func handleExperimentScan(w http.ResponseWriter, r *http.Request) {
	req, err := decodeExperiment(w, r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if len(req.Values) < 1 || len(req.Values) > 8 {
		writeError(w, 400, "对比组数须在 1～8 之间")
		return
	}
	requests := make([]experimentRequest, 0, len(req.Values))
	var budget int
	for _, v := range req.Values {
		current := req
		switch req.ScanParam {
		case "mutationRate":
			current.Params.MutationRate = v
		case "crossoverRate":
			current.Params.CrossoverRate = v
		case "population":
			if math.Trunc(v) != v || v < 4 || v > 500 {
				writeError(w, 400, "种群规模须为 4～500 的整数")
				return
			}
			current.Params.Population = int(v)
		case "elitism":
			if math.Trunc(v) != v || v < 0 || v > 499 {
				writeError(w, 400, "精英数须为 0～499 的整数")
				return
			}
			current.Params.Elitism = int(v)
		default:
			writeError(w, 400, "不支持的对比参数")
			return
		}
		if err := current.Params.Validate(current.Problem); err != nil {
			writeError(w, 400, err.Error())
			return
		}
		budget += current.Params.Population * current.Params.Generations
		requests = append(requests, current)
	}
	if budget > 1500000 {
		writeError(w, 400, "对比实验总评估次数不能超过 1500000，请降低代数或种群规模")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
	defer cancel()
	rows := make([]scanResult, 0, len(requests))
	for i, current := range requests {
		res, err := solveExperiment(ctx, current)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}
		curve := make([]float64, len(res.Generations))
		for j, g := range res.Generations {
			curve[j] = g.BestSoFar
		}
		rows = append(rows, scanResult{Value: req.Values[i], Best: res.Best, Error: math.Abs(res.Best - res.Optimal),
			ConvergedGen: res.ConvergedGen, ElapsedMs: res.ElapsedMs, Curve: curve})
	}
	writeJSON(w, 200, map[string]any{"results": rows})
}
