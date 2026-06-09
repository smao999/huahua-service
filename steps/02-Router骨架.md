# Step 2：Router 骨架

> **目标**：一口气创建 10 个路由文件，每个只写 `// TODO` 占位。
> 让 README.md §5 列的所有接口在目录层面先可见，避免后续遗漏。

**这步不写业务逻辑**，只是搭架子。就像盖房子先画好每个房间的位置，后面再逐个装修。

---

## 2.1 理解 Python 的路由注册

Python 用 `APIRouter` + 装饰器注册路由：

```python
# routers/auth.py
from fastapi import APIRouter, Depends

router = APIRouter(prefix="/api/auth", tags=["auth"])

@router.post("/register")
async def register(...):
    ...

@router.post("/token")
async def login(...):
    ...

# 然后在 app_factory.py 里注册：
app.include_router(auth.router)  # 挂到 /api/auth 前缀下
```

**Gin 的区别**：Go 把"路由注册（哪个 URL 对应哪个函数）"和"处理函数"放在不同的包里。
`router/` 包只做注册，`handler/` 包写处理函数。

---

## 2.2 先写 router 入口

```go
// internal/router/router.go — 路由注册入口
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

// Register 把所有子路由装配到 gin.Engine
// 对应 Python app_factory.py 的 app.include_router(...) 调用
func Register(r *gin.Engine, h *handler.Handlers) {
    api := r.Group("/api")
    
    // 下面每行对应 Python 的一个 routers/*.py 文件
    registerAuth(api, h.Auth)       // routers/auth.py
    registerFund(api, h.Fund)       // routers/fund.py
    registerMarket(api, h.Market)   // routers/market.py
    registerUser(api, h.User)       // routers/user.py
    registerAdmin(api, h.Admin)     // routers/admin.py
    registerHealth(api, h.Health)   // routers/health.py
    registerVersion(api, h.Version) // routers/version.py
    registerAgentRequest(api, h.AgentRequest) // routers/agent_request.py
    registerJcti(api, h.Jcti)       // routers/jcti.py
    registerPublic(api, h.Public)   // routers/public.py
}
```

### Gin 的 Group 机制

Gin 的 `Group` 创建路由分组——所有在这个分组里注册的路由都会加上指定前缀：

```go
api := r.Group("/api")      // 所有路由都以 /api 开头
g := api.Group("/auth")     // 现在路由以 /api/auth 开头
g.POST("/register", handler) // 变成了 POST /api/auth/register
```

**对比 Python**：Python 的 `APIRouter(prefix="/api/auth")` 效果完全一样。

---

## 2.3 写一个完整的路由文件（health 为例）

```go
// internal/router/health.go — 健康检查路由
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

// registerHealth 注册健康检查路由
// 对应 Python routers/health.py
//
// Python 源码：
// router = APIRouter(prefix="/api", tags=["health"])
// @router.get("/health")       →  GET /api/health
// @router.get("/health/redis") →  GET /api/health/redis
func registerHealth(rg *gin.RouterGroup, h *handler.HealthHandler) {
    g := rg.Group("/health")
    g.GET("", h.Ping)         // GET /api/health
    g.GET("/redis", h.Redis)  // GET /api/health/redis
}
```

## 2.4 其他 9 个路由文件全部占位

每个文件只写函数签名 + 注释说明 Python 对应什么路由，然后全部 `// TODO`：

```go
// internal/router/auth.go — 认证路由
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

// registerAuth 注册认证相关路由
// 对应 Python routers/auth.py
//
// Python 路径前缀：/api/auth
// 包含接口：
//   POST /send-code     — 发送邮箱验证码
//   POST /register      — 注册
//   POST /token         — 登录（获取 JWT）
//   POST /login-by-email — 邮箱登录
//   POST /logout        — 登出
//   GET  /me            — 获取当前用户信息
//   PUT  /profile       — 更新个人资料
//   POST /upload_avatar — 上传头像
//   POST /change-password — 修改密码
//   POST /bind-email    — 绑定邮箱
//   POST /forgot-password — 忘记密码
//   POST /claim_admin   — 领取管理权限
//   POST /agent-token   — 创建 Agent Token
//   GET  /agent-tokens  — 列出 Agent Token
//   DELETE /agent-token/:id — 删除 Agent Token
func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler) {
    g := rg.Group("/auth")
    
    // ===== 无需认证 =====
    // g.POST("/send-code", h.SendEmailCode)       // Python: POST /api/auth/send-code
    // g.POST("/register", h.Register)              // Python: POST /api/auth/register
    // g.POST("/token", h.Login)                    // Python: POST /api/auth/token
    // g.POST("/login-by-email", h.LoginByEmail)    // Python: POST /api/auth/login-by-email
    
    // ===== 需要认证（后面加中间件）=====
    // auth := g.Group("", mw.AuthRequired())
    // auth.GET("/me", h.Me)
    // auth.POST("/logout", h.Logout)
    // auth.PUT("/profile", h.UpdateProfile)
    // auth.POST("/change-password", h.ChangePassword)
}
```

