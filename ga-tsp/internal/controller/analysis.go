package controller

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"simple_tuan/internal/logic"
	"simple_tuan/internal/models"
)

func handleCompare(c *gin.Context) {
	var req models.Request
	if err := decodeDecision(c, &req); err != nil {
		writeError(c, 400, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 75*time.Second)
	defer cancel()
	result, err := logic.Compare(ctx, req)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	writeJSON(c, 200, result)
}
