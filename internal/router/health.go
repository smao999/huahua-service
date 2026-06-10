package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerHealth(rg *gin.RouterGroup, h *handler.HealthHandler) {
	g := rg.Group("/health")
	g.GET("", h.Ping)
}
