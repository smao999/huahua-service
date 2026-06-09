# 重构huahua

---

## 0. 总览：技术栈映射表（Python → Go）

| Python 依赖 / 概念 | Go 推荐替代 | 备注 |
|---|---|---|
| FastAPI | `github.com/gin-gonic/gin` | HTTP 框架；Gin 没有"依赖注入"，用中间件 + Context 传值 |
| Pydantic (Schemas) | `binding:"required,..."` + 自写 validator | 用 Gin `ShouldBindJSON` + `validator/v10` |
| SQLAlchemy | `gorm.io/gorm` (推荐) 或 `database/sql` + `sqlx` | 业务用 GORM 更接近 ORM 体验 |
| PostgreSQL Driver `psycopg2` | `gorm.io/driver/postgres` (内部 pgx) | |
| Redis `redis.asyncio` | `github.com/redis/go-redis/v9` | API 几乎一一对应 |
| `httpx.AsyncClient` | `net/http` 或 `github.com/go-resty/resty/v2` | resty 更易用 |
| `asyncio.Semaphore` | `chan struct{}` (有缓冲) 或 `golang.org/x/sync/semaphore` | |
| `asyncio.Lock` / `_inflight_xxx` | `sync.Mutex` + `sync.Map`，或 `golang.org/x/sync/singleflight` | singleflight 直接解决"同一 key 同时只跑一次" |
| `asyncio.gather` | `golang.org/x/sync/errgroup` | |
| `passlib bcrypt` | `golang.org/x/crypto/bcrypt` | |
| `PyJWT` | `github.com/golang-jwt/jwt/v5` | |
| `fastapi-mail` | `gopkg.in/gomail.v2` 或 `github.com/jordan-wright/email` | |
| `Pillow` (头像压缩 / 截图归一化) | `github.com/disintegration/imaging` | |
| `pandas` (akshare 内部用) | 大多不需要 — 我们走 HTTP API；本地数据用切片/`map` | |
| `akshare` Python 库 | **没有等价物**，直接走它背后的 HTTP（东财、腾讯、Sina、Twelve Data）| 见 §6 |
| `google-genai` / OpenRouter | 直接 `net/http` POST | 两家都是 REST JSON |
| `pydantic-settings` | `github.com/spf13/viper` 或 `github.com/caarlos0/env` | |
| `python-dotenv` | `github.com/joho/godotenv` | |
| `logging` | `log/slog` (Go 1.21+) | |
| `ContextVar` / `request.state` | `*gin.Context` 的 `c.Set` / `c.Get` | |
| `run_in_threadpool` | Go 协程默认就是异步，**无需特殊处理** | 但要把"长任务"用 `errgroup` 或后台 `go func()` 隔离 |
| BackgroundTask (lifespan) | 启动后 `go backgroundLoop()` + `context.Context` 控制退出 | 见 §11 |
| `decimal.Decimal` | `github.com/shopspring/decimal` | 价格、α/β 计算必须用 `decimal`，禁止 `float64` |

---

## 1. 建议的 Go/Gin 目录结构

```
backend-go/
├── cmd/
│   └── server/
│       └── main.go                    # 启动入口（对应 main.py）
├── internal/
│   ├── app/
│   │   ├── app.go                     # gin.Engine 装配（对应 core/app_factory.py）
│   │   └── lifespan.go                # 启动/停止钩子（对应 core/lifespan.py）
│   ├── config/
│   │   └── config.go                  # viper 读 .env（对应 config.py）
│   ├── middleware/
│   │   ├── ratelimit.go               # Redis 滑窗限流（对应 RedisRateLimitMiddleware）
│   │   ├── blogstats.go               # 博客访问统计中间件
│   │   ├── auth.go                    # JWT + AgentToken 中间件（对应 get_current_user）
│   │   ├── admin.go                   # 管理员检查
│   │   ├── cors.go                    # CORS 配置
│   │   ├── gzip.go                    # gin-contrib/gzip
│   │   └── recovery.go                # 全局 500 兜底（对应 _global_exception_handler）
│   ├── router/                        # 路由集中注册（按 prefix 分文件）
│   │   ├── router.go                  # 入口：装配所有子路由
│   │   ├── auth.go                    # /api/auth/*
│   │   ├── fund.go                    # /api/* 基金核心
│   │   ├── market.go                  # /api/market/*
│   │   ├── user.go                    # /api/sync/*、/api/danmaku/*、/api/import_*
│   │   ├── admin.go                   # /api/admin/*
│   │   ├── version.go                 # /api/version
│   │   ├── agent_request.go           # /api/agent/*
│   │   ├── jcti.go                    # /api/jcti/*
│   │   ├── public.go                  # /api/public/*
│   │   └── health.go                  # /api/health*
│   ├── handler/                       # HTTP 处理器（对应 routers/*.py）
│   │   ├── auth.go                    # 来自 routers/auth.py
│   │   ├── fund.go                    # 来自 routers/fund.py
│   │   ├── market.go                  # 来自 routers/market.py
│   │   ├── user.go                    # 来自 routers/user.py
│   │   ├── admin.go                   # 来自 routers/admin.py
│   │   ├── version.go                 # 来自 routers/version.py
│   │   ├── agent_request.go           # 来自 routers/agent_request.py
│   │   ├── jcti.go                    # 来自 routers/jcti.py
│   │   ├── public.go                  # 来自 routers/public.py
│   │   └── health.go                  # 来自 routers/health.py
│   ├── service/                       # 业务逻辑（对应 services/*.py）
│   │   ├── fund/
│   │   │   ├── fund_service.go
│   │   │   ├── holdings_estimate.go
│   │   │   └── win_rate.go
│   │   ├── akshare/                   # 上游数据源封装（拆分以便维护）
│   │   │   ├── client.go              # httpx 客户端 + cookie/UA
│   │   │   ├── trading_calendar.go    # 交易日 / 时区工具
│   │   │   ├── history.go             # 历史净值
│   │   │   ├── estimate.go            # 实时估值 (fundgz)
│   │   │   ├── basic_info.go          # 基础信息 / 持仓 / 行业
│   │   │   ├── dividends.go
│   │   │   ├── kline.go               # A 股 / 港股 / 美股 K 线
│   │   │   ├── market_index.go        # 指数行情
│   │   │   ├── rankings.go            # 板块/基金涨幅榜
│   │   │   ├── forex.go               # USDCNH / HKDCNH
│   │   │   ├── twelve_data.go         # Twelve Data 客户端 + 令牌桶
│   │   │   └── cooldown.go            # akshare 网络故障熔断
│   │   ├── ai/
│   │   │   ├── ai_service.go          # 顶层调度（OpenRouter → Gemini 回退）
│   │   │   ├── openrouter.go
│   │   │   ├── gemini.go
│   │   │   ├── prompts.go             # 截图识别 / JCTI / 板块分类 prompt
│   │   │   ├── sector_rules.go        # 规则化板块/指数表（_BROAD_INDEX_PATTERNS 等）
│   │   │   └── image.go               # 截图归一化 (imaging 替代 PIL)
│   │   ├── night/
│   │   │   └── night_estimate.go      # T+2+ 夜估
│   │   ├── calibration/
│   │   │   └── calibration.go         # α/β EMA 校准
│   │   ├── market_index/
│   │   │   └── market_index.go        # 大盘指数预热与读取
│   │   └── mail/
│   │       └── mail.go                # SMTP 验证码
│   ├── task/
│   │   └── background.go              # 启动后台循环（对应 tasks/background_refresh.py）
│   ├── model/                         # GORM 数据模型（对应 models.py）
│   │   ├── user.go
│   │   ├── user_data.go
│   │   ├── fund_nav.go
│   │   ├── fund_basic_info.go
│   │   ├── fund_sector.go
│   │   ├── app_version.go
│   │   ├── agent_token.go
│   │   ├── agent_request.go
│   │   ├── fund_calibration.go
│   │   ├── system_notice.go
│   │   └── system_kv.go
│   ├── repository/                    # 数据访问层（独立于 service，便于单元测试）
│   │   ├── repository.go              # 通用接口与 Tx 助手
│   │   ├── user_repo.go               # users 表所有 CRUD
│   │   ├── user_data_repo.go          # user_data 同步表
│   │   ├── fund_nav_repo.go           # fund_navs 历史净值
│   │   ├── fund_basic_info_repo.go    # fund_basic_info / fund_sectors
│   │   ├── agent_token_repo.go        # agent_tokens
│   │   ├── agent_request_repo.go      # agent_requests
│   │   ├── calibration_repo.go        # fund_calibrations
│   │   ├── app_version_repo.go
│   │   ├── system_notice_repo.go
│   │   └── system_kv_repo.go
│   ├── schema/                        # 请求/响应 DTO（对应 schemas.py）
│   │   ├── auth.go
│   │   ├── fund.go
│   │   ├── admin.go
│   │   └── ...
│   ├── db/
│   │   └── db.go                      # GORM 连接 + 迁移（对应 database.py）
│   └── security/                      # 业务相关的安全工具（含业务命名 / 业务字段）
│       ├── ratelimit.go               # 业务级限流 helper（rl:auth:* / rl:api:* 键名）
│       ├── etag.go                    # build_sync_etag（剔除 funds[*] 瞬时字段）
│       └── validators.go              # FundCode / BenchmarkCode 校验
├── pkg/                               # 业务无关、可被其他服务复用的通用包
│   ├── util/                          # 时间 / 数值 / 通用工具（对应 utils.py）
│   │   ├── time.go                    # utcnow / 北京时间 / get_seconds_until_next_open
│   │   └── decimal.go                 # round_half_up（基于 shopspring/decimal）
│   ├── cache/                         # Redis 封装 + 进程内 LRU 降级（对应 cache.py）
│   │   ├── cache.go                   # Get/Set/Incr/SAdd/SCard/Delete/PushList/GetList...
│   │   ├── lock.go                    # 分布式锁 acquire_lock / release_lock（Lua 防误删）
│   │   └── memfallback.go             # LRU 降级（_MEM_MAX_KEYS=2000，主动 TTL 清理）
│   ├── httpx/                         # 通用 HTTP 增强（对应 akshare 那套 client patch）
│   │   ├── client.go                  # http.Client 默认超时 / UA / Retry
│   │   ├── retry.go                   # _run_with_retry 等价（指数退避 + IncompleteRead 处理）
│   │   └── circuit.go                 # 5 次失败 120s 熔断（_record_akshare_network_fail）
│   ├── leader/                        # Redis 分布式领导锁（对应 lock:bg_refresh_loop:v1）
│   │   └── leader.go                  # Acquire/Heartbeat/Release，TTL 180s / 心跳 60s
│   ├── ratelimit/                     # 算法层：Redis 滑窗 + 本地降级（不绑定 Gin / 业务键名）
│   │   ├── sliding_window.go          # ZREMRANGEBYSCORE + ZCARD + ZADD pipeline
│   │   └── local_lru.go               # OrderedDict + sync.Mutex 等价
│   ├── ipx/                           # 客户端 IP 提取（对应 get_client_ip）
│   │   └── ip.go                      # 信任 N 个反代，按 X-Forwarded-For 反向取候选
│   ├── jsonx/                         # LLM 响应健壮解析（对应 AIService 那一坨 _parse_*）
│   │   ├── partial.go                 # _extract_partial_objects 大括号匹配
│   │   ├── normalize.go               # _normalize_json_like_text（单引号 / 尾逗号 / BOM）
│   │   └── parse.go                   # _parse_json_array 顶层入口
│   ├── imagex/                        # 图片归一化（对应 _normalize_uploaded_image_sync）
│   │   └── resize.go                  # imaging.Resize + JPEG 质量重编码
│   ├── llm/                           # 通用 LLM 客户端（无 prompt / 无业务规则）
│   │   ├── openrouter.go              # /chat/completions
│   │   ├── gemini.go                  # /v1beta/models/:model:generateContent
│   │   └── fallback.go                # provider 顺序回退框架
│   └── tokenbucket/                   # 通用令牌桶限速器（对应 _TwelveDataRateLimiter）
│       └── bucket.go                  # 按"每分钟 N 个 credits"的版本
├── migrations/                        # 手写 SQL（对应 lifespan.py 里的 _migrate_*）
├── scripts/                           # 一次性脚本（对应 scripts/*.py）
└── go.mod
```

