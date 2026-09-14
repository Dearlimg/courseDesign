// Package api 提供 REST 接口：预置迷宫、随机生成、迷宫求解。
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"mazeweb/internal/maze"
	"mazeweb/internal/solver"
)

// NewMux 构建路由并注册所有 API 端点。
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/preset", handlePreset)
	mux.HandleFunc("POST /api/maze/generate", handleGenerate)
	mux.HandleFunc("POST /api/solve", handleSolve)
	return mux
}

// handlePreset 返回课程设计指导书题目十二的示例迷宫。
func handlePreset(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"maze": maze.Preset().Grid})
}

// generateRequest 是随机生成迷宫的请求。
type generateRequest struct {
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Density float64 `json:"density"`
	Seed    *int64  `json:"seed"`
}

// handleGenerate 生成随机迷宫。未指定 seed 时使用当前时间，保证每次不同。
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	var req generateRequest
	if err := decodeBody(w, r, &req); err != nil {
		return
	}
	if req.Width == 0 && req.Height == 0 {
		// 未传尺寸时使用默认值 15x15、密度 0.3
		req.Width, req.Height, req.Density = 15, 15, 0.3
	}
	if req.Density == 0 {
		req.Density = 0.3
	}
	seed := timeNowNanos()
	if req.Seed != nil {
		seed = *req.Seed
	}
	m, err := maze.Random(req.Width, req.Height, req.Density, seed)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"maze": m.Grid, "seed": seed})
}

// solveRequest 是求解迷宫的请求。
type solveRequest struct {
	Maze      [][]int `json:"maze"`
	Algorithm string  `json:"algorithm"`
}

// handleSolve 用指定算法求解迷宫，返回路径、探索过程与统计信息。
func handleSolve(w http.ResponseWriter, r *http.Request) {
	var req solveRequest
	if err := decodeBody(w, r, &req); err != nil {
		return
	}
	m, err := maze.New(req.Maze)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Algorithm == "" {
		req.Algorithm = "bfs"
	}
	res, err := solver.Solve(m, req.Algorithm)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// decodeBody 解析 JSON 请求体，失败时写出 400 并返回错误。
func decodeBody(w http.ResponseWriter, r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return err
	}
	return nil
}

// writeJSON 写出 JSON 响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// 响应已开始写入，只能记录
		_ = err
	}
}

// writeError 写出统一的错误响应。
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// timeNowNanos 返回当前时间的纳秒数。
func timeNowNanos() int64 { return time.Now().UnixNano() }
