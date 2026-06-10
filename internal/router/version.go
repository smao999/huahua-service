package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerVersion(rg *gin.RouterGroup, h *handler.VersionHandler) {
	g := rg.Group("/version")
	g.GET("", h.Latest)
}
