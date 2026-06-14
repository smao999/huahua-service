package middleware

import (
	"huahua-service/internal/security"
	"huahua-service/internal/service/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthRequired(jwtCfg security.JWTConfig, authSvc *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		token := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = authHeader[7:]
		}

		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := security.ParseToken(jwtCfg.Secret, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized，token 无效"})
			return
		}
		user, err := authSvc.GetUserByUsername(c.Request.Context(), claims.Sub)
		if err != nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not fund"})
			return
		}
		c.Set("user", user)
		c.Next()
	}
}
