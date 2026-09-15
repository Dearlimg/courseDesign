package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

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

func registerExperiments(api *gin.RouterGroup) {
	api.GET("/knapsack/instance", func(c *gin.Context) {
		n := 24
		if raw := c.Query("n"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeError(c, 400, "物品数量须为整数")
				return
			}
			n = parsed
		}
		seed := int64(42)
		if raw := c.Query("seed"); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				writeError(c, 400, "种子须为整数")
				return
			}
			seed = parsed
		}
		k, err := optimization.RandomKnapsack(n, seed)
		if err != nil {
			writeError(c, 400, err.Error())
			return
		}
		writeJSON(c, 200, k)
	})
	api.GET("/cec/functions", func(c *gin.Context) {
		writeJSON(c, 200, optimization.Benchmarks())
	})
	api.GET("/cec/landscape", func(c *gin.Context) {
		dimension, err := strconv.Atoi(c.Query("dimension"))
		if err != nil {
			writeError(c, 400, "维数须为整数")
			return
		}
		cec, err := optimization.NewCEC(c.Query("function"), dimension)
		if err != nil {
			writeError(c, 400, err.Error())
			return
		}
		writeJSON(c, 200, map[string]any{"instance": cec, "grid": cec.Landscape()})
	})
	api.POST("/experiments/solve", handleExperimentSolve)
	api.POST("/experiments/scan", handleExperimentScan)
}

func decodeExperiment(c *gin.Context) (experimentRequest, error) {
	var req experimentRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
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
		cec, err := optimization.NewCEC(req.Function, req.Dimension)
		if err != nil {
			return optimization.Result{}, err
		}
		return optimization.SolveCEC(ctx, cec, req.Params)
	default:
		return optimization.Result{}, fmt.Errorf("未知问题类型")
	}
}

func handleExperimentSolve(c *gin.Context) {
	req, err := decodeExperiment(c)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	result, err := solveExperiment(ctx, req)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	writeJSON(c, 200, result)
}

type scanResult struct {
	Value        float64   `json:"value"`
	Best         float64   `json:"best"`
	Error        float64   `json:"error"`
	ConvergedGen int       `json:"convergedGen"`
	ElapsedMs    float64   `json:"elapsedMs"`
	Curve        []float64 `json:"curve"`
}

func handleExperimentScan(c *gin.Context) {
	req, err := decodeExperiment(c)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	if len(req.Values) < 1 || len(req.Values) > 8 {
		writeError(c, 400, "对比组数须在 1～8 之间")
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
				writeError(c, 400, "种群规模须为 4～500 的整数")
				return
			}
			current.Params.Population = int(v)
		case "elitism":
			if math.Trunc(v) != v || v < 0 || v > 499 {
				writeError(c, 400, "精英数须为 0～499 的整数")
				return
			}
			current.Params.Elitism = int(v)
		default:
			writeError(c, 400, "不支持的对比参数")
			return
		}
		if err := current.Params.Validate(current.Problem); err != nil {
			writeError(c, 400, err.Error())
			return
		}
		budget += current.Params.Population * current.Params.Generations
		requests = append(requests, current)
	}
	if budget > 1500000 {
		writeError(c, 400, "对比实验总评估次数不能超过 1500000，请降低代数或种群规模")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 75*time.Second)
	defer cancel()
	rows := make([]scanResult, 0, len(requests))
	for i, current := range requests {
		res, err := solveExperiment(ctx, current)
		if err != nil {
			writeError(c, 400, err.Error())
			return
		}
		curve := make([]float64, len(res.Generations))
		for j, g := range res.Generations {
			curve[j] = g.BestSoFar
		}
		rows = append(rows, scanResult{Value: req.Values[i], Best: res.Best, Error: math.Abs(res.Best - res.Optimal),
			ConvergedGen: res.ConvergedGen, ElapsedMs: res.ElapsedMs, Curve: curve})
	}
	writeJSON(c, 200, map[string]any{"results": rows})
}
