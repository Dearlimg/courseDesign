package api

import (
	"context"
	"net/http"
	"time"

	"simple_tuan/internal/analysis"
)

func handleCompare(w http.ResponseWriter, r *http.Request) {
	var req analysis.Request
	if err := decodeDecision(w, r, &req); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
	defer cancel()
	result, err := analysis.Compare(ctx, req)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 200, result)
}
