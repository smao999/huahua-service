package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, h *handler.HealthHandler) {
	api := r.Group("/api")
	registerHealth(api, h)
}
