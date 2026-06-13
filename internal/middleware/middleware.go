package middleware

import "github.com/gin-gonic/gin"

// Middleware 聚合所有中间件，供 router 层按路由注入
type Middleware struct {
	AuthRequired gin.HandlerFunc
	// AdminRequired gin.HandlerFunc  // 后续加管理员鉴权
}
