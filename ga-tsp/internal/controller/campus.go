package controller

import (
	"github.com/gin-gonic/gin"
	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
	"simple_tuan/pkg/campus"
)

func registerCampus(api *gin.RouterGroup) {
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
