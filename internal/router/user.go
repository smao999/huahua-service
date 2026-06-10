package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerUser(rg *gin.RouterGroup, h *handler.UserHandler) {
	g := rg.Group("")
	g.POST("/danmaku/send", h.SendDanmaku)
	g.GET("/danmaku/list", h.ListDanmaku)
}
