package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func doJSON(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("构造请求失败: %v", err)
		}
		req = httptest.NewRequest(method, target, bytes.NewReader(raw))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	NewMux().ServeHTTP(rec, req)
	return rec
}

func TestPresetEndpoint(t *testing.T) {
	rec := doJSON(t, http.MethodGet, "/api/preset", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", rec.Code)
	}
	var got struct {
		Maze [][]int `json:"maze"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if len(got.Maze) != 5 || len(got.Maze[0]) != 5 {
		t.Fatalf("预置迷宫应为 5x5，实际 %dx%d", len(got.Maze), len(got.Maze[0]))
	}
}

func TestGenerateEndpoint(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/maze/generate", map[string]any{
		"width": 8, "height": 6, "density": 0.3, "seed": 42,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Maze [][]int `json:"maze"`
		Seed int64   `json:"seed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if len(got.Maze) != 6 || len(got.Maze[0]) != 8 {
		t.Fatalf("期望 8x6 迷宫，实际 %dx%d", len(got.Maze[0]), len(got.Maze))
	}
	if got.Maze[0][0] != 0 || got.Maze[5][7] != 0 {
		t.Fatal("起点和终点必须是通路")
	}
	if got.Seed != 42 {
		t.Fatalf("应回传使用的种子 42，实际 %d", got.Seed)
	}
}

func TestGenerateEndpointRejectsBadSize(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/maze/generate", map[string]any{
		"width": 0, "height": 6,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非法尺寸应返回 400，实际 %d", rec.Code)
	}
}

var presetMaze = [][]int{
	{0, 1, 0, 0, 1},
	{0, 0, 0, 0, 0},
	{1, 0, 0, 0, 1},
	{0, 1, 0, 1, 1},
	{0, 1, 0, 0, 0},
}

func TestSolveEndpointBFS(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/solve", map[string]any{
		"maze": presetMaze, "algorithm": "bfs",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Found       bool `json:"found"`
		PathLength  int  `json:"pathLength"`
		VisitedCount int `json:"visitedCount"`
		Steps        []struct {
			Cell struct {
				X int `json:"x"`
				Y int `json:"y"`
			} `json:"cell"`
			Action string `json:"action"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if !got.Found || got.PathLength != 8 {
		t.Fatalf("BFS 应找到 8 步最短路径，实际 found=%v len=%d", got.Found, got.PathLength)
	}
	if len(got.Steps) == 0 {
		t.Fatal("应返回探索过程步骤")
	}
}

func TestSolveEndpointRejectsInvalidMaze(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/solve", map[string]any{
		"maze": [][]int{{0, 2}, {1, 0}}, "algorithm": "bfs",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非法迷宫应返回 400，实际 %d", rec.Code)
	}
}

func TestSolveEndpointRejectsUnknownAlgorithm(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/solve", map[string]any{
		"maze": presetMaze, "algorithm": "astar",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("未知算法应返回 400，实际 %d", rec.Code)
	}
}

func TestSolveEndpointRejectsBadJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/solve", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	NewMux().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非法请求体应返回 400，实际 %d", rec.Code)
	}
}