> **依赖方向：`internal/` 可以 import `pkg/`；`pkg/` 绝不能 import `internal/`**。这条规则一旦破坏，pkg 就失去了通用性，等于在玩文字游戏。每个 pkg 包的 godoc 都该独立、能被任何项目 copy 出去就用。

---

## 2. 启动入口 (cmd/server/main.go)

**Python 对应**：`backend/main.py` (1-49)

需要做的事：
1. 加载 `.env` (godotenv)。
2. 初始化 slog logger，可选写入轮转文件（对应 `LOG_FILE`/`LOG_MAX_BYTES`，行 16-26）。Go 用 `gopkg.in/natefinch/lumberjack.v2` 做轮转。
3. 调用 `app.New()` 拿到 `*gin.Engine`，注入到 `http.Server`。
4. 监听 `PORT` (默认 8080)。
5. 启动 `task.Run(ctx)` 后台循环（go func + cancel context）。
6. 信号优雅退出：捕获 `SIGINT/SIGTERM`，先 `srv.Shutdown(ctx)`，再 cancel 后台 ctx，再关 Redis / DB / HTTP client。

**Go 关键点**：
- 使用 `http.Server.Shutdown` 给"已建立的连接"留 10s 退出（对应 Python 里 `asyncio.wait_for(..., timeout=10)`）。
- 启动后台任务时**必须传 `context.Context`**，所有内层 goroutine 都监听它，否则进程关不掉。

---

## 3. 应用装配 (internal/app/app.go)

**Python 对应**：`backend/core/app_factory.py` (1-379)

需要做的事：
1. **生产环境守卫**（行 32-48）：检查 `SECRET_KEY` 不为默认值、`CORS_ORIGINS` 不为 `*`。Go 直接 `panic` 或 `log.Fatal`。
2. **中间件顺序**（行 282-314，顺序很重要）：
    - ProxyHeaders（信任反向代理）→ Go 用 `gin.Engine.SetTrustedProxies`
    - RateLimit（Redis 滑窗）
    - BlogVisitStats（记录访问数据）
    - CORS（gin-contrib/cors）
    - GZip（gin-contrib/gzip）
3. **静态文件**（行 316-318）：
    - `/static/uploads/*` → `engine.Static("/static/uploads", uploadsDir)`
    - `/static/buy/*` → `engine.Static("/static/buy", buyDir)`
4. **路由注册**（行 321-331）：调用 `router.Register(engine, deps)`，由 `internal/router/router.go` 集中分发到各子路由文件。详细分层说明见 §4a。
5. **全局异常处理**（行 333-337）：Gin 用 `engine.Use(middleware.Recovery())`，记录堆栈、对外只返 `{"detail": "Internal server error"}`，避免泄露内部信息。
6. **SPA 兜底**（行 339-377）：
    - `GET /` → 若 `dist/index.html` 存在则返回；否则返回健康信息。
    - `NoRoute` 处理器实现 `_safe_resolve` 防目录穿越：先 `filepath.Clean`，再校验 `strings.HasPrefix(absPath, baseDir)`。
    - **`/api/` 前缀的未知路径必须返回 JSON 404**（防 SPA 误捕获 API 路径，对应行 360-364）。

---

## 4a. 分层架构与依赖方向（重要）

整个 Go 项目采用**严格的单向依赖**，**禁止反向引用**：

```
router → handler → service → repository → gorm.DB
            ↘         ↘            ↘
           schema   pkg/cache    pkg/httpx / pkg/llm / pkg/jsonx / ...

所有层 → pkg/*       （任何 internal 层都可以 import pkg）
pkg/*  ↛ internal/*  （pkg 包绝对禁止反向引用 internal，否则就不是"通用包"了）
```

