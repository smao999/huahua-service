package handler

import (
	"errors"
	"net/http"

	"huahua-service/internal/schema"
	fundSvc "huahua-service/internal/service/fund"
	"huahua-service/pkg/validation"

	"github.com/gin-gonic/gin"
)

// FundHandler 基金相关接口处理器
type FundHandler struct {
	fundService *fundSvc.Service
}

func NewFundHandler(fundService *fundSvc.Service) *FundHandler {
	return &FundHandler{fundService: fundService}
}

func (h *FundHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, fundSvc.ErrAkshareUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"detail": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
}

func isValidFundCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// BatchEstimate POST /api/estimate/batch — 批量估值
func (h *FundHandler) BatchEstimate(c *gin.Context) {
	var req schema.BatchEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": validation.Translate(err, req)})
		return
	}

	results, err := h.fundService.GetEstimates(c.Request.Context(), req.Codes)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      results,
		"truncated": false,
		"limit":     50,
	})
}

// History GET /api/history/:code — 历史净值
func (h *FundHandler) History(c *gin.Context) {
	code := c.Param("code")
	if !isValidFundCode(code) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "基金代码格式无效"})
		return
	}

	records, err := h.fundService.GetHistory(c.Request.Context(), code)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, records)
}

// Dividends GET /api/fund/dividends/:code — 分红信息
func (h *FundHandler) Dividends(c *gin.Context) {
	code := c.Param("code")
	if !isValidFundCode(code) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "基金代码格式无效"})
		return
	}

	data, err := h.fundService.GetDividends(c.Request.Context(), code)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// Fees GET /api/fund/fees/:code — 费率信息
func (h *FundHandler) Fees(c *gin.Context) {
	code := c.Param("code")
	if !isValidFundCode(code) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "基金代码格式无效"})
		return
	}

	data, err := h.fundService.GetFees(c.Request.Context(), code)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.Data(http.StatusOK, "application/json", data)
}

// Detail GET /api/fund/:code — 基金详情
func (h *FundHandler) Detail(c *gin.Context) {
	code := c.Param("code")
	if !isValidFundCode(code) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "基金代码格式无效"})
		return
	}

	detail, err := h.fundService.GetDetail(c.Request.Context(), code)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, detail)
}

// TodayRank GET /api/fund/today-rank — 今日涨跌榜
func (h *FundHandler) TodayRank(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"date":  "",
		"gain":  []any{},
		"loss":  []any{},
		"total": 0,
	})
}

// Search GET /api/search — 基金搜索
func (h *FundHandler) Search(c *gin.Context) {
	key := c.Query("key")
	results, err := h.fundService.SearchFunds(c.Request.Context(), key)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, results)
}
