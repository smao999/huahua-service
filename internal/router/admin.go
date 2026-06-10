package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerAdmin(rg *gin.RouterGroup, h *handler.AdminHandler) {
	g := rg.Group("/admin")
	g.POST("/activate_vip", h.ActivateVIP)
	g.POST("/activate_vip_all", h.ActivateVIPAll)
	g.POST("/deduct_vip", h.DeductVIP)
	g.GET("/users", h.ListUsers)
	g.GET("/users/search", h.SearchUsers)
	g.POST("/version", h.CreateVersion)
	g.POST("/broadcast", h.Broadcast)
	g.GET("/stats", h.Stats)
}
