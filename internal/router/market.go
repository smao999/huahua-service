package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerMarket(rg *gin.RouterGroup, h *handler.MarketHandler) {
	g := rg.Group("/market")
	g.GET("/status", h.Status)
	g.GET("/overview", h.Overview)
	g.GET("/indices", h.Indices)
	g.GET("/next-trading-day", h.NextTradingDay)
	g.POST("/night-est", h.NightEst)
	g.POST("/calculate-dates", h.CalculateDates)
}
