package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
	"time"
)

func registerCampus(api *gin.RouterGroup) {
	api.POST("/campus/plan", handleCampusPlan)
	api.GET("/campus/map", func(c *gin.Context) { writeJSON(c, 200, campus.Default()) })
	api.POST("/campus/batches/generate", func(c *gin.Context) {
		var req models.BatchRequest
		if err := decodeDecision(c, &req); err != nil {
			writeError(c, 400, err.Error())
			return
		}
		batch, err := logic.GenerateBatch(req)
		if err != nil {
			writeError(c, 400, err.Error())
			return
		}
		writeJSON(c, 200, batch)
	})
}

func handleCampusPlan(c *gin.Context) {
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
	writeJSON(c, 200, result)
}
