package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// JctiHandler 投资者人格分析接口处理器
// 对应 Python routers/jcti.py
type JctiHandler struct{}

func NewJctiHandler() *JctiHandler {
	return &JctiHandler{}
}

func (h *JctiHandler) Analyze(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
