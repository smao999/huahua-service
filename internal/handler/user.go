package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户相关接口处理器
// 对应 Python routers/user.py
type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) SendDanmaku(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *UserHandler) ListDanmaku(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
