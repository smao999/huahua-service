package router

import (
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func registerJcti(rg *gin.RouterGroup, h *handler.JctiHandler, mw *middleware.Middleware) {
	g := rg.Group("/jcti", mw.AuthRequired)
	g.POST("/analyze", h.Analyze)
}
