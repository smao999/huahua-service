package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// FundHandler 基金相关接口处理器
// 对应 Python routers/fund.py
type FundHandler struct{}

func NewFundHandler() *FundHandler {
	return &FundHandler{}
}

// BatchEstimate POST /api/estimate/batch — 批量估值
func (h *FundHandler) BatchEstimate(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}

// History GET /api/history/:code — 历史净值
func (h *FundHandler) History(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}

// Dividends GET /api/fund/dividends/:code — 分红信息
func (h *FundHandler) Dividends(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}

// Fees GET /api/fund/fees/:code — 费率信息
func (h *FundHandler) Fees(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}

// Detail GET /api/fund/:code — 基金详情
func (h *FundHandler) Detail(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}

// TodayRank GET /api/fund/today-rank — 今日涨跌榜
func (h *FundHandler) TodayRank(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}

// Search GET /api/search — 基金搜索
func (h *FundHandler) Search(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
