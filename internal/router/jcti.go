package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerJcti(rg *gin.RouterGroup, h *handler.JctiHandler) {
	g := rg.Group("/jcti")
	g.POST("/analyze", h.Analyze)
}
