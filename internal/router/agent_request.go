package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerAgentRequest(rg *gin.RouterGroup, h *handler.AgentRequestHandler) {
	g := rg.Group("/agent")
	g.POST("/request", h.Create)
	g.GET("/request", h.List)
	g.PUT("/request/:id", h.Update)
}
