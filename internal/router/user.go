package router

import (
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func registerUser(rg *gin.RouterGroup, h *handler.UserHandler, mw *middleware.Middleware) {
	g := rg.Group("")
	g.POST("/danmaku/send", mw.AuthRequired, h.SendDanmaku)
	g.GET("/danmaku/list", h.ListDanmaku)
}
