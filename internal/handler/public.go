package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// PublicHandler 公开接口（博客统计等）
// 对应 Python routers/public.py
type PublicHandler struct{}

func NewPublicHandler() *PublicHandler {
	return &PublicHandler{}
}

func (h *PublicHandler) BlogStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
