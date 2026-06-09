# Step 6：handler/fund.go / market.go 接入

> **目标**：把所有已实现的 handler 正式注册到路由中，完成"能跑通"的闭环。

---

## 6.1 检查清单

这一步不需要写新代码，而是把前几步写的**所有东西串起来**。

### 6.1.1 Fund 接口检查

对照 Python `routers/fund.py`，检查 Go 是否全部对应：

| 接口 | Python 路由 | Go handler | Go 路由 | 状态 |
|---|---|---|---|---|
| 批量估值 | `POST /api/estimate/batch` | `FundHandler.BatchEstimate` | `fund.go` | ✅ |
| 基金详情 | `GET /api/fund/{code}` | `FundHandler.Detail` | `fund.go` | ✅ |
| 历史净值 | `GET /api/history/{code}` | `FundHandler.History` | `fund.go` | ❌ |
| 分红信息 | `GET /api/fund/dividends/{code}` | `FundHandler.Dividends` | `fund.go` | ❌ |
| 费率信息 | `GET /api/fund/fees/{code}` | `FundHandler.Fees` | `fund.go` | ❌ |
| 今日涨跌榜 | `GET /api/fund/today-rank` | `FundHandler.TodayRank` | `fund.go` | ❌ |
| 基金搜索 | `GET /api/search` | `FundHandler.Search` | `fund.go` | ❌ |

### 6.1.2 Market 接口检查

| 接口 | Python 路由 | Go handler | 状态 |
|---|---|---|---|
| 市场状态 | `GET /api/market/status` | `MarketHandler.Status` | ❌ |
| 市场概览 | `GET /api/market/overview` | `MarketHandler.Overview` | ❌ |
| 指数行情 | `GET /api/market/indices` | `MarketHandler.Indices` | ❌ |
| 下一个交易日 | `GET /api/market/next-trading-day` | `MarketHandler.NextTradingDay` | ❌ |
| 夜估 | `POST /api/market/night-est` | `MarketHandler.NightEst` | ❌ |
| 日期计算 | `POST /api/market/calculate-dates` | `MarketHandler.CalculateDates` | ❌ |

---

## 6.2 逐个补齐

以"基金搜索"为例，看它是怎么被一步步串联起来的：

**Python routers/fund.py**：
```python
@router.get("/search")
async def api_search(key: str = ""):
    all_funds = await AkshareService.read_fund_info_all()
    key_upper = key.upper()
    res = []
    for code, info in all_funds.items():
        if key_upper in code or key_upper in info.get("name", "").upper():
            res.append({"code": code, "name": info.get("name", "未知"), "type": info.get("type", "基金")})
            if len(res) >= 20:
                break
    return res
```

**Go akshare client** 先加基金搜索数据方法：
<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （18 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/service/akshare/basic_info.go
package akshare

// FundBasicInfoSearch 基金基本信息（搜索用）
type FundBasicInfoSearch struct {
    Code string `json:"code"`
    Name string `json:"name"`
    Type string `json:"type"`
}

// SearchFunds 搜索基金
// 对应 Python api_search()
// 从内存缓存的基金全列表中搜索
func (c *Client) SearchFunds(key string) []FundBasicInfoSearch {
    // 从缓存取所有基金列表（由后台任务定时写入 Redis）
    // ...
    return nil
}
```
</details>


**Go handler** 加方法：
```go
// FundHandler 加
func (h *FundHandler) Search(c *gin.Context) {
    key := c.Query("key")
    if len(key) > 100 {
        c.JSON(http.StatusOK, []interface{}{})
        return
    }
    results := h.akshareClient.SearchFunds(key) // 或放到 service
    c.JSON(http.StatusOK, results)
}
```

**Go router** 打开路由：
```go
g.GET("/search", h.Search)
```

---

## 6.3 打开所有路由

```go
// internal/router/fund.go
func registerFund(rg *gin.RouterGroup, h *handler.FundHandler) {
    g := rg.Group("")
    g.POST("/estimate/batch", h.BatchEstimate)
    g.GET("/history/:code", h.History)
    g.GET("/fund/dividends/:code", h.Dividends)
    g.GET("/fund/fees/:code", h.Fees)
    g.GET("/fund/:code", h.Detail)
    g.GET("/fund/today-rank", h.TodayRank)
    g.GET("/search", h.Search)
}
```

**注意路由顺序**：Gin 的 `:code` 通配符必须在具体路由后面。
```go
// 错误：/fund/:code 会捕获 /fund/dividends/xxx
g.GET("/fund/:code", h.Detail)
g.GET("/fund/dividends/:code", h.Dividends)  // 永远不会匹配到！

// 正确：具体路径在前
g.GET("/fund/dividends/:code", h.Dividends)  // 先匹配
g.GET("/fund/:code", h.Detail)               // 后匹配通配
```

---

## 6.4 验证

```bash
# 测试每个 Fund 接口
curl "http://localhost:8080/api/search?key=上证"
curl "http://localhost:8080/api/fund/000001"
curl "http://localhost:8080/api/history/000001"
curl "http://localhost:8080/api/fund/dividends/000001"