- **router**：只做 `engine.Group(...).METHOD(path, handler)` 的注册映射。**不写业务逻辑**。
- **handler**：参数绑定（`ShouldBindJSON`）、权限校验、调 service、组装响应。**不直接读写 DB**。
- **service**：业务规则（α/β 校准、win-rate、估值优先级链）、缓存策略、跨数据源编排。**不直接写 SQL，统一通过 repository**。
- **repository**：所有 `gorm.DB` 调用集中在此层。**返回 `*model.X` 或 `error`**，绝不返回 `*gorm.DB`。
- **pkg/***：业务无关的通用包（util / cache / httpx / leader / ratelimit / ipx / jsonx / imagex / llm / tokenbucket）。每个包应当**可以被复制到任何项目里就用**，所以禁止 import `internal/model`、禁止读 `internal/config`，需要配置就通过函数参数或 `Options` 结构注入。

### router 包写法范式

```
// internal/router/router.go
func Register(engine *gin.Engine, h *Handlers, mw *Middleware) {
    api := engine.Group("/api")
    registerAuth(api, h.Auth, mw)
    registerFund(api, h.Fund, mw)
    registerMarket(api, h.Market, mw)
    // ... 其他子路由
}

// internal/router/auth.go
func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler, mw *Middleware) {
    g := rg.Group("/auth")
    g.POST("/send-code",        h.SendEmailCode)
    g.POST("/register",         h.Register)
    g.POST("/token",            h.Login)
    g.POST("/login-by-email",   h.LoginByEmail)
    g.POST("/logout",           h.Logout)
    // 需鉴权的路径单独建 subgroup
    auth := g.Group("", mw.AuthRequired())
    auth.GET("/me",                  h.Me)
    auth.GET("/invite-info",         h.GetInviteInfo)
    auth.POST("/change-password",    h.ChangePassword)
    auth.POST("/bind-email",         h.BindEmail)
    auth.PUT("/profile",             h.UpdateProfile)
    auth.POST("/upload_avatar",      h.UploadAvatar)
    auth.POST("/claim_admin",        h.ClaimAdmin)
    auth.POST("/agent-token",        h.CreateAgentToken)
    auth.GET("/agent-tokens",        h.ListAgentTokens)
    auth.DELETE("/agent-token/:id",  h.DeleteAgentToken)
}
```

**好处**：
- 新人查"所有 endpoint 在哪"只需翻 `internal/router/` 一个目录。
- 中间件挂载（哪些路径要 `AuthRequired`、哪些要 `AdminRequired`）一目了然，不会跟 Python `Depends` 那样散在 handler 签名里。
- 想加 `/api/v2/*` 路径前缀时改一处即可。

### repository 包写法范式

每个 repository 是一个**接口 + 实现**，便于在 service 测试时用 mock 替换：

```
// internal/repository/user_repo.go
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*model.User, error)
    GetByUsername(ctx context.Context, username string) (*model.User, error)
    GetByUID(ctx context.Context, uid string) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    Create(ctx context.Context, u *model.User) error
    Update(ctx context.Context, u *model.User) error
    CountInvitedBy(ctx context.Context, inviterID int64) (int64, error)
    LockByIDForUpdate(ctx context.Context, tx *gorm.DB, id int64) (*model.User, error)
    // 管理员侧
    Search(ctx context.Context, q string, page, pageSize int) ([]model.User, int64, error)
    CountActiveVIP(ctx context.Context, now time.Time) (int64, error)
    CountActivePro(ctx context.Context, now time.Time) (int64, error)
    PaginateAll(ctx context.Context, offset, limit int) ([]model.User, error)
}

type userRepo struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepository { return &userRepo{db: db} }

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
    var u model.User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil  // 业务上"不存在"用 nil + nil，不当错误抛
    }
    return &u, err
}
```

**事务规则**（这是 Repository 层最容易出错的地方）：
- 简单单表操作：service 直接调 `repo.Xxx(ctx, ...)`。
- 跨表事务（如注册时同时改 `users` + 给邀请人发奖励，对应 `routers/auth.py:347-407`）：
  ```
  // service 层
  err := s.db.Transaction(func(tx *gorm.DB) error {
      newUser, err := s.userRepo.CreateTx(ctx, tx, &model.User{...})
      if err != nil { return err }
      inviter, err := s.userRepo.LockByIDForUpdateTx(ctx, tx, inviterID)  // SELECT FOR UPDATE
      if err != nil { return err }
      // 在锁保护下重新统计 reward count
      count, err := s.userRepo.CountInvitedByTx(ctx, tx, inviter.ID)
      if err != nil { return err }
      if count < MaxInviteRewardTimes {
          inviter.VIPExpiresAt = extendVIP(inviter.VIPExpiresAt, InviteRewardDays)
          if err := s.userRepo.UpdateTx(ctx, tx, inviter); err != nil { return err }
      }
      return nil
  })
  ```
- 每个 repository 方法都应该提供 `XxxTx(ctx, tx, ...)` 版本接收外部 tx，或者用变体接口（`type UserTxRepository`）。推荐**所有方法都通过一个 internal helper 统一 db/tx 选择**：
  ```
  func (r *userRepo) txOrDB(tx *gorm.DB) *gorm.DB {
      if tx != nil { return tx }
      return r.db
  }
  ```

### handler / service / repository 三层职责对照表

以"注册"为例：

| 层 | 文件 | 职责 | 不该做 |
|---|---|---|---|
| router | `router/auth.go` | `g.POST("/register", h.Register)` 一行 | 任何业务 |
| handler | `handler/auth.go: Register` | 1) `ShouldBindJSON(&req)`；2) IP 限流；3) `MailService.Verify`；4) 调 `AuthService.Register(ctx, req)`；5) 转 200/4xx 响应 | 直接查 DB |
| service | `service/auth/auth_service.go: Register` | 邀请码逻辑、邀请人锁、reward 计算、bcrypt 哈希、事务编排 | 写 SQL、绑参数 |
| repository | `repository/user_repo.go` | 单表 CRUD + `SELECT FOR UPDATE` | 调 cache、限流、bcrypt |

### 与 Python 行号的对应方式

虽然加了 repository 层，**搬代码的对应关系仍然清晰**：
- Python `db.query(User).filter(User.username == username).first()` → Go `userRepo.GetByUsername(ctx, username)`
- Python `db.add(new_user); db.commit()` → Go `userRepo.Create(ctx, newUser)`
- Python `.with_for_update()` → Go `userRepo.LockByIDForUpdateTx(ctx, tx, id)`

第一次搬时直接把 Python 的 DB 调用**整段转成 repository 方法**即可；service 层基本是把 Python 函数里"非 DB 部分"原样保留。

---

## 4. 中间件实现要点

### 4.1 限流 RedisRateLimitMiddleware
**Python 对应**：`core/app_factory.py:54-150`
- 算法：Redis Sorted Set 滑窗（`ZREMRANGEBYSCORE` + `ZCARD` + `ZADD` + `EXPIRE`，行 119-124）。
- 仅限 `/api/` 路径，跳过 `OPTIONS`（行 107-110）。
- Redis 不可用时降级到进程内 `OrderedDict + sync.Mutex` 滑窗（行 78-104）。
- LRU 淘汰：本地 windows 超过 10000 个 IP 时踢出最旧/过期条目（行 91-103）。
- 客户端断连保护：`No response returned` 错误返 503，不重抛（行 138-150）。
- Go 实现：
    - 走 `go-redis` Pipeline；本地 fallback 用 `container/list` + `sync.Mutex`。
    - 用 `gin.Context.AbortWithStatusJSON(429, ...)` 返回。

### 4.2 BlogVisitStatsMiddleware
**Python 对应**：`core/app_factory.py:153-229`
- 记录每日 `visit_count` / 来访唯一访客哈希集合 / 已登录用户哈希集合（行 214-227）。
- 只对部分 `/api/` 路径生效；跳过 `blog-stats` 自身、跳过热路径（行 188-202）。
- 解析 JWT：先看 `Authorization: Bearer`，跳过 `AgentToken`，回落到 `access_token` cookie（行 169-186）。
- Go 实现：写在 `defer c.Next() 之后` 比较直观；JWT 解析用 `golang-jwt/jwt/v5`。

### 4.3 Auth 中间件（关键）
**Python 对应**：`routers/auth.py:200-271` (`get_current_user`)
- 双协议：
    1. `Authorization: AgentToken <raw>` → SHA256 哈希后查 `agent_tokens`，应用 scope 白名单（行 161-198 `_is_agent_token_allowed`）。每次访问更新 `last_used_at/last_used_ip`，并对 AgentToken 路径再做限流（行 211）。
    2. `Authorization: Bearer <jwt>` 或 cookie `access_token` → 解 JWT → 查 `users` 表。
- Go 实现：
    - 一个 `func AuthRequired() gin.HandlerFunc`，最后把 `*model.User` 用 `c.Set("user", ...)` 存进 Context。处理器用 `c.MustGet("user").(*model.User)` 取出。
    - Scope 白名单：在中间件里完成"路径+方法→所需 scope"判定（用 `map` + 前缀 trie，或用注册 handler 时绑定的 metadata）。
    - **重要**：JWT 解析、bcrypt 校验都是 CPU 操作，Go 协程里直接同步跑就行；不要再额外开线程池（Python 是因为 GIL 才需要 `run_in_threadpool`）。

### 4.4 Admin 中间件
**Python 对应**：`routers/auth.py:273-276`
- 简单：从 Context 拿 `*User`，校验 `user.UID == settings.ADMIN_UID`，否则 403。

---

## 5. HTTP 路由总表（功能点 → Python 来源 → Go 落位）

> 这是**主清单**，不要漏掉任何一行。每个 endpoint 对应三处 Go 文件：
> - **router**：`internal/router/<file>.go` 中的一行 `g.METHOD(path, h.Xxx)` 注册
> - **handler**：`internal/handler/<file>.go` 中的处理函数（参数绑定、调 service、组装响应）
> - **service / repository**：`internal/service/...` 业务逻辑 + `internal/repository/...` 数据访问

### 5.1 认证 `/api/auth/*` → `handler/auth.go`
**Python 对应**：`routers/auth.py`（817 行）

| 方法 | 路径 | Python 函数 (行) | Go handler | 说明 / 关键点 |
|---|---|---|---|---|
| POST | `/api/auth/send-code` | `send_email_code` (278-329) | `SendEmailCode` | 限流 10/60s 与 3/300s；按 type 分 4 种用途（REGISTER/FORGOT_PASSWORD/BIND/AGENT_TOKEN）；冷却 60s |
| POST | `/api/auth/register` | `register` (331-409) | `Register` | 限流（IP 24h≤3）、临时邮箱黑名单（行 60-65）、邀请码 reward（`SELECT FOR UPDATE`）、IntegrityError 兜底 |
| GET | `/api/auth/invite-info` | `get_invite_info` (412-450) | `GetInviteInfo` | `FRONTEND_BASE_URL` 必填；生产禁止用请求头拼接 |
| POST | `/api/auth/forgot-password` | `forgot_password` (452-471) | `ForgotPassword` | 限流 5/60s |
| POST | `/api/auth/change-password` | `change_password` (473-492) | `ChangePassword` | 限流 5/600s |
| POST | `/api/auth/bind-email` | `bind_email` (494-515) | `BindEmail` | UNIQUE 约束兜底防 TOCTOU |
| POST | `/api/auth/token` | `login` (517-549) | `Login` | OAuth2 表单；同时设 cookie；native 端额外返 token |
| POST | `/api/auth/login-by-email` | `login_by_email` (551-589) | `LoginByEmail` | 同上但用邮箱+密码 |
| GET | `/api/auth/me` | `read_users_me` (591-597) | `Me` | ADMIN_UID 自动同步 `is_admin=true` |
| POST | `/api/auth/logout` | `logout` (599-610) | `Logout` | 删 cookie |
| GET | `/api/auth/check-entitlement` | `check_entitlement` (612-619) | `CheckEntitlement` | 检查 VIP 或 Pro |
| PUT | `/api/auth/profile` | `update_profile` (621-628) | `UpdateProfile` | 更新昵称 / 头像 |
| POST | `/api/auth/upload_avatar` | `upload_avatar` (632-712) | `UploadAvatar` | Content-Length 预检 + 分块读 + 大小上限 + Pillow 缩略图（Go 用 imaging.Resize+JPEG 75 质量） |
| POST | `/api/auth/claim_admin` | `claim_admin` (714-725) | `ClaimAdmin` | 用 `crypto/subtle.ConstantTimeCompare` 比较 secret |
| POST | `/api/auth/agent-token` | `create_agent_token` (729-783) | `CreateAgentToken` | `secrets.token_urlsafe(32)` → Go 用 `crypto/rand.Read(32)` 然后 base64.URLEncoding；记录 SHA256 |
| GET | `/api/auth/agent-tokens` | `list_agent_tokens` (785-791) | `ListAgentTokens` | 不返明文 |
| DELETE | `/api/auth/agent-token/{id}` | `delete_agent_token` (793-816) | `DeleteAgentToken` | 仅删自己的 |

### 5.2 基金核心 `/api/*` → `handler/fund.go`
**Python 对应**：`routers/fund.py`（178 行）

| 方法 | 路径 | Python 函数 (行) | Go handler | 说明 |
|---|---|---|---|---|
| POST | `/api/estimate/batch` | `api_estimate` (25-40) | `EstimateBatch` | 最多 50 个 code；调 `FundService.GetEstimates` |
| GET | `/api/history/{code}` | `api_history` (42-52) | `History` | 限流 20/60s/IP |
| GET | `/api/fund/dividends/{code}` | `api_dividends` (56-59) | `Dividends` | |
| GET | `/api/fund/period-rank/{code}` | `api_period_rank` (61-64) | `PeriodRank` | |
| POST | `/api/fund/period-rank/batch` | `api_period_rank_batch` (66-69) | `PeriodRankBatch` | dedupe 输入 |
| GET | `/api/fund/fees/{code}` | `api_fund_fees` (71-74) | `FundFees` | |
| GET | `/api/fund/today-timeline/{code}` | `api_today_timeline` (76-87) | `TodayTimeline` | 只读 Redis key `fund:est:timeline:v1:{code}` |
| GET | `/api/fund/today-rank` | `api_today_rank` (89-108) | `TodayRank` | 只读 Redis key `fund:today_rank:v1`；缓存空时返空结构 |
| GET | `/api/fund/{code}` | `api_detail` (112-142) | `FundDetail` | 详情主路径强依赖；胜率表弱依赖（异常仅返 `winRateStatus=unavailable`） |
| GET | `/api/search` | `api_search` (144-177) | `Search` | 拼音/代码/名称匹配，上限 20 条；只读 fund-info 缓存 |

**Go 路由注册顺序**：Gin 路径优先级与 Python 不同。`/api/fund/dividends/:code` 等具体路径必须用**单独注册**，不会与 `/api/fund/:code` 冲突（Gin 用 radix 树自动处理），但仍建议按 Python 里的顺序写以方便对照。

### 5.3 市场 `/api/market/*` → `handler/market.go`
**Python 对应**：`routers/market.py`（97 行）

| 方法 | 路径 | Python 函数 (行) | Go handler | 说明 |
|---|---|---|---|---|
| GET | `/api/market/status` | `api_market_status` (17-20) | `MarketStatus` | 是否交易日 |
| POST | `/api/market/calculate-dates` | `api_calc_dates` (22-45) | `CalculateDates` | T+N 日期换算 |
| GET | `/api/market/next-trading-day` | `api_next_trading_day` (47-56) | `NextTradingDay` | |
| GET | `/api/market/overview` | `api_market_overview` (58-60) | `MarketOverview` | 首页大盘 |
| GET | `/api/market/indices` | `api_market_indices` (62-65) | `MarketIndices` | 只读 `market:idx:v3` |
| GET | `/api/market/night-est` | `api_market_night_est` (67-78) | `NightEst` | VIP/Pro Only；最多 30 code |
| GET | `/api/market/benchmark-history/{code}` | `api_benchmark_history` (80-96) | `BenchmarkHistory` | 区分指数(`sh000300`) vs ETF(`510300`) |

### 5.4 用户数据 `/api/sync/*` `/api/danmaku/*` `/api/import_*` `/api/notices` → `handler/user.go`
**Python 对应**：`routers/user.py`（658 行）

| 方法 | 路径 | Python 函数 (行) | Go handler | 说明 |
|---|---|---|---|---|
| POST | `/api/sync/upload` | `upload_data` (291-340) | `SyncUpload` | VIP/Pro Only；5MB 上限；PG UPSERT；ETag |
| GET | `/api/sync/meta` | `download_data_meta` (342-358) | `SyncMeta` | 返 size/etag/updated_at |
| GET | `/api/sync/download` | `download_data` (360-372) | `SyncDownload` | |
| POST | `/api/danmaku/send` | `send_danmaku` (375-398) | `DanmakuSend` | 写入 Redis 列表 (TTL=86400) |
| GET | `/api/danmaku/{code}` | `get_danmaku` (400-406) | `DanmakuGet` | 读 Redis 列表 |
| POST | `/api/import_screenshot` | `import_screenshot` (409-542) | `ImportScreenshot` | VIP/Pro Only；含 holdings/watchlist 两模式；并发 4；模糊匹配 |
| POST | `/api/import_transactions` | `import_transactions` (545-641) | `ImportTransactions` | Pro Only；返回结构化交易 |
| GET | `/api/notices` | `get_notices` (643-657) | `Notices` | 增量拉取系统公告 |

**`user.py` 顶部那一坨匹配工具（行 12-203）**——`_normalize_fund_name` / `_match_fund_code` / `_match_watchlist_item` / `_extract_share_class` / `_extract_currency` / `_normalize_match_name` / `_is_valid_ai_trade_date`——**全部搬到 `internal/service/ai/match.go`**，handler 只调一个 `MatchFundCode(aiName, allFunds)` 顶层入口。Go 里模糊匹配用 `github.com/sahilm/fuzzy` 或 `agnivade/levenshtein`（替代 Python 的 `difflib`）。

### 5.5 管理员 `/api/admin/*` → `handler/admin.go`
**Python 对应**：`routers/admin.py`（267 行）

| 方法 | 路径 | Python 函数 (行) | Go handler | 说明 |
|---|---|---|---|---|
| POST | `/api/admin/activate_vip` | `admin_activate_vip` (26-56) | `ActivateVIP` | tier=vip/pro，按 days/months 延期 |
| POST | `/api/admin/activate_vip_all` | `admin_activate_vip_all` (59-104) | `ActivateVIPAll` | 1000 一批，独立 commit |
| POST | `/api/admin/deduct_vip` | `admin_deduct_vip` (107-176) | `DeductVIP` | 无 months/days 则立即撤销 |
| GET | `/api/admin/users` | `admin_get_users` (179-196) | `GetUsers` | 分页搜索 |
| GET | `/api/admin/user-stats` | `admin_user_stats` (199-205) | `UserStats` | |
| GET | `/api/admin/app-version` | `admin_get_app_version` (210-216) | `GetAppVersion` | |
| POST | `/api/admin/app-version` | `admin_set_app_version` (219-255) | `SetAppVersion` | 已存在则更新 |
| POST | `/api/admin/broadcast` | `admin_broadcast` (258-266) | `Broadcast` | 写入 `system_notices` 表 |

**所有处理器都必须 audit log**（`logger.info("[ADMIN_AUDIT] ...")`）。Go 用 `slog.Info("admin_audit", "op", ..., "operator", ...)`。

### 5.6 版本检测 `/api/version` → `handler/version.go`
**Python 对应**：`routers/version.py`（24 行）
- GET `/api/version` → 返最新一条 `app_versions`；无记录时返 v1.0 占位（防 APP 触发更新弹窗）。

### 5.7 Agent 请求 `/api/agent/*` → `handler/agent_request.go`
**Python 对应**：`routers/agent_request.py`（132 行）

| 方法 | 路径 | Python 函数 (行) | Go handler | 说明 |
|---|---|---|---|---|
| POST | `/api/agent/request` | `create_agent_request` (29-75) | `CreateAgentRequest` | 校验 5 种 action_type；IMPORT_* 单次≤300 条≤1MB；每用户最多保留 20 条 |
| GET | `/api/agent/request` | `get_agent_requests` (78-107) | `ListAgentRequests` | 7 天内 PENDING |
| PUT | `/api/agent/request/{id}` | `update_agent_request` (110-131) | `UpdateAgentRequest` | PROCESSED / DISMISSED |

### 5.8 JCTI 性格分析 `/api/jcti/*` → `handler/jcti.go`
**Python 对应**：`routers/jcti.py`（78 行）
- POST `/api/jcti/analyze` (42-77) — VIP/Pro Only；8 种 personality_id；调 AIService.AnalyzeJCTI。

### 5.9 博客统计 `/api/public/*` → `handler/public.go`
**Python 对应**：`routers/public.py`（78 行）
- GET `/api/public/blog-stats` (37-78) — 用 `X-Blog-Stats-Key` 或 `Authorization: Bearer` 校验（`crypto/subtle.ConstantTimeCompare`）；同时读 Redis（今日访问数）和 DB（累计用户数）。

### 5.10 健康检查 `/api/health` `/api/health/redis` → `handler/health.go`
**Python 对应**：`routers/health.py`（78 行）
- GET `/api/health` (50-60) — DB select 1 + Redis ping。任一 error → 503，fallback 仍 200(degraded)。
- GET `/api/health/redis` (63-77) — 返回连接池/maxclients 诊断信息。

---

## 6. Service 层重写指南

### 6.1 AkshareService（**整个项目最复杂**）
**Python 对应**：`services/akshare_service.py`（3511 行）

由于过于庞大，Go 里**必须拆分**为 `internal/service/akshare/` 下的多个文件（见 §1 目录）。下表给出主功能与建议的 Go 落位（行号引用原文）：

| 功能模块 | Python 行段 | Go 文件 | 说明 |
|---|---|---|---|
| HTTP 客户端 + UA/Referer/超时 | 227-242 | `client.go` | Go：`http.Client{Timeout: 30s, Transport: &http.Transport{MaxIdleConnsPerHost: 100}}` + 全局 UA |
| `_TwelveDataRateLimiter` 令牌桶 | 34-87 | `twelve_data.go` | Go：用带容量 channel + `time.Ticker` 每分钟补 N 个令牌；或 `golang.org/x/time/rate.Limiter` |
| `patch_akshare_session` (urllib3 Retry/Adapter) | 133-188 | 不需要 | Go 直接在 `http.Client` 配 `transport` |
| akshare 网络故障熔断（5 次 → 120s 冷却） | 190-209 | `cooldown.go` | `sync.Mutex` + `time.Time` 计数 |
| `_run_with_retry` 重试包装 | 365-410 | `client.go` | `for attempt := 0; attempt < 3; attempt++` + 指数退避 |
| 时区与时间工具 (`get_beijing_time`, `parse_datetime_like`) | 253-285 | `trading_calendar.go` | `time.LoadLocation("Asia/Shanghai")` |
| 美股 PRE/OPEN/POST 判定 (`get_us_market_status`) | 306-318 | `trading_calendar.go` | `time.LoadLocation("America/New_York")` |
| 动态 TTL (`get_dynamic_ttl`) | 335-358 | `trading_calendar.go` | |
| 交易日 / 节假日 (`is_trading_day`, `get_next_trading_day`, `add_trading_days`, `sub_trading_days`) | 416 起 | `trading_calendar.go` | Go 没有 akshare 节假日；走 Redis 缓存 + 自维护节假日表（年初拉一次：东财日历 API 或上交所） |
| 历史净值 (`fetch_fund_history_akshare`) | — | `history.go` | URL：`https://api.fund.eastmoney.com/f10/lsjz?...` |
| 实时估值 (`fetch_estimate` 走 fundgz.1234567.com.cn) | — | `estimate.go` | 抓 `jsonpgz(...)`，正则 `r'jsonpgz\s*=?\s*\((.*)\)'` |
| 基础信息 / 持仓 / 行业 (`get_fund_basic_info`) | — | `basic_info.go` | 走东财 pingzhongdata + tiantian + 蛋卷雪球；DB 持久化到 `fund_basic_info` |
| 分红 (`fetch_dividends`) | — | `dividends.go` | |
| K 线（A 股 push2his.eastmoney / 港股 / 美股 Tencent / Twelve Data） | — | `kline.go` | 多源 fallback |
| 大盘指数 (`refresh_market_indices`) | — | `market_index.go` | 5-way 并发 (Tencent + Stooq...) |
| 板块涨幅榜 (`refresh_sector_rankings`) | — | `rankings.go` | TTL 11 分钟；long_ttl 模式持续到次日开盘 |
| 基金涨幅榜 (`refresh_fund_rankings_with_ttl`) | — | `rankings.go` | 长 TTL 到次日 20:00 |
| 全量基金列表 (`refresh_fund_info_all`, `read_fund_info_all`) | — | `basic_info.go` | TTL 25h；启动时一次性预热 |
| 全日净值快照 (`read_daily_nav`) | — | `history.go` | `market:daily_nav_em:v1` |
| 外汇 (USD/HKD CNH) | — | `forex.go` | 走 Twelve Data |
| 白银期货 / 黄金现货 | — | `forex.go` 或 `kline.go` | |
| 单股 K 线预热（P5 核心 30 股） | — | `kline.go` | 函数名 `prewarm_core_stock_kline_change`，需保留 |
| Holdings stock changes 批量缓存 | — | `kline.go` | 函数名 `_fetch_holdings_stock_changes` |

**Go 实现要点**：
- "in-flight 单飞"（`_inflight_estimates` 等 dict） → 直接用 `golang.org/x/sync/singleflight.Group`，每个域名一个 group。
- 并发信号量 (`AKSHARE_*_CONCURRENCY`) → `chan struct{}` 缓冲 + `defer <-ch`，或 `golang.org/x/sync/semaphore`。
- 正则匹配 → `regexp.MustCompile` 在 var 块里。
- akshare Python 库**不要试图找 Go 替代**，全部用 `net/http` 自己 GET 它后面的 HTTP API（Eastmoney / Sina / Tencent / Twelve Data），URL 在 Python 源里能看到。

### 6.2 AIService
**Python 对应**：`services/ai_service.py`（1083 行）

| 功能 | Python 行段 | Go 文件 | 说明 |
|---|---|---|---|
| 规则化板块表 `_BROAD_INDEX_PATTERNS`/`_SECTOR_LABELS`/`_SECTOR_ALIASES`/`_NAME_SECTOR_PATTERNS` | 58-187 | `sector_rules.go` | 抄常量；Go 用 `[]struct{Pattern *regexp.Regexp; Label string}` |
| 图片归一化 `_normalize_uploaded_image_sync` | 190-205 | `image.go` | `imaging.Resize(src, w, h, imaging.Lanczos)` → JPEG 60 质量 |
| OpenRouter 调用 (`_build_openrouter_payload`, `_extract_openrouter_text`) | 322-373 | `openrouter.go` | URL=`{base}/chat/completions`，Header `Authorization Bearer` + `X-OpenRouter-Title: HuahuaDaily` |
| Gemini 调用 (`_build_gemini_payload`, `_extract_gemini_text`) | 306-351 | `gemini.go` | URL=`https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent`，Header `x-goog-api-key` |
| 代理支持 (`_get_proxies`) | 278-295 | `client.go` | Go：`http.Transport.Proxy = http.ProxyURL(...)` |
| Provider 回退序 (`_get_provider_order`) | 222-233 | `ai_service.go` | 主 provider 失败按 `["openrouter","gemini"]` 兜底 |
| JSON 健壮解析 (`_normalize_json_like_text`, `_parse_json_array`, `_extract_partial_objects`) | 468-571 | `ai_service.go` | 不能丢；AI 经常返回不规范 JSON（带 ```json``` 围栏、单引号、尾逗号、被截断）。Go 用 `encoding/json` + 自写 brace-matching |
| 持仓截图 OCR (`analyze_screenshots`) | 757-810 | `ai_service.go` + `prompts.go` | |
| 自选截图 OCR (`analyze_watchlist_screenshots`) | 813-879 | 同上 | |
| 交易截图 OCR (`analyze_transaction_screenshots`) | 882-968 | 同上 | `max_output_tokens=8192` |
| JCTI 分析 (`analyze_jcti`) | 971-1036 | 同上 | |
| 板块分类 (`classify_fund_sector`) | 1039-1083 | 同上 | 先规则匹配，再 Gemini |

**Go 关键点**：
- 用 `errgroup.WithContext` 做"OpenRouter 失败回退 Gemini"；不要并发跑两个 provider（浪费配额）。
- 大 ThreadPoolExecutor → Go 直接 goroutine + `semaphore.Weighted(settings.AI_PROVIDER_WORKERS)`。

### 6.3 FundService
**Python 对应**：`services/fund_service.py`（1365 行）

| 公开方法 | Python 行段 | Go 函数 |
|---|---|---|
| `get_market_phase` | 180-188 | `MarketPhase(now, isTrading) string` |
| `get_estimates` | 278-342 | `GetEstimates(ctx, codes []string) []EstimateItem` |
| `get_fund_detail_full` | 1190-1247 | `GetFundDetailFull(ctx, code string) (*FundDetail, error)` |
| `refresh_today_rank` | 1250-1264 | `RefreshTodayRank(ctx) (*TodayRank, error)` |
| `build_rank_from_db` | 1267-1340 | `BuildRankFromDB(ctx, today string) (*TodayRank, error)` |
| `get_market_overview` | 1343-1365 | `GetMarketOverview(ctx) (*Overview, error)` |

私有大函数 `_process_single_fund`（行 415-1110，**695 行**）是整个项目的核心，Go 里建议**拆成 9 个小函数**（每个对应 P1-P9 一个优先级路径），主函数只做调度。

**Go 关键点**：
- 跨用户去重（`_inflight_funds`）→ `singleflight.Group`，key = code。
- 详情锁（`_detail_locks: defaultdict(asyncio.Lock)`）→ `sync.Map[string]*sync.Mutex` 或直接 singleflight。
- 价格/收益所有计算用 `shopspring/decimal`，禁止 `float64`。

### 6.4 NightEstimateService
**Python 对应**：`services/night_estimate_service.py`（1209 行）

| 功能 | Python 行段 | Go 函数 |
|---|---|---|
| 入口 `get_night_est` | 1098-1209 | `GetNightEst(ctx, codes []string) (*NightEstResult, error)` |
| 单基金 `_build_item` | 987-1096 | `buildItem(ctx, code, phase) (*NightItem, error)` |
| 持仓股票变化 `_fetch_night_stock_changes` | 765-985 | `fetchStockChanges(...)` |
| 持仓 contribution `_build_breakdown` | 702-763 | `buildBreakdown(...)` |
| Twelve Data quote `_fetch_twelve_quote` | 354-430 | `fetchTwelveQuote(...)` |
| Tencent 实时 `_fetch_tencent_us_quote` / `_fetch_tencent_realtime_change` | 432-473, 650-693 | `fetchTencentQuote(...)` |
| 美股 session/holiday `_market_phase`/`_detect_holiday`/`_us_settle_grace` | 188-303 | `internal/service/night/market_phase.go`（业务相关，含交易日历查询） |
| 外汇 (USD/HKD) `_fetch_fx_changes` | 575-648 | `fetchFXChanges(...)` |
| 最新净值 `_latest_nav_or_fetch_history` | 239-262 | `latestNAV(...)` |
| `last_good` 持久化与回落 | 69-186 | `saveLastGood/loadLastGood` |

**关键设计**：
- 用 Redis key `fund:night_est:v3:{phase}:{md5_of_codes}` 缓存整批结果（行 1098 附近）；singleflight。
- "last good" 同时写 Redis（TTL 7 天）+ `system_kv` 表，Redis 失效时从 DB 回填、标记 `freshness=stale`。

### 6.5 CalibrationService
**Python 对应**：`services/calibration_service.py`（465 行）

| 功能 | Python 行段 | Go 函数 | 模型 |
|---|---|---|---|
| 读 calibration `get_calibration` | 57-110 | `GetCalibration(ctx, code, holdingsHash) (*Calibration, error)` | Redis-first，DB fallback |
| 持仓漂移检测 `_ensure_holdings_match` | 112-147 | `ensureHoldingsMatch(...)` | hash 变化则重置 β |
| 应用 calibration `apply_calibration` | 175-181 | `Apply(raw decimal.Decimal, calib *Calibration) decimal.Decimal` | `raw * β + α`，冷启动 β=1 α=0 |
| 日常训练 `run_daily_calibration` | 185-221 | `RunDailyCalibration(ctx)` | 后台任务夜里跑 |
| 单基金更新 `_calibrate_single` | 225-437 | `calibrateSingle(...)` | EMA λ=0.85，约 7 个交易日半衰期 |

**常量**：`_LAMBDA = 0.85`、`_MIN_SAMPLES = 10`、`_BETA_MIN_SAMPLES = 10`、`_CACHE_PREFIX = "fund:calib:v1"`、`_CACHE_TTL = 86400`。这些都搬到 Go 包级常量。

### 6.6 MarketIndexService
**Python 对应**：`services/market_index_service.py`（572 行）

`internal/service/market_index/market_index.go`。多源（Tencent / Stooq / 等）抓 5 个大盘指数，5-way 并发。逻辑相对独立。

### 6.7 WinRateService
**Python 对应**：`services/win_rate_service.py`（387 行）

`internal/service/fund/win_rate.go`。
- 公开入口 `get_or_compute(code, confirm_days)`：先查缓存，没有就计算。
- 内部根据历史 NAV 计算持有 N 天的胜率表。
- 失败要降级（详情接口里被弱依赖处理，行 `routers/fund.py:121-140`）。

### 6.8 MailService
**Python 对应**：`services/mail_service.py`（76 行）
- `internal/service/mail/mail.go`：用 `gopkg.in/gomail.v2`。
- 验证码 SHA256→6 位数字（crypto/rand），Redis 存 10 分钟，120 秒发送冷却。
- 4 种 prefix：`reg` / `reset` / `bind` / `agent`。

### 6.9 HoldingsEstimate（持仓估值通用模型）
**Python 对应**：`services/holdings_estimate.py`（111 行）

直接搬到 `internal/service/fund/holdings_estimate.go`：
- 常量 `MIN_HOLDINGS_COVERAGE=25`、`MIN_LOW_CONCENTRATION_COVERAGE=20`、`MIN_TOP10_COVERAGE_RATIO=0.90`。
- `HoldingsEstimateResult` 结构体。
- 函数 `NavDateOffset`、`ParseHoldings`、`HoldingsHashFromJSON`、`HasEnoughHoldingsCoverage`、`EquityPctForEstimate`、`BuildHoldingsEstimate`。
- 这里全程用 `shopspring/decimal`。

---

## 7. 后台任务 (`internal/task/background.go`)

**Python 对应**：`tasks/background_refresh.py`（556 行）

实现：`func Run(ctx context.Context, deps Deps) error`，内部死循环 `select { case <-ctx.Done(): return; case <-ticker.C: ... }`，ticker 60 秒。

### 7.1 分布式领导锁
**Python 对应**：行 38-103（lock key `lock:bg_refresh_loop:v1`，TTL 180s，60s 心跳）
- Go 实现：Redis `SET NX EX` 拿锁，每 60 秒续期；续期失败则停 sub-tasks，下个 tick 重新抢。
- `leader_id = pid:<uuid>`，用 Lua 脚本保证"只有持有者才能删"。

### 7.2 子任务调度（按 Python 顺序保留，全部要做）

| # | 任务 | 触发条件（北京时间） | 用途 |
|---|---|---|---|
| Startup | 一次性预热全量基金列表 | 进程启动后第一次 | `AkshareService.RefreshFundInfoAll()` |
| ① | 每日全量基金列表刷新 | 04:00-04:15 每天一次 | TTL 25h |
| ⑪ | P5 核心 30 股 K 线预热 | 06:00-09:00 每 60s 一只 | `PrewarmCoreStockKlineChange` |
| ⑪' | P5 预热日报 | 09:00-09:15 一次 | 仅日志 |
| ⑫ | 美股盘后快照 | 04:00-09:00 每 60s 一批 (15 只) | `NightEstimateService.GetNightEst` |
| ⑧ | 大盘指数缓存 | 交易日 300s/次，非交易日 1800s/次 | `RefreshMarketIndices` |
| ②-a | 板块榜短 TTL 刷新 | 09:15-15:05 交易日 600s/次 | `RefreshSectorRankings(false)` |
| ②-b | 板块榜长 TTL 收尾 | 15:05-15:20 交易日一次 | `RefreshSectorRankings(true)` |
| ⑨-a | 今日涨跌榜（盘中聚合） | 09:30-15:05 交易日 300s/次 | `FundService.RefreshTodayRank` |
| ⑨-b | 今日涨跌榜（收盘 final） | 15:05-15:30 交易日一次 | 长 TTL 到次日 09:00 |
| ⑨' | timeline TTL 收尾 sweep | 15:05-15:30 交易日一次 | 修正 Friday/节假日前 TTL |
| ③ | 基金涨幅榜 | 20:00-23:59 交易日一次 | TTL 到次日 20:00 |
| ⑥ | α/β 校准 | 23:30-23:45 交易日一次 | `CalibrationService.RunDailyCalibration` |
| ⑦ | 胜率表更新 | 凌晨 | `WinRateService.RecomputeAll` |

**Go 调度建议**：写一个 `type Schedule struct { Name string; Should func(now time.Time, isTrading bool, state *State) bool; Run func(ctx context.Context) error }`，循环跑。每个 sub-task 在执行前后都要 `if !leader.Holding() { break }` —— 对应 Python 里的 `_ensure_leadership()`。

---

## 8. 数据库模型

**Python 对应**：`models.py`（162 行）

GORM 标签示例（**只列示意，结构以原 Python 为准**）：

| GORM 模型 | 表名 | Python 对应 | 备注 |
|---|---|---|---|
| `User` | `users` | 15-36 | uid 默认 8 位数字（`crypto/rand`），bcrypt 哈希密码，`invited_by` 加索引 |
| `UserData` | `user_data` | 38-44 | `user_id` UNIQUE；`data_etag VARCHAR(32)` |
| `FundNav` | `fund_navs` | 46-59 | NUMERIC(10,4)/(7,4)；`(fund_code,date)` UNIQUE 索引 |
| `FundSector` | `fund_sectors` | 61-66 | |
| `AppVersion` | `app_versions` | 68-75 | |
| `FundBasicInfo` | `fund_basic_info` | 77-88 | `code` 主键；`equity_pct` 可空；`holdings_json/industry_json/fees_json` 用 TEXT |
| `AgentToken` | `agent_tokens` | 90-103 | 存 SHA256，不存明文 |
| `AgentRequest` | `agent_requests` | 106-116 | `id` 主键 UUID 字符串 |
| `FundCalibration` | `fund_calibrations` | 119-144 | α/β NUMERIC(10,6)，coverage NUMERIC(7,2)；`holdings_hash VARCHAR(16)` |
| `SystemNotice` | `system_notices` | 147-153 | id UUID hex |
| `SystemKV` | `system_kv` | 156-161 | 用于"夜估快照 last_good" 持久化 |

**幂等迁移**（lifespan 里的 `_migrate_*`）→ Go 用 `gorm.AutoMigrate` + 手写 SQL（写到 `internal/db/migrate.go`），全部要保留：
- `_migrate_empty_emails` (lifespan.py:19-29)
- `_migrate_equity_pct` (32-48)
- `_migrate_calibration_numeric` (51-80) — Float → NUMERIC
- `_migrate_calibration_holdings_hash` (83-100)
- `_migrate_calibration_training_state` (103-139)
- `_migrate_agent_token_columns` (142-171)
- `_migrate_user_data_columns` (174-199)

所有迁移**幂等**且**失败必须 panic**（数据完整性优先于服务可用性）。

---

## 9. Schemas / DTO

**Python 对应**：`schemas.py`（239 行）

Go 用 struct + `binding` tag（gin 默认接 `validator/v10`）：
- `USERNAME_REGEX = "^[a-zA-Z0-9]{4,20}$"` → `binding:"required,alphanum,min=4,max=20"`
- 密码强度（行 12-21）：在 handler 里手动 check（min 8、含字母+数字、不含空格）。
- `EmailStr` → `binding:"required,email"`。
- `FundCodesRequest`（行 105-119）：单次最多 50 代码；每个必须是 6 位数字。
- `EmailVerifyType` 枚举 → `binding:"oneof=REGISTER FORGOT_PASSWORD BIND AGENT_TOKEN"`。
- `AgentTokenCreate.scope` 自定义校验 → 单独 validator（参考行 184-203）。
- 所有响应 DTO 都要带 `json` 标签；BeijingTime 输出 ISO 8601 字符串。

---

## 10. 缓存层 (`pkg/cache/cache.go`)

**Python 对应**：`cache.py`（432 行）

需要的方法（Go 接口）：
```
Get(ctx, key) ([]byte, error)            // 序列化交给上层
Set(ctx, key, val, ttl) error
Incr(ctx, key, amount, ttl) (int64, error)
SAdd(ctx, key, member, ttl) (int64, error)
SCard(ctx, key) (int64, error)
Delete(ctx, key) error
DeleteMany(ctx, keys []string) (int64, error)
PushList(ctx, key, item, ttl) error
GetList(ctx, key, start, end) ([]any, error)
ExpireOnly(ctx, key, ttl) (bool, error)
ExpireMany(ctx, keys, ttl) error
AcquireLock(ctx, key, ttl, owner) (bool, error)
ReleaseLock(ctx, key, owner) error
ScanKeys(ctx, pattern, limit) ([]string, error)   // 必须用 SCAN 而非 KEYS
GetMany(ctx, keys) ([][]byte, error)
Pipeline() redis.Pipeliner  // 直接暴露给限流中间件
```

**进程内降级缓存**（行 31-104）：
- 容量上限 `_MEM_MAX_KEYS = 2000`，LRU 淘汰。
- Go 用 `container/list` + `map`（手写 LRU）或 `github.com/hashicorp/golang-lru/v2`。
- 主动过期清理：启动一个 goroutine 每 5 分钟扫一遍。
- Redis 不可用判定：`is_connection_error` 行 125-133（按错误信息匹配），首次失败打一次 WARN。

**Lock 关键点**：
- `acquire_lock` 用 `SET NX EX`；同 owner 重入则刷 TTL。
- `release_lock` 用 Lua（`if get == owner then del end`）防止误删。
- Redis 失联时退化到本地 map，单实例部署不被锁死。

---

## 11. 安全工具（`internal/security/` + `pkg/ipx/` + `pkg/ratelimit/`）

**Python 对应**：`security.py`（166 行）

> Python 源里把"通用算法"和"业务键名/字段"混在一个文件，Go 版按"是否含业务知识"拆开：
> - 算法（IP 解析、滑窗）→ `pkg/`
> - 配置（限流键、ETag 字段剔除规则、基金代码正则）→ `internal/security/`

| 函数 | Python 行 | Go 落位 | 备注 |
|---|---|---|---|
| `get_client_ip(request)` | 41-75 | `pkg/ipx/ip.go: ClientIP(headers http.Header, peer string, trustedProxyCount int) string` | 不依赖 Gin，便于复用 |
| `check_rate_limit` 算法 | 78-105 | `pkg/ratelimit/sliding_window.go: Allow(ctx, redis, key, limit, period) (bool, error)` | 只关心"key+limit+period"，不知道 key 怎么拼 |
| `check_rate_limit` 业务封装 | 同上 | `internal/security/ratelimit.go: CheckRateLimit(ctx, ns, keySuffix, limit, period, detail) error` | 拼 `rl:{ns}:{keySuffix}`，错时抛 `HTTPException` 等价 |
| `enforce_cooldown` | 108-119 | 同上文件 | 简单 `SET NX EX` |
| `is_valid_fund_code` | 122-123 | `internal/security/validators.go: IsValidFundCode(code) bool` | 业务正则 |
| `is_valid_benchmark_code` | 126-127 | 同上 | 业务正则 |
| `build_sync_etag(payload)` | 138-157 | `internal/security/etag.go: BuildSyncEtag(payload string) string` | 含 funds[*] 字段过滤，业务相关 |
| `parse_bearer_subject(header)` | 160-165 | `pkg/util/auth.go: ParseBearerSubject(header string) string` 或直接放 `internal/middleware/auth.go` 当 helper | |

---

## 12. 配置 (`internal/config/config.go`)

**Python 对应**：`config.py`（126 行）

Go 用 `viper` 或 `caarlos0/env`，所有字段一一对应。**重要默认值**保留：
- `SECRET_KEY` 默认 `smartfund_secret_key_change_me` → 生产环境必须改（在 `app.New` 启动检查）。
- `ACCESS_TOKEN_EXPIRE_MINUTES = 60*24*7`（一周）。
- `DB_POOL_SIZE = 20`、`DB_MAX_OVERFLOW = 40`（Go GORM：`db.DB().SetMaxOpenConns(60)`，SetMaxIdleConns(20)）。
- 各 concurrency 字段（AKSHARE_*_CONCURRENCY / AI_PROVIDER_WORKERS 等）：Go 里建对应的 `semaphore.Weighted`。
- `GLOBAL_RATE_LIMIT_CALLS=1200`、`GLOBAL_RATE_LIMIT_PERIOD=60`。
- Redis 池：`REDIS_MAX_CONNECTIONS=100`、`REDIS_SOCKET_TIMEOUT=5s`、`REDIS_CONNECT_TIMEOUT=3s`。
- `DISABLE_SSL_VERIFY=false`（开发环境用，Go 用 `tls.Config{InsecureSkipVerify}`，**生产禁用**）。
- 启动时检测默认密钥并打 WARN（行 117-125）。

---

## 13. 工具函数 (`pkg/util/`)

**Python 对应**：`utils.py`（33 行）
- `utcnow()` → `pkg/util/time.go: Utcnow() time.Time`，用 `time.Now().UTC()`（不要去掉时区，GORM 会自己处理）。
- `round_half_up(value, digits)` → `pkg/util/decimal.go: RoundHalfUp(v decimal.Decimal, digits int32) decimal.Decimal`，用 `v.Quantize(decimal.New(1, -digits), decimal.RoundHalfUp)`（**直接 `Round()` 是银行家舍入，不一样**）。
- 北京时间 / 美东时间常量也放这里：`var BJ = time.FixedZone("Asia/Shanghai", 8*3600)`、`var NY, _ = time.LoadLocation("America/New_York")`。

---

## 14. 一次性脚本 (`scripts/`)

**Python 对应**：`scripts/*.py`

是否搬到 Go：**优先级低**。可以保留 Python 文件作为运维工具，或者按需求逐个搬到 `cmd/migrate/`、`cmd/cleanup/` 下：
- `add_pro_tier.py` / `fix_fund_type.py` / `cleanup_*.py` / `migrate_*.py` / `reset_calibration.py`

如果搬：每个写成 `cmd/<name>/main.go`，复用 `internal/config` + `internal/db` 初始化。

---

## 15. 测试

**Python 对应**：`tests/test_*.py`（12 个文件）

Go 测试放对应包下：
- `internal/service/fund/fund_service_test.go`
- `internal/service/calibration/calibration_test.go`
- `internal/service/night/night_estimate_test.go`
- `internal/service/win_rate/win_rate_test.go`
- `internal/service/akshare/cache_logic_test.go`（对应 `test_cache_logic.py`）
- handler 集成测试用 `httptest.NewServer + gin.Engine`，建议放 `internal/handler/<name>_test.go`。

测试关键点：mock Redis 用 `github.com/alicebob/miniredis/v2`；mock HTTP 用 `httptest.NewServer`。

---

## 16. Go 新手常见陷阱与对照

| Python 习惯 | Go 必须的写法 | 不这样做的后果 |
|---|---|---|
| `async def fn(...)` | 普通 `func fn(...)` + 内部并发用 `go` | 不需要 `async/await` 关键字 |
| `await asyncio.gather(...)` | `errgroup.Go(...)` + `g.Wait()` | 不能丢错误 |
| `asyncio.Semaphore(N)` | `sem := semaphore.NewWeighted(N)` + `sem.Acquire(ctx, 1)` / `sem.Release(1)` | 失控并发拖垮上游 |
| `try: ... except: pass` | `if err != nil { ... }` — 每次必须显式处理 | Go 不允许"默默吃错误" |
| `decimal.Decimal` | `decimal.NewFromString` / `decimal.NewFromFloat` / `.Mul/.Add` | 用 `float64` 会出现 0.1+0.2 != 0.3 |
| ORM 自动事务 | `db.Transaction(func(tx *gorm.DB) error { ... })` | 多语句不在事务里就没有原子性 |
| `request.state.xxx` | `c.Set("xxx", val)` / `c.MustGet("xxx")` | 跨中间件传值 |
| `HTTPException(400, "...")` | `c.AbortWithStatusJSON(400, gin.H{"detail": "..."})` + `return` | 忘 return 后续代码还会跑 |
| `with SessionLocal()` | `db := gormDB.WithContext(ctx)`，连接由 GORM 池管理 | 不要 `defer db.Close()` 关掉单个 *gorm.DB |
| `time.sleep(N)` 在 async | `select { case <-ctx.Done(): return; case <-time.After(N): }` | 不监听 ctx 会让进程关不掉 |
| 字典默认值 `dict.get(k, v)` | Go：`if v, ok := m[k]; ok { ... } else { v = default }` | 没有"默认值"语法糖 |

---

## 17. 重写步骤建议（顺序很重要）

1. **基础设施先行**：`config` → `db` → `cache` → `security` → `model` → `repository`（先写接口与最常用的 `user_repo` / `fund_basic_info_repo` / `agent_token_repo`） → 一个空的 `gin.Engine` 跑起来。
2. **router 骨架**：先在 `internal/router/router.go` 把 10 个子路由文件全部建出来（仅占位 `// TODO`），让 §5 的功能清单在目录层面**先可见**，避免后续遗漏。
3. **不依赖外部 API 的 handler 先做**：`health` / `version` / `admin` / `auth`（含 JWT 中间件） / `public`。每个 handler 写完同步把 router 行打开、把对应 repository 方法补齐。
4. **核心数据源封装**：`internal/service/akshare/` 全部 —— 这是后续所有功能的底座。**写完一个就先用单元测试 + `httptest.NewServer` mock 上游** 验证。
5. **基础业务**：`fund_service` / `win_rate` / `holdings_estimate` / `market_index`（这些 service 调 `fund_nav_repo` / `fund_basic_info_repo`）。
6. **handler/fund.go / market.go** 接入。
7. **AI 与用户数据**：`ai_service` → `handler/user.go`（截图 / 同步 / 弹幕）；新增 `user_data_repo`。
8. **夜估**：`calibration_repo` → `calibration` service → `night_estimate` → `market.NightEst` handler。
9. **后台任务**：`task/background.go` 一次性接入所有 sub-task；先用 `--no-leader-lock` 模式本地跑通，再开启分布式锁。
10. **Agent 相关**：`agent_request_repo` + `agent_request` handler + Auth 中间件里的 `AgentToken` 路径校验。
11. **JCTI / 邀请裂变 / 截图导入**：业务尾部。
12. **静态文件 & SPA fallback**：最后接入。
13. **端到端联调**：用真实 APP / 前端打全部接口；对比 Python 版的响应字段是否一一对齐（**字段名拼写、大小写必须 100% 一致**，前端硬编码读它们）。

---

## 18. 最后的检查清单（防遗漏）

- [ ] 所有 §5 表里的 HTTP 接口都有 `router/<file>.go` 注册行 + `handler/<file>.go` 处理函数，且响应 JSON 字段名与 Python 一致。
- [ ] handler **不直接** import `gorm.io/gorm`，所有 DB 访问都走 `repository` 包。
- [ ] `pkg/` 下所有包**不 import `internal/`**（用 `go vet ./pkg/...` 加 `go list -deps` 抽查依赖图）。
- [ ] 跨表写操作通过 `db.Transaction(...)` 包裹，repository 提供 `XxxTx(ctx, tx, ...)` 变体。
- [ ] `SELECT FOR UPDATE`（注册时锁邀请人）有专用 `LockByIDForUpdateTx` 方法，对应 `routers/auth.py:390-403`。
- [ ] 所有 §7 后台 sub-task 都注册到 `internal/task/background.go`。
- [ ] 所有 §8 GORM 模型都执行了 `AutoMigrate`，且 §8 末尾的 7 个迁移函数都有 Go 等价物（幂等 + 失败 panic）。
- [ ] 中间件顺序与 Python 一致（ProxyHeaders → RateLimit → BlogStats → CORS → GZip → 路由）。
- [ ] Auth 中间件正确处理 `Authorization: AgentToken xxx` 与 scope 白名单。
- [ ] 限流 / 缓存 在 Redis 不可用时降级到本地 LRU。
- [ ] `_safe_resolve` 防目录穿越（SPA 兜底 + `/static/`）。
- [ ] 上传头像 / 截图导入有 Content-Length 预检 + 分块读 + 大小硬上限。
- [ ] AI 调用：JSON 截断恢复、provider 失败回退（OpenRouter → Gemini）、超时与并发信号量都接好。
- [ ] CalibrationService 的 λ=0.85 / 冷启动 β=1,α=0 / holdings_hash 漂移检测都保留。
- [ ] Twelve Data 令牌桶限速、akshare 5 次失败 120s 熔断这两个"上游保护"机制都保留。
- [ ] 后台任务用 Redis 分布式领导锁（`lock:bg_refresh_loop:v1`，TTL 180s，60s 心跳）；丢锁后立即停手。
- [ ] 所有的 `[ADMIN_AUDIT]` / `[AUTH_AUDIT]` 日志都保留（用 `slog.Info` 带结构化字段）。
- [ ] 优雅退出：先 `http.Server.Shutdown`，再停后台 ctx，再关 Redis/DB/HTTP/AI client，最后释放 leader lock。

---

完成这份指南所列的每一项，新人即可在不读 Python 源的情况下，逐文件、逐接口地把后端按对应关系搬到 Go/Gin。
