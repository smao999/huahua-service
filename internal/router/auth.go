package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler) {
	g := rg.Group("/auth")

	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
}
