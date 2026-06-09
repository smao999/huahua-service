# Step 10：Auth 模块

> **目标**：实现用户注册、登录、获取用户信息等认证功能。
> 这是第一个完整的"三层架构"模块（handler → service → repository）。
> 参考 `steps-v1/03-不依赖外部API的handler.md` 的 3.2 节。

---

## 10.1 完整的 Auth 模块包含

```
handler/auth.go   — 处理 HTTP 请求（注册、登录、获取用户信息）
service/auth/     — 业务逻辑（密码验证、JWT 生成）
repository/       — 数据库操作（查用户、建用户）← Step 8 已建
middleware/       — JWT 鉴权中间件
schema/           — 请求/响应数据结构
```

## 10.2 Schema

```go
// internal/schema/auth.go
package schema

type RegisterRequest struct {
    Username string `json:"username" binding:"required,min=4,max=20"`
    Password string `json:"password" binding:"required,min=8"`
    Email    string `json:"email" binding:"required,email"`
}

type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    UserID      int64  `json:"user_id"`
}
```

## 10.3 Auth Service

```go
// internal/service/auth/auth_service.go
package auth

import (
    "context"
    "errors"
    "fmt"
    "time"

    "huahua-service/internal/model"
    "huahua-service/internal/repository"
    "huahua-service/internal/schema"
    "huahua-service/internal/security"
    "gorm.io/gorm"
)

type AuthService struct {
    db       *gorm.DB
    userRepo repository.UserRepository
    jwtCfg   security.JWTConfig
}

func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, jwtCfg security.JWTConfig) *AuthService {
    return &AuthService{db: db, userRepo: userRepo, jwtCfg: jwtCfg}
}

func (s *AuthService) Register(ctx context.Context, req schema.RegisterRequest) (*model.User, string, error) {
    existing, _ := s.userRepo.GetByUsername(ctx, req.Username)
    if existing != nil {
        return nil, "", errors.New("用户名已被注册")
    }
    hashedPwd, _ := security.HashPassword(req.Password)
    user := &model.User{
        Username:       req.Username,
        Email:          req.Email,
        HashedPassword: hashedPwd,
        Nickname:       req.Username,
        UID:            fmt.Sprintf("%08d", time.Now().UnixNano()%100000000),
    }
    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, "", err
    }
    token, _ := security.GenerateToken(s.jwtCfg, user.Username)
    return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*model.User, string, error) {
    user, err := s.userRepo.GetByUsername(ctx, username)
    if err != nil || user == nil {
        return nil, "", errors.New("用户名或密码错误")
    }
    if !security.CheckPassword(password, user.HashedPassword) {
        return nil, "", errors.New("用户名或密码错误")
    }
    token, _ := security.GenerateToken(s.jwtCfg, user.Username)
    return user, token, nil
}
```

## 10.4 Auth Handler

```go
// internal/handler/auth.go
package handler

import (
    "net/http"
    "huahua-service/internal/schema"
    "huahua-service/internal/service/auth"
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
        c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
        return
    }
    user, token, err := h.authService.Register(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
        return
    }
    c.JSON(http.StatusOK, schema.TokenResponse{
        AccessToken: token, TokenType: "bearer", UserID: user.ID,
    })
}

func (h *AuthHandler) Login(c *gin.Context) {
    var req schema.LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
        return
    }
    user, token, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"detail": err.Error()})
        return
    }
    c.JSON(http.StatusOK, schema.TokenResponse{
        AccessToken: token, TokenType: "bearer", UserID: user.ID,
    })
}
```

## 10.5 Auth 中间件

```go
// internal/middleware/auth.go
package middleware

import (
    "net/http"
    "strings"
    "huahua-service/internal/repository"
    "huahua-service/internal/security"
    "github.com/gin-gonic/gin"
)

func AuthRequired(jwtCfg security.JWTConfig, userRepo repository.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        token := ""
        if strings.HasPrefix(authHeader, "Bearer ") {
            token = authHeader[7:]
        }
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "未登录"})
            return
        }
        claims, err := security.ParseToken(jwtCfg.Secret, token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Token 无效"})
            return
        }
        user, err := userRepo.GetByUsername(c.Request.Context(), claims.Sub)
        if err != nil || user == nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "用户不存在"})
            return
        }
        c.Set("user", user)
        c.Next()
    }
}
```

## 10.6 注册路由 + 装配

```go
// internal/router/auth.go
func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler) {
    g := rg.Group("/auth")
    g.POST("/register", h.Register)
    g.POST("/token", h.Login)
}
```

在 `handler/handler.go` 和 `router/router.go` 中打开 Auth 的注册行。

## 10.7 验证

```bash
# 注册
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"Pass1234","email":"test@test.com"}'
# 响应：{"access_token":"eyJ...","token_type":"bearer","user_id":1}

# 登录
curl -X POST http://localhost:8080/api/auth/token \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"Pass1234"}'
```

---

## 本步总结

- ✅ Schema（请求/响应结构体）
- ✅ Auth Service（注册 + 登录业务逻辑）
- ✅ Auth Handler（HTTP 请求处理）
- ✅ Auth 中间件（JWT 验证）
- ✅ 完整的"三层架构"链路
