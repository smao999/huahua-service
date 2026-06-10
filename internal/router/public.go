package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerPublic(rg *gin.RouterGroup, h *handler.PublicHandler) {
	g := rg.Group("/public")
	g.GET("/blog-stats", h.BlogStats)
}
