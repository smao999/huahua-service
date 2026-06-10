package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AgentRequestHandler AI Agent 交易请求接口处理器
// 对应 Python routers/agent_request.py
type AgentRequestHandler struct{}

func NewAgentRequestHandler() *AgentRequestHandler {
	return &AgentRequestHandler{}
}

func (h *AgentRequestHandler) Create(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *AgentRequestHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
func (h *AgentRequestHandler) Update(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "not implemented yet"})
}
