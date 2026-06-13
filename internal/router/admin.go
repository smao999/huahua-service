package router

import (
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func registerAdmin(rg *gin.RouterGroup, h *handler.AdminHandler, mw *middleware.Middleware) {
	g := rg.Group("/admin", mw.AuthRequired)
	g.POST("/activate_vip", h.ActivateVIP)
	g.POST("/activate_vip_all", h.ActivateVIPAll)
	g.POST("/deduct_vip", h.DeductVIP)
	g.GET("/users", h.ListUsers)
	g.GET("/users/search", h.SearchUsers)
	g.POST("/version", h.CreateVersion)
	g.POST("/broadcast", h.Broadcast)
	g.GET("/stats", h.Stats)
}