# 测试 Market 接口
curl "http://localhost:8080/api/market/status"
curl "http://localhost:8080/api/market/indices"
```

---

## 本级小结

- ✅ Fund 路由全部打开
- ✅ Market 路由全部打开
- ✅ 确保路由顺序正确（具体路径先于通配符）
- ✅ 逐个接口验证

**下一步**：Step 7 —— AI 服务与用户数据接口。

---

## 补充：遗漏的 handler 全部补齐（从 Step 14 移入）

除了 Fund 和 Market，Python 还有以下 routers 需要对应的 Go handler：

### 6.1 AgentRequest Handler

**Python routers/agent_request.py**（AI Agent 发起的交易请求，App 端确认后执行）：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（16 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
@router.post("/agent/request")
def create_agent_request(body, current_user, db):
    # 校验 action_type ∈ {BUY, SELL, IMPORT_HOLDINGS, ...}
    # 每用户最多保留 20 条请求
    # 写入 agent_requests 表
    return {"status": "ok", "id": uuid}

@router.get("/agent/request")
def get_agent_requests(current_user, db):
    # 返回最近 7 天 PENDING 的请求
    return result

@router.put("/agent/request/{request_id}")
def update_agent_request(request_id, body, current_user, db):
    # 更新状态：PROCESSED 或 DISMISSED
    return {"status": "ok"}
```
</details>


**Go router 注册**：
```go
// internal/router/agent_request.go
func registerAgentRequest(rg *gin.RouterGroup, h *handler.AgentRequestHandler, mw *middleware.Middleware) {
    g := rg.Group("/agent")
    auth := g.Group("", mw.AuthRequired)
    auth.POST("/request", h.Create)
    auth.GET("/request", h.List)
    auth.PUT("/request/:id", h.Update)
}
```

### 6.2 Admin Handler

**Python routers/admin.py**：

```python
# 接口清单：
# POST /api/admin/activate_vip         — 激活单个 VIP
# POST /api/admin/activate_vip_all     — 批量激活 VIP（全员 + 天数）
# POST /api/admin/deduct_vip           — 扣除 VIP
# GET  /api/admin/users                — 用户列表（分页）
# GET  /api/admin/users/search         — 搜索用户
# POST /api/admin/version              — 发布新版本
# POST /api/admin/broadcast            — 发送系统通知
# GET  /api/admin/stats                — 统计数据（用户数、VIP 数）
```

**Go router**：
```go
// internal/router/admin.go
func registerAdmin(rg *gin.RouterGroup, h *handler.AdminHandler, mw *middleware.Middleware) {
    g := rg.Group("/admin")
    auth := g.Group("", mw.AuthRequired, mw.AdminRequired)
    auth.POST("/activate_vip", h.ActivateVIP)
    auth.POST("/activate_vip_all", h.ActivateVIPAll)
    auth.POST("/deduct_vip", h.DeductVIP)
    auth.GET("/users", h.ListUsers)
    auth.GET("/users/search", h.SearchUsers)
    auth.POST("/version", h.CreateVersion)
    auth.POST("/broadcast", h.Broadcast)
    auth.GET("/stats", h.Stats)
}
```

### 6.3 Version Handler

**Python routers/version.py**：
```python
@router.get("/version")
def get_version(db):
    """返回最新 App 版本信息"""
    version = db.query(AppVersion).order_by(AppVersion.id.desc()).first()
    return {"version": v.version, "changelog": v.changelog, "download_url": v.download_url, "is_mandatory": v.is_mandatory}
```

**Go router**：
```go
// internal/router/version.go
func registerVersion(rg *gin.RouterGroup, h *handler.VersionHandler) {
    rg.Group("/version").GET("", h.Latest)
}
```

### 6.4 Public/BlogStats Handler

**Python routers/public.py**：需要 API Key 鉴权，返回今日访问统计。

```go
// internal/router/public.go
func registerPublic(rg *gin.RouterGroup, h *handler.PublicHandler) {
    g := rg.Group("/public")
    g.GET("/blog-stats", h.BlogStats)
}
```

### 6.5 JCTI Handler

**Python routers/jcti.py**：投资者人格分析（需 VIP 会员）。

```go
// internal/router/jcti.go
func registerJcti(rg *gin.RouterGroup, h *handler.JctiHandler, mw *middleware.Middleware) {
    g := rg.Group("/jcti")
    g.POST("/analyze", mw.AuthRequired, h.Analyze)
}
```

### 6.6 User Handler 补充（Danmaku 弹幕）

**Python routers/user.py** 中的弹幕功能：

```python
@router.post("/danmaku/send")
async def send_danmaku(req, current_user, db):
    # 写入弹幕（Redis 或数据库）
    return {"status": "ok"}

@router.get("/danmaku/list")
async def list_danmaku(code: str):
    # 返回某只基金的弹幕列表
    return danmaku_list
```

**Go router**：
```go
// internal/router/user.go
func registerUser(rg *gin.RouterGroup, h *handler.UserHandler, mw *middleware.Middleware) {
    g := rg.Group("")
    auth := g.Group("", mw.AuthRequired)
    auth.POST("/danmaku/send", h.SendDanmaku)
    g.GET("/danmaku/list", h.ListDanmaku)       // 读弹幕不需要登录
}
```

### 所有新增 handler 注册到 router.go

```go
// internal/router/router.go
func Register(r *gin.Engine, h *handler.Handlers, mw *middleware.Middleware) {
    api := r.Group("/api")
    registerHealth(api, h.Health)
    registerAuth(api, h.Auth, mw)
    registerFund(api, h.Fund)
    registerMarket(api, h.Market, mw) // market 有的接口需要登录
    registerUser(api, h.User, mw)
    registerAdmin(api, h.Admin, mw)
    registerVersion(api, h.Version)
    registerAgentRequest(api, h.AgentRequest, mw)
    registerJcti(api, h.Jcti, mw)
    registerPublic(api, h.Public)
}
```
