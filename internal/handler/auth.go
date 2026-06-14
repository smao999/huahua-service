package handler

import (
	"huahua-service/internal/schema"
	"huahua-service/internal/service/auth"
	"huahua-service/pkg/validation"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req schema.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": validation.Translate(err, req)})
		return
	}

	user, token, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schema.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		UId:         user.UID,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req schema.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"errors": validation.Translate(err, req)})
		return
	}

	user, token, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schema.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		UId:         user.UID,
	})
}
