package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerFund(rg *gin.RouterGroup, h *handler.FundHandler) {
	g := rg.Group("")
	// 注意路由顺序：具体路径必须在通配符 :code 前面
	// 否则 /fund/dividends/:code 会被 /fund/:code 拦截
	g.POST("/estimate/batch", h.BatchEstimate)
	g.GET("/history/:code", h.History)
	g.GET("/fund/dividends/:code", h.Dividends) // 先匹配具体
	g.GET("/fund/fees/:code", h.Fees)           // 先匹配具体
	g.GET("/fund/today-rank", h.TodayRank)      // 先匹配具体
	g.GET("/fund/:code", h.Detail)              // 通配符放最后
	g.GET("/search", h.Search)
}
