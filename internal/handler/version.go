package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// VersionHandler 应用版本接口处理器
// 对应 Python routers/version.py
type VersionHandler struct{}

func NewVersionHandler() *VersionHandler {
	return &VersionHandler{}
}

func (h *VersionHandler) Latest(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
