package router

import (
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func registerAgentRequest(rg *gin.RouterGroup, h *handler.AgentRequestHandler, mw *middleware.Middleware) {
	g := rg.Group("/agent", mw.AuthRequired)
	g.POST("/request", h.Create)
	g.GET("/request", h.List)
	g.PUT("/request/:id", h.Update)
}
