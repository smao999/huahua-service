# Step 6：Router 骨架

> **目标**：一口气创建 10 个路由文件，全部写 `// TODO` 占位，让所有 API 端点先登记在册。
> 参考 `steps-v1/02-Router骨架.md`。

---

## 6.1 更新 Handler 聚合结构

```go
// internal/handler/handler.go
package handler

type Handlers struct {
    Health       *HealthHandler
    Auth         *AuthHandler       // TODO
    Fund         *FundHandler       // TODO
    Market       *MarketHandler     // TODO
    User         *UserHandler       // TODO
    Admin        *AdminHandler      // TODO
    Version      *VersionHandler    // TODO
    AgentRequest *AgentRequestHandler // TODO
    Jcti         *JctiHandler       // TODO
    Public       *PublicHandler     // TODO
}

func NewHandlers() *Handlers {
    return &Handlers{
        Health: NewHealthHandler(),
        // 其他 handler 暂时为 nil，后面逐步实现
    }
}
```

## 6.2 更新 Router 入口

```go
// internal/router/router.go
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, h *handler.Handlers) {
    api := r.Group("/api")
    registerHealth(api, h.Health)

    // 下面逐步打开——每个对应 Python 的一个 routers/*.py
    // registerAuth(api, h.Auth)           // routers/auth.py
    // registerFund(api, h.Fund)           // routers/fund.py
    // registerMarket(api, h.Market)       // routers/market.py
    // registerUser(api, h.User)           // routers/user.py
    // registerAdmin(api, h.Admin)         // routers/admin.py
    // registerVersion(api, h.Version)     // routers/version.py
    // registerAgentRequest(api, h.AgentRequest) // routers/agent_request.py
    // registerJcti(api, h.Jcti)           // routers/jcti.py
    // registerPublic(api, h.Public)       // routers/public.py
}
```

## 6.3 创建 9 个占位路由文件

每个文件只写函数签名，路由行全部注释掉，模仿这个模板：

```go
// internal/router/auth.go
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler) {
    g := rg.Group("/auth")
    // TODO: 逐步打开
    // g.POST("/register", h.Register)
    // g.POST("/token", h.Login)
    // g.POST("/send-code", h.SendEmailCode)
    // g.GET("/me", h.Me)
}
```

## 6.4 验证

```bash
go build ./...
# 应该能编译通过——所有 handler 为 nil 但路由注册只在运行时调用
```

---

## 本步总结

- ✅ Handler 聚合结构（10 个 handler 占位）
- ✅ Router 入口（10 个 group 占位）
- ✅ 9 个路由文件全部创建
- ✅ 编译通过
