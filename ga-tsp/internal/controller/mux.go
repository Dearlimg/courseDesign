// Package controller 是 HTTP 协议适配层：
// 实例管理（内置/随机）、GA 求解、参数扫描与骑手调度 API。
package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"simple_tuan/pkg/ga"
	"simple_tuan/pkg/tsp"
)

// handleListInstances 返回内置实例列表。
func handleListInstances(c *gin.Context) {
	list := []map[string]any{
		{"name": "att48", "size": 48, "optimal": 10628.0, "edgeType": "att"},
	}
	writeJSON(c, http.StatusOK, list)
}

// handleGetInstance 按名称返回内置实例。
func handleGetInstance(c *gin.Context) {
	name := c.Param("name")
	if name != "att48" {
		writeError(c, http.StatusNotFound, "未知实例 "+name)
		return
	}
	writeJSON(c, http.StatusOK, tsp.Att48())
}

// handleRandomInstance 生成随机欧氏实例。
func handleRandomInstance(c *gin.Context) {
	var req struct {
		N    int   `json:"n"`
		Seed int64 `json:"seed"`
	}
	if err := decodeBody(c, &req); err != nil {
		return
	}
	if req.Seed == 0 {
		req.Seed = timeNowNanos()
	}
	inst, err := tsp.Random(req.N, req.Seed)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, inst)
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
func handleSolve(c *gin.Context) {
	var req solveRequest
	if err := decodeBody(c, &req); err != nil {
		return
	}
	inst, err := buildInstance(req.Instance)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.Params.Normalize(); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := ga.Solve(inst, req.Params)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(c, http.StatusOK, result)
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
func handleScan(c *gin.Context) {
	var req scanRequest
	if err := decodeBody(c, &req); err != nil {
		return
	}
	inst, err := buildInstance(req.Instance)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := req.Params.Normalize(); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	setter, ok := scanParamDefs[req.Param]
	if !ok {
		writeError(c, http.StatusBadRequest, "未知扫描参数 "+strconv.Quote(req.Param))
		return
	}
	if len(req.Values) == 0 || len(req.Values) > 8 {
		writeError(c, http.StatusBadRequest, "扫描值数量须在 [1,8]")
		return
	}

	results := make([]map[string]any, 0, len(req.Values))
	for _, v := range req.Values {
		p := req.Params
		setter(&p, v)
		result, err := ga.Solve(inst, p)
		if err != nil {
			writeError(c, http.StatusBadRequest, fmt.Sprintf("参数 %s=%v 求解失败: %v", req.Param, v, err))
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
	writeJSON(c, http.StatusOK, map[string]any{"results": results})
}

// decodeBody 解析 JSON 请求体，失败时写出 400 并返回错误。
func decodeBody(c *gin.Context, v any) error {
	if err := json.NewDecoder(c.Request.Body).Decode(v); err != nil {
		writeError(c, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return err
	}
	return nil
}
