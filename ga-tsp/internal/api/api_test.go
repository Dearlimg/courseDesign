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

func TestListInstances(t *testing.T) {
	rec := doJSON(t, http.MethodGet, "/api/instances", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", rec.Code)
	}
	var got []struct {
		Name    string  `json:"name"`
		Size    int     `json:"size"`
		Optimal float64 `json:"optimal"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("实例列表不应为空")
	}
	if got[0].Name != "att48" || got[0].Size != 48 || got[0].Optimal != 10628 {
		t.Fatalf("应包含 att48（48 城，最优 10628），实际 %+v", got[0])
	}
}

func TestGetInstance(t *testing.T) {
	rec := doJSON(t, http.MethodGet, "/api/instances/att48", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d", rec.Code)
	}
	var got struct {
		Name     string `json:"name"`
		EdgeType string `json:"edgeType"`
		Cities   []struct {
			ID int     `json:"id"`
			X  float64 `json:"x"`
			Y  float64 `json:"y"`
		} `json:"cities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if got.Name != "att48" || got.EdgeType != "att" || len(got.Cities) != 48 {
		t.Fatalf("att48 实例数据不正确: %s", rec.Body.String())
	}
}

func TestGetInstanceNotFound(t *testing.T) {
	rec := doJSON(t, http.MethodGet, "/api/instances/berlin52", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未知实例应返回 404，实际 %d", rec.Code)
	}
}

func TestRandomInstanceEndpoint(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/instance/random", map[string]any{
		"n": 20, "seed": 42,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		EdgeType string `json:"edgeType"`
		Cities   []struct {
			ID int `json:"id"`
		} `json:"cities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if got.EdgeType != "euclid" || len(got.Cities) != 20 {
		t.Fatal("应返回 20 城欧氏实例")
	}
}

func TestRandomInstanceRejectsBadSize(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/instance/random", map[string]any{"n": 2})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("城市数过少应返回 400，实际 %d", rec.Code)
	}
}

// 12 城小实例，求解参数压小以便测试快速完成
func smallInstance() map[string]any {
	cities := []map[string]any{}
	for i := 0; i < 12; i++ {
		cities = append(cities, map[string]any{
			"id": i,
			"x":  float64((i*37)%12) * 3,
			"y":  float64((i*53)%12) * 2,
		})
	}
	return map[string]any{"edgeType": "euclid", "cities": cities}
}

func smallParams() map[string]any {
	return map[string]any{
		"population":  30,
		"generations": 40,
		"seed":       7,
	}
}

func TestSolveEndpoint(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/solve", map[string]any{
		"instance": smallInstance(),
		"params":    smallParams(),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		BestDistance float64 `json:"bestDistance"`
		Generations  []struct {
			Gen      int     `json:"gen"`
			Best     float64 `json:"best"`
			BestTour []int   `json:"bestTour"`
		} `json:"generations"`
		ConvergedGen int `json:"convergedGen"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if got.BestDistance <= 0 || len(got.Generations) != 40 {
		t.Fatalf("求解结果不完整: %s", rec.Body.String())
	}
	if len(got.Generations[0].BestTour) != 12 {
		t.Fatal("每代快照应包含最优回路")
	}
}

func TestSolveEndpointRejectsBadParams(t *testing.T) {
	bad := map[string]any{
		"population":  1,
		"generations": 40,
	}
	rec := doJSON(t, http.MethodPost, "/api/solve", map[string]any{
		"instance": smallInstance(),
		"params":   bad,
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非法参数应返回 400，实际 %d", rec.Code)
	}
}

func TestSolveEndpointRejectsBadInstance(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/solve", map[string]any{
		"instance": map[string]any{"edgeType": "euclid", "cities": []map[string]any{}},
		"params":   smallParams(),
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("空实例应返回 400，实际 %d", rec.Code)
	}
}

func TestScanEndpoint(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/scan", map[string]any{
		"instance": smallInstance(),
		"params":    smallParams(),
		"param":     "mutationRate",
		"values":    []float64{0.01, 0.1},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码应为 200，实际 %d: %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Results []struct {
			Label        string    `json:"label"`
			BestDistance float64   `json:"bestDistance"`
			BestPerGen   []float64 `json:"bestPerGen"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应解析失败: %v", err)
	}
	if len(got.Results) != 2 {
		t.Fatalf("应返回 2 组扫描结果，实际 %d", len(got.Results))
	}
	for i, r := range got.Results {
		if r.Label == "" || r.BestDistance <= 0 || len(r.BestPerGen) != 40 {
			t.Fatalf("第 %d 组扫描结果不完整: %+v", i, r)
		}
	}
}

func TestScanEndpointRejectsUnknownParam(t *testing.T) {
	rec := doJSON(t, http.MethodPost, "/api/scan", map[string]any{
		"instance": smallInstance(),
		"params":   smallParams(),
		"param":    "unknown",
		"values":   []float64{1},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("未知扫描参数应返回 400，实际 %d", rec.Code)
	}
}