其他路由文件同理——这里给出每个文件的 Python 对照，方便你写注释：

```go
// internal/router/fund.go
// Python routers/fund.py
// POST /api/estimate/batch  — 批量获取实时估值
// GET  /api/history/{code}  — 基金历史净值
// GET  /api/fund/dividends/{code}  — 分红信息
// GET  /api/fund/fees/{code}       — 费率信息
// GET  /api/fund/{code}            — 基金详情
// GET  /api/fund/today-rank        — 今日涨跌榜
// GET  /api/fund/today-timeline/{code} — 今日分时估值
// GET  /api/search                 — 基金搜索
```

```go
// internal/router/market.go
// Python routers/market.py
// GET /api/market/status        — 市场状态（开/闭盘）
// GET /api/market/overview      — 市场概览
// GET /api/market/indices       — 指数行情
// POST /api/market/calculate-dates — 日期计算
// POST /api/market/night-est    — 夜估
// GET /api/market/next-trading-day  — 下一个交易日
```

```go
// internal/router/user.go
// Python routers/user.py
// POST /api/sync/upload   — 上传用户数据
// POST /api/sync/download — 下载用户数据
// GET  /api/sync/meta     — 同步元数据
// POST /api/danmaku/send  — 发弹幕
// GET  /api/danmaku/list  — 弹幕列表
// POST /api/import_screenshot — 截图导入
// POST /api/import_transactions — 交易记录导入
```

```go
// internal/router/admin.go
// Python routers/admin.py
// GET  /api/admin/users           — 用户列表
// GET  /api/admin/users/search   — 搜索用户
// POST /api/admin/vip/activate   — 激活 VIP
// POST /api/admin/vip/batch      — 批量激活 VIP
// POST /api/admin/vip/deduct     — 扣除 VIP
// POST /api/admin/broadcast      — 广播通知
// GET  /api/admin/stats          — 统计数据
// POST /api/admin/version        — 创建版本
// POST /api/admin/import/codes   — 导入基金代码
// GET  /api/admin/logs           — 日志
```

```go
// internal/router/version.go
// Python routers/version.py
// GET /api/version — 获取最新版本信息
```

```go
// internal/router/agent_request.go
// Python routers/agent_request.py
// POST /api/agent/request     — 创建 Agent 请求
// GET  /api/agent/request     — 获取 Agent 请求列表
// PUT  /api/agent/request/:id — 更新 Agent 请求状态
```

```go
// internal/router/jcti.go
// Python routers/jcti.py
// POST /api/jcti/evaluate     — JCTI 评测
// GET  /api/jcti/result       — 获取评测结果
```

```go
// internal/router/public.go
// Python routers/public.py
// GET /api/public/blog-stats        — 博客统计数据
// POST /api/public/blog-stats/record — 记录博客访问
```

---

## 2.5 在 app.go 中启用路由注册

```go
// internal/app/app.go
package app

import (
    "huahua-service/internal/handler"
    "huahua-service/internal/router"
    "github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    
    h := handler.NewHandlers()
    router.Register(r, h)  // ← 注册所有路由
    
    return r
}
```

---

## 2.6 验证：编译检查

```bash
go build ./...
# 目前应该报错 "handler.NewHandlers 返回的 Auth、Fund 等是 nil"
# 因为 handler 还没实现——这是正常的！
# 后面 Step 3 开始逐个实现
```

---

## 本级小结

- ✅ 创建了 `router.go` 入口文件，调用 10 个子路由
- ✅ 创建了 `health.go` 完整路由注册（这个马上就能用）
- ✅ 创建了 9 个占位路由文件（auth, fund, market, user, admin, version, agent_request, jcti, public）
- ✅ 每个占位文件里有完整的 Python 对照注释

**下一步**：Step 3 —— 从 health handler 开始，逐个实现不依赖外部 API 的模块。
