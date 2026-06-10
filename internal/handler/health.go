package handler

import (
	"context"
	"huahua-service/internal/db"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Ping(c *gin.Context) {
	components := gin.H{}

	dbStatus := "ok"
	if db.GORM != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var result int
		if err := db.GORM.WithContext(ctx).Raw("select 1").Scan(&result); err != nil {
			dbStatus = "error"
		}
	} else {
		dbStatus = "error"
	}
	components["db"] = dbStatus

	redisStatus := "ok"
	if db.Redis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.Redis.Ping(ctx).Err(); err != nil {
			redisStatus = "error"
		}
	} else {
		redisStatus = "error"
	}
	components["redis"] = redisStatus

	statusCode := http.StatusOK
	status := "ok"
	if dbStatus == "error" || redisStatus == "error" {
		statusCode = http.StatusServiceUnavailable
		status = "error"
	}

	c.JSON(statusCode, gin.H{"status": status, "components": components})
}
