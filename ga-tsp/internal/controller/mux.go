// Package api 提供 GA-TSP 系统的 REST 接口：
// 实例管理（内置/随机）、GA 求解、参数扫描对比实验。
package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"simple_tuan/pkg/ga"
	"simple_tuan/pkg/tsp"
)

// NewMux 构建路由并注册所有 API 端点。
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	registerExperiments(mux)
	registerDispatch(mux)
	mux.HandleFunc("GET /api/instances", handleListInstances)
	mux.HandleFunc("GET /api/instances/{name}", handleGetInstance)
	mux.HandleFunc("POST /api/instance/random", handleRandomInstance)
	mux.HandleFunc("POST /api/solve", handleSolve)
	mux.HandleFunc("POST /api/scan", handleScan)
	return mux
}

// handleListInstances 返回内置实例列表。
func handleListInstances(w http.ResponseWriter, _ *http.Request) {
	list := []map[string]any{
		{"name": "att48", "size": 48, "optimal": 10628.0, "edgeType": "att"},
	}
	writeJSON(w, http.StatusOK, list)
}

// handleGetInstance 按名称返回内置实例。
func handleGetInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name != "att48" {
		writeError(w, http.StatusNotFound, "未知实例 "+name)
		return
	}
	writeJSON(w, http.StatusOK, tsp.Att48())
}

// handleRandomInstance 生成随机欧氏实例。
func handleRandomInstance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		N    int   `json:"n"`
		Seed int64 `json:"seed"`
	}
	if err := decodeBody(w, r, &req); err != nil {
		return
	}
	if req.Seed == 0 {
		req.Seed = timeNowNanos()
	}
	inst, err := tsp.Random(req.N, req.Seed)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, inst)
}

// instancePayload 是请求中的实例数据。
type instancePayload struct {
	Name     string     `json:"name"`
	EdgeType string     `json:"edgeType"`
	Cities   []tsp.City `json:"cities"`
}

// solveRequest 是 GA 求解请求。
type solveRequest struct {
	Instance instancePayload `json:"instance"`
	Params   ga.Params       `json:"params"`
}

// buildInstance 将请求负载转为 TSP 实例（忽略客户端传入的名称与已知最优）。
func buildInstance(p instancePayload) (*tsp.Instance, error) {
	if len(p.Cities) < 4 {
		return nil, fmt.Errorf("城市数至少为 4，实际 %d", len(p.Cities))
	}
	if p.EdgeType != "att" && p.EdgeType != "euclid" {
		return nil, fmt.Errorf("边类型必须为 att 或 euclid，实际 %q", p.EdgeType)
	}
	for i, c := range p.Cities {
		if c.ID != i {
			return nil, fmt.Errorf("城市编号应从 0 连续递增，第 %d 个为 %d", i, c.ID)
		}
	}
	return &tsp.Instance{
		Name:     p.Name,
		EdgeType: p.EdgeType,
		Cities:   p.Cities,
	}, nil
}

// handleSolve 运行 GA 并返回逐代快照与最终结果。
func handleSolve(w http.ResponseWriter, r *http.Request) {
	var req solveRequest
	if err := decodeBody(w, r, &req); err != nil {
		return
	}
	inst, err := buildInstance(req.Instance)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.Params.Normalize(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := ga.Solve(inst, req.Params)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// scanRequest 是参数扫描请求：固定其他参数，仅变化一个参数观察收敛差异。
type scanRequest struct {
	Instance instancePayload `json:"instance"`
	Params   ga.Params       `json:"params"`
	Param    string          `json:"param"`
	Values   []float64       `json:"values"`
}

// 可扫描的参数及其设置方式。
var scanParamDefs = map[string]func(*ga.Params, float64){
	"population":    func(p *ga.Params, v float64) { p.Population = int(v) },
	"generations":   func(p *ga.Params, v float64) { p.Generations = int(v) },
	"crossoverRate": func(p *ga.Params, v float64) { p.CrossoverRate = v },
	"mutationRate":  func(p *ga.Params, v float64) { p.MutationRate = v },
	"elitism":       func(p *ga.Params, v float64) { p.Elitism = int(v) },
}

// handleScan 对同一实例用相同种子跑多组参数，返回精简的收敛数据。
func handleScan(w http.ResponseWriter, r *http.Request) {
	var req scanRequest
	if err := decodeBody(w, r, &req); err != nil {
		return
	}
	inst, err := buildInstance(req.Instance)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.Params.Normalize(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	setter, ok := scanParamDefs[req.Param]
	if !ok {
		writeError(w, http.StatusBadRequest, "未知扫描参数 "+strconv.Quote(req.Param))
		return
	}
	if len(req.Values) == 0 || len(req.Values) > 8 {
		writeError(w, http.StatusBadRequest, "扫描值数量须在 [1,8]")
		return
	}

	results := make([]map[string]any, 0, len(req.Values))
	for _, v := range req.Values {
		p := req.Params
		setter(&p, v)
		result, err := ga.Solve(inst, p)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("参数 %s=%v 求解失败: %v", req.Param, v, err))
			return
		}
		bestPerGen := make([]float64, len(result.Generations))
		for i, g := range result.Generations {
			bestPerGen[i] = g.Best
		}
		results = append(results, map[string]any{
			"label":        fmt.Sprintf("%s=%v", req.Param, v),
			"bestDistance": result.BestDistance,
			"convergedGen": result.ConvergedGen,
			"elapsedMs":    result.ElapsedMs,
			"bestPerGen":   bestPerGen,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

// decodeBody 解析 JSON 请求体，失败时写出 400 并返回错误。
func decodeBody(w http.ResponseWriter, r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return err
	}
	return nil
}
