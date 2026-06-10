package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MarketHandler 市场行情接口处理器
// 对应 Python routers/market.py
type MarketHandler struct{}

func NewMarketHandler() *MarketHandler {
	return &MarketHandler{}
}

func (h *MarketHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *MarketHandler) Overview(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *MarketHandler) Indices(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *MarketHandler) NextTradingDay(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *MarketHandler) NightEst(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *MarketHandler) CalculateDates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
