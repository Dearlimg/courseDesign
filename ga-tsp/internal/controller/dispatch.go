package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
)

func handleRoute(c *gin.Context) {
	var req models.RouteRequest
	if err := decodeDecision(c, &req); err != nil {
		writeError(c, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	result, err := logic.Route(ctx, req)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	writeJSON(c, 200, result)
}

// decodeDecision 严格解析请求体：未知字段与多段 JSON 均拒绝。
func decodeDecision(c *gin.Context, v any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("请求格式错误：%w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("请求须包含单个 JSON 对象")
	}
	return nil
}

func handleSelection(c *gin.Context) {
	var req models.SelectionRequest
	if err := decodeDecision(c, &req); err != nil {
		writeError(c, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	result, err := logic.Select(ctx, req)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	writeJSON(c, 200, result)
}
