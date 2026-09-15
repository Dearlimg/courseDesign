package controller

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"simple_tuan/internal/dao"
	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
)

func registerSnapshots(api *gin.RouterGroup, store logic.CampusStore) {
	api.GET("/campus/history", func(c *gin.Context) {
		kind := c.Query("kind")
		if kind != "batch" && kind != "plan" {
			writeError(c, 400, "类型须为 batch 或 plan")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		rows, err := store.List(ctx, c.GetInt64("userID"), kind)
		if err != nil {
			writeError(c, 503, "历史记录读取失败")
			return
		}
		writeJSON(c, 200, rows)
	})
	api.GET("/campus/history/:id", func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			writeError(c, 400, "非法记录编号")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		row, err := store.Get(ctx, c.GetInt64("userID"), id)
		if errors.Is(err, dao.ErrNotFound) {
			writeError(c, 404, "记录不存在")
			return
		}
		if err != nil {
			writeError(c, 503, "历史记录读取失败")
			return
		}
		writeJSON(c, 200, row)
	})
	api.POST("/campus/batches/save", func(c *gin.Context) {
		var batch models.OrderBatch
		if err := decodeDecision(c, &batch); err != nil {
			writeError(c, 400, err.Error())
			return
		}
		req := models.PlanRequest{Batch: batch, CapacityGrams: 10000, MaxMinutes: 60, SpeedKPH: 12}
		if err := logic.ValidatePlan(&req); err != nil {
			writeError(c, 400, err.Error())
			return
		}
		saveSnapshot(c, store, "batch", batch)
	})
	// Persist only results computed on the server, never client-supplied metrics.
	api.POST("/campus/plans/save", func(c *gin.Context) {
		var req models.PlanRequest
		if err := decodeDecision(c, &req); err != nil {
			writeError(c, 400, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()
		result, err := logic.Plan(ctx, req)
		if err != nil {
			writeError(c, 400, err.Error())
			return
		}
		saveSnapshot(c, store, "plan", result)
	})
}

func saveSnapshot(c *gin.Context, store logic.CampusStore, kind string, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		writeError(c, 500, "快照编码失败")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	row, err := store.Save(ctx, c.GetInt64("userID"), kind, body)
	if err != nil {
		writeError(c, 503, "保存失败，请重试")
		return
	}
	writeJSON(c, 201, row)
}
