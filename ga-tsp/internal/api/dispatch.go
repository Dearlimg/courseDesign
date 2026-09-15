package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
)

func registerDispatch(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/analysis/compare", handleCompare)
	mux.HandleFunc("POST /api/dispatch/route", handleRoute)
	mux.HandleFunc("POST /api/dispatch/select", handleSelection)
}

func handleRoute(w http.ResponseWriter, r *http.Request) {
	var req models.RouteRequest
	if err := decodeDecision(w, r, &req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	result, err := logic.Route(ctx, req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, result)
}

func decodeDecision(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("请求格式错误：%w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("请求须包含单个 JSON 对象")
	}
	return nil
}

func handleSelection(w http.ResponseWriter, r *http.Request) {
	var req models.SelectionRequest
	if err := decodeDecision(w, r, &req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	result, err := logic.Select(ctx, req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, result)
}
