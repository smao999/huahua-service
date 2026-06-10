package handler

import (
	"context"
	"huahua-service/internal/db"
	"log"
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
		tx := db.GORM.WithContext(ctx).Raw("select 1").Scan(&result)
		if tx.Error != nil {
			dbStatus = "error"
		}
	} else {
		log.Printf(dbStatus)
		dbStatus = "error"
	}
	components["db"] = dbStatus

	redisStatus := "ok"
	if db.Redis != nil {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		if err := db.Redis.Ping(ctx2).Err(); err != nil {
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
