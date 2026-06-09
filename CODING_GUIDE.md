# 从零开始 Go 重写 Huahua 服务 — 新手通关指南

> **这份指南是做什么的？**
>
> 本项目是对 `/Users/xqs/Documents/a/baiye-fund-main/backend/`（Python FastAPI）的 Go + Gin 重写。
> 原项目是**花花日记**的后端——一个基金投资助手，提供基金净值查询、实时估值、持仓分析、AI 截图导入等功能。
> 本指南按照 README.md 第 17 条的 13 个步骤，**逐步展开**，配合原 Python 代码对照，让你边写边学 Go 和 Gin。
>
> **前提**：已过 Go 基础语法（变量、函数、结构体、切片、map、指针）。没过的先去 [Go Tour](https://go.dev/tour/) 花 2 小时。

---

## 目录

1. [Step 1：基础设施先行](#step-1基础设施先行)
2. [Step 2：Router 骨架](#step-2router-骨架)
3. [Step 3：不依赖外部 API 的 handler](#step-3不依赖外部-api-的-handler)
4. [Step 4：核心数据源封装 (akshare)](#step-4核心数据源封装-akshare)
5. [Step 5：基础业务 (Fund Service)](#step-5基础业务-fund-service)
6. [Step 6：handler/fund.go / market.go 接入](#step-6handlerfundgo--marketgo-接入)
7. [Step 7：AI 与用户数据](#step-7ai-与用户数据)
8. [Step 8：夜估 (Night Estimate)](#step-8夜估-night-estimate)
9. [Step 9：后台任务](#step-9后台任务)
10. [Step 10：Agent 相关](#step-10agent-相关)
11. [Step 11：JCTI / 邀请裂变 / 截图导入](#step-11jcti--邀请裂变--截图导入)
12. [Step 12：静态文件 & SPA fallback](#step-12静态文件--spa-fallback)
13. [Step 13：端到端联调](#step-13端到端联调)
14. [附录：Go 新手必坑 & 推荐学习路径](#附录go-新手必坑--推荐学习路径)

---

## 先理解原 Python 项目的结构

原项目 `backend/` 目录长这样，先混个眼熟：

```
backend/           ← Python FastAPI 后端
├── main.py        ← 启动入口（uvicorn）
├── config.py      ← 配置（pydantic-settings）
├── database.py    ← SQLAlchemy 数据库连接
├── models.py      ← 数据库模型（User, FundNav 等）
├── schemas.py     ← 请求/响应数据结构
├── security.py    ← 限流、IP提取、基金编码校验
├── cache.py       ← Redis 缓存 + 内存降级
├── utils.py       ← 工具函数
├── core/
│   ├── app_factory.py   ← FastAPI 应用装配
│   └── lifespan.py      ← 启动/停止钩子
├── routers/       ← HTTP 处理器（10个文件）
│   ├── auth.py    ← 注册登录/JWT/AgentToken
│   ├── fund.py    ← 基金查询
│   ├── market.py  ← 市场行情
│   ├── user.py    ← 用户数据同步/弹幕
│   ├── admin.py   ← 管理员操作
│   ├── health.py  ← 健康检查
│   ├── ...        ← version / agent_request / jcti / public
├── services/      ← 业务逻辑层
│   ├── fund_service.py       ← 基金业务
│   ├── akshare_service.py    ← 财经数据源（最重要）
│   ├── ai_service.py         ← AI 调用（OpenRouter → Gemini）
│   ├── calibration_service.py← 夜估校准
│   ├── night_estimate_service.py
│   ├── holdings_estimate.py
│   ├── win_rate_service.py
│   ├── mail_service.py       ← 邮件发送
│   └── market_index_service.py
├── tasks/
│   └── background_refresh.py ← 后台定时任务
├── tests/         ← 12 个测试文件
└── scripts/       ← 运维脚本
```

**重写后的 Go 项目结构（参照 README.md §1）：**

```
huahua-service/     ← 当前 Go 项目（现在只有文件夹骨架）
├── cmd/server/main.go       ← main.py 对应
├── internal/
│   ├── app/app.go           ← app_factory.py 对应
│   ├── config/config.go     ← config.py 对应
│   ├── db/db.go             ← database.py 对应
│   ├── model/*.go           ← models.py 对应
│   ├── schema/*.go          ← schemas.py 对应
│   ├── middleware/*.go      ← app_factory.py 的中间件
│   ├── router/*.go          ← routers/ 对应
│   ├── handler/*.go         ← routers/ py 里的处理函数
│   ├── repository/*.go      ← 新增层：所有数据库操作
│   ├── service/*.go         ← services/ 对应
│   ├── security/*.go        ← security.py 对应
│   └── task/background.go   ← background_refresh.py 对应
├── pkg/cache/               ← cache.py 对应
└── migrations/              ← Python 原版迁移脚本
```

### 对应关系速览（Python → Go）

| Python 文件 | Go 目标 | 难度 |
|---|---|---|
| `main.py` | `cmd/server/main.go` | ☆ |
| `config.py` | `internal/config/config.go` | ☆ |
| `database.py` | `internal/db/db.go` | ☆ |
| `models.py` | `internal/model/*.go` | ☆☆ |
| `schemas.py` | `internal/schema/*.go` | ☆ |
| `security.py` | `internal/security/*.go` | ☆☆ |
| `cache.py` | `pkg/cache/cache.go` | ☆☆☆ |
| `core/app_factory.py` | `internal/app/app.go` | ☆☆ |
| `routers/health.py` | `router/health.go` + `handler/health.go` | ☆ |
| `routers/auth.py` | `router/auth.go` + `handler/auth.go` + ... | ☆☆☆ |
| `services/akshare_service.py` | `service/akshare/` 多个文件 | ☆☆☆ |
| `services/fund_service.py` | `service/fund/fund_service.go` | ☆☆☆ |
| `tasks/background_refresh.py` | `task/background.go` | ☆☆☆ |
| 新增层 `repository/` | **Go 独有**，Python 里是嵌在 router 里的 | — |

> **关键差异**：Python 用 FastAPI 直接在 router 里写 SQLAlchemy 查询。
> Go 强制分层：**handler 不碰数据库**，所有 DB 操作走 `repository` 层。
> 这是"做对"和"做错"的核心区别。

---

## 语言对照：Python → Go 速记

在开始之前，先把最常用的 Python 写法对应到 Go：

### 1. 变量、函数、结构体

```python
# Python
name = "hello"
def add(a, b):
    return a + b
class User:
    def __init__(self, name):
        self.name = name
```

```go
// Go
name := "hello"
func add(a, b int) int {
    return a + b
}
type User struct {
    Name string
}
// Go 没有构造函数，用函数代替
func NewUser(name string) *User {
    return &User{Name: name}
}
```

### 2. 错误处理

```python
# Python - 用 try/except
try:
    result = do_something()
except Exception as e:
    print(f"失败: {e}")
    return None
```

```go
// Go - err 是返回值，必须检查
result, err := doSomething()
if err != nil {
    log.Printf("失败: %v", err)
    return nil, err
}
// result 可以安全使用
```

### 3. HTTP 处理

```python
# Python FastAPI
@router.get("/hello/{name}")
async def hello(name: str):
    return {"message": f"Hello {name}"}
```

```go
// Go Gin
r.GET("/hello/:name", func(c *gin.Context) {
    name := c.Param("name")
    c.JSON(http.StatusOK, gin.H{"message": "Hello " + name})
})
```

### 4. JSON 请求体绑定

```python
# Python Pydantic
class LoginReq(BaseModel):
    username: str
    password: str

@router.post("/login")
async def login(req: LoginReq):
    ...
```

```go
// Go Gin + struct tag
type LoginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

r.POST("/login", func(c *gin.Context) {
    var req LoginReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"detail": err.Error()})
        return
    }
    // 用 req.Username, req.Password
})
```

### 5. 数据库查询

```python
# Python SQLAlchemy
user = db.query(User).filter(User.username == name).first()
if user is None:
    raise HTTPException(404)
```

```go
// Go GORM (在 repository 层)
var user model.User
err := db.WithContext(ctx).Where("username = ?", name).First(&user).Error
if errors.Is(err, gorm.ErrRecordNotFound) {
    return nil, nil  // 业务上 nil 表示"不存在"
}
if err != nil {
    return nil, err  // 真正的数据库错误
}
return &user, nil
```

---

## Step 1：基础设施先行

> **这步做什么**：让程序能从零跑起来，输出一句日志，证明 Gin 正常工作了。

对应 Python 文件：`config.py`、`database.py`、`models.py`、`main.py`、`core/app_factory.py`

### 1.1 config.go —— 配置管理

**Python 原版参考**：[config.py](Python backend → config.py)

Python 用 `pydantic-settings` + 环境变量读配置。Go 里用标准 `os.Getenv`：

```go
// internal/config/config.go
package config

import (
    "os"
    "strconv"
)

type Config struct {
    Port    string
    DB      DatabaseConfig
    Redis   RedisConfig
    SecretKey string
}

type DatabaseConfig struct {
    DSN          string
    MaxOpenConns int
    MaxIdleConns int
}

type RedisConfig struct {
    Addr     string
    Password string
    DB       int
}

// Load 读环境变量构造 Config，相当于 Python Settings() 
func Load() *Config {
    return &Config{
        Port: getEnv("PORT", "8080"),
        SecretKey: getEnv("SECRET_KEY", "smartfund_secret_key_change_me"),
        DB: DatabaseConfig{
            DSN:          getEnv("DATABASE_URL", "postgres://localhost:5432/huahuadaily?sslmode=disable"),
            MaxOpenConns: getEnvInt("DB_POOL_SIZE", 20),
            MaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 10),
        },
        Redis: RedisConfig{
            Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getEnvInt("REDIS_DB", 0),
        },
    }
}

// 工具函数：读环境变量，没设置就用默认值
func getEnv(key, defaultVal string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return defaultVal
}
```

**对比 Python 源码**：

```python
# Python：用 pydantic-settings，自动读 .env 文件
class Settings(BaseSettings):
    SECRET_KEY: str = "smartfund_secret_key_change_me"
    DATABASE_URL: str = "postgresql+psycopg2://localhost:5432/huahuadaily"
    DB_POOL_SIZE: int = 20
    REDIS_HOST: str = "localhost"
    # ... 更多配置

    class Config:
        env_file = ".env"

settings = Settings()  # 全局单例
```

> **差异**：Python 的 `pydantic-settings` 自动从环境变量读，Go 需要手动 `os.Getenv`。
> 但 Python 帮不了你检查`_`和拼写错误，Go 手动写反而更清晰。

### 1.2 db/db.go —— 连接数据库 + Redis

**Python 原版参考**：[database.py](Python backend → database.py) + [cache.py](Python backend → cache.py)

Python 用 SQLAlchemy 的 `create_engine` + `SessionLocal`。
Go 用 GORM 操作 PostgreSQL，`go-redis` 操作 Redis。

```go
// internal/db/db.go
package db

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "huahua-service/internal/config"

    "github.com/redis/go-redis/v9"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

// 全局变量——整个项目共用这两个连接
var (
    GORM  *gorm.DB
    Redis *redis.Client
)

func InitGORM(cfg config.DatabaseConfig) {
    // postgres.Open 接收 DSN (postgres://user:pass@host:port/db)
    db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),  // 开发时打印 SQL
    })
    if err != nil {
        panic(fmt.Sprintf("数据库连接失败: %v", err))
    }

    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
    sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(30 * time.Minute)

    GORM = db
    slog.Info("数据库已连接", "max_open", cfg.MaxOpenConns)
}

func InitRedis(cfg config.RedisConfig) {
    client := redis.NewClient(&redis.Options{
        Addr:     cfg.Addr,
        Password: cfg.Password,
        DB:       cfg.DB,
    })

    // PING 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        panic(fmt.Sprintf("Redis 连接失败: %v", err))
    }

    Redis = client
    slog.Info("Redis 已连接", "addr", cfg.Addr)
}

func Close() {
    if GORM != nil {
        sqlDB, _ := GORM.DB()
        sqlDB.Close()
    }
    if Redis != nil {
        Redis.Close()
    }
}
```

**对比 Python 源码：**

```python
# Python database.py
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker

engine = create_engine(
    settings.DATABASE_URL,
    pool_size=settings.DB_POOL_SIZE,
    max_overflow=settings.DB_MAX_OVERFLOW,
    pool_pre_ping=True,   # 取连接前先 PING
)

SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
```

```python
# Python cache.py —— 对 redis 的封装
import redis.asyncio as redis

class Cache:
    def __init__(self):
        self.redis = redis.Redis(
            host=settings.REDIS_HOST,
            port=settings.REDIS_PORT,
            password=settings.REDIS_PASSWORD,
            db=settings.REDIS_DB,
            socket_timeout=settings.REDIS_SOCKET_TIMEOUT,
            connect_timeout=settings.REDIS_CONNECT_TIMEOUT,
            max_connections=settings.REDIS_MAX_CONNECTIONS,
        )

cache = Cache()  # 全局单例
```

### 1.3 model/*.go —— 数据库模型

**Python 原版参考**：[models.py](Python backend → models.py)（161 行，~10 个表）

把 `model.py` 里每个 SQLAlchemy 类转成 GORM 结构体：

```go
// internal/model/user.go
package model

import "time"

// User 对应 users 表
// GORM 默认把 User 映射到表名 users（小写复数）
type User struct {
    ID           int64      `gorm:"primaryKey;autoIncrement"`
    Username     string     `gorm:"uniqueIndex;size:50"`
    Email        string     `gorm:"uniqueIndex;size:100"`
    HashedPassword string   `gorm:"size:255"`
    Avatar       string     `gorm:"size:255;default:''"`
    Nickname     string     `gorm:"size:50;default:''"`
    UID          string     `gorm:"uniqueIndex;size:32"`
    InvitedBy    *int64     `gorm:"default:null"`          // 指针类型：允许 NULL
    VIPExpiresAt time.Time  `gorm:"default:'2000-01-01'"`
    ProExpiresAt *time.Time `gorm:"default:null"`
    IsAdmin      bool       `gorm:"default:false"`
    CreatedAt    time.Time  `gorm:"autoCreateTime"`
}
```

> **`*int64` 和 `*time.Time` 是什么意思？**
> Go 里指针表示"这个字段可能为 nil"（数据库的 NULL）。
> `InvitedBy *int64` = 数据库里 invited_by 列可以为 NULL。
> 如果写成 `InvitedBy int64`，那它在数据库里永远是 0，没法表示"没有邀请人"。

```go
// internal/model/fund_basic_info.go
package model

import "time"

type FundBasicInfo struct {
    Code         string     `gorm:"primaryKey;size:6"`
    Name         string     `gorm:"size:100"`
    Type         string     `gorm:"size:20"`
    ConfirmDays  int        `gorm:"default:1"`
    FeesJSON     string     `gorm:"column:fees_json;type:text"`
    HoldingsJSON string     `gorm:"column:holdings_json;type:text"`
    IndustryJSON string     `gorm:"column:industry_json;type:text"`
    Sector       string     `gorm:"size:50;default:''"`
    EquityPct    *float64   `gorm:"column:equity_pct;default:null"`
    UpdatedAt    time.Time  `gorm:"autoUpdateTime"`
}

// TableName 自定义表名（默认是 fund_basic_infos，但实际表叫 fund_basic_info）
func (FundBasicInfo) TableName() string {
    return "fund_basic_info"
}
```

**对比 Python models.py 关键片段：**

```python
# Python SQLAlchemy
class FundBasicInfo(Base):
    __tablename__ = "fund_basic_info"
    code = Column(String, primary_key=True, index=True)
    name = Column(String)
    type = Column(String)
    equity_pct = Column(Float, nullable=True)
    # ...
```

### 1.4 repository/user_repo.go —— 数据库操作层

Go 加的层，Python 里没有显式区分。在 Python 里查数据库是直接写在 router 里的：

```python
# Python（routers/auth.py）直接在 handler 里查
def _do_register():
    user = db.query(User).filter(User.username == user_in.username).first()
    if user:
        raise HTTPException(400, "Username already registered")
```

**Go 里必须把这种操作放进 repository：**

```go
// internal/repository/user_repo.go
package repository

import (
    "context"
    "errors"

    "huahua-service/internal/model"

    "gorm.io/gorm"
)

// UserRepository 接口——方便测试时替换为 mock
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*model.User, error)
    GetByUsername(ctx context.Context, username string) (*model.User, error)
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    GetByUID(ctx context.Context, uid string) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
    Update(ctx context.Context, user *model.User) error
}

type userRepo struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
    return &userRepo{db: db}
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil  // 没查到返回 nil, nil——不是错误
    }
    return &user, err
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepo) Update(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Save(user).Error
}
```

> **为什么用 interface？**
> 测试时可以写一个 MockUserRepository 替代真正的数据库，不用连真实的 PostgreSQL。
> 这是 Go 推荐的 DI（依赖注入）模式——依赖是传进来的，不是自己 new 的。

### 1.5 cmd/server/main.go —— 启动入口

**Python 原版参考**：[main.py](Python backend → main.py) + [core/app_factory.py](Python backend → core/app_factory.py)

```go
// cmd/server/main.go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "huahua-service/internal/app"
    "huahua-service/internal/config"
    "huahua-service/internal/db"
)

func main() {
    // 1. 读配置
    cfg := config.Load()

    // 2. 启动日志
    slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    })))

    // 3. 安全警告（参考 Python config.py 的默认密钥检测）
    if cfg.SecretKey == "smartfund_secret_key_change_me" {
        slog.Warn("⚠️ SECRET_KEY 仍为默认值，请在生产环境修改")
    }

    // 4. 连接数据库
    db.InitGORM(cfg.DB)
    db.InitRedis(cfg.Redis)
    defer db.Close()

    // 5. 组装 Gin 引擎（对应 Python create_app()）
    engine := app.New(cfg)

    // 6. 启动 HTTP 服务
    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: engine,
    }

    go func() {
        slog.Info("服务启动", "addr", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("服务异常退出", "error", err)
            os.Exit(1)
        }
    }()

    // 7. 优雅退出
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    slog.Info("正在关闭服务...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
    slog.Info("服务已关闭")
}
```

**对比 Python main.py：**

```python
# Python main.py
from config import settings
from core.app_factory import create_app

app = create_app()

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("PORT", 8080))
    uvicorn.run(app, host="0.0.0.0", port=port, 
                proxy_headers=settings.PROXY_HEADERS,
                forwarded_allow_ips=settings.FORWARDED_ALLOW_IPS)
```

### 1.6 app/app.go —— 装配 Gin Engine

**Python 原版参考**：[core/app_factory.py](Python backend → core/app_factory.py)（~379 行，最重要的文件）

```go
// internal/app/app.go
package app

import (
    "log/slog"

    "huahua-service/internal/config"

    "github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
    // 生产模式不打印调试信息
    if cfg.Mode == "release" {
        gin.SetMode(gin.ReleaseMode)
    }

    r := gin.New()  // 干净引擎，无默认中间件
    r.Use(gin.Recovery())  // 全局 panic 捕获

    // 之后会加更多中间件：限流、CORS、GZip、Auth...
    // 见 Step 3

    // 之后会注册路由
    // router.Register(r, ...)

    slog.Info("应用已装配", "port", cfg.Port)
    return r
}
```

**对比 Python app_factory.py 的关键部分：**

```python
# Python create_app()
def create_app() -> FastAPI:
    app = FastAPI(title="SmartFund API", lifespan=lifespan)
    
    # 中间件顺序很重要！
    app.add_middleware(ProxyHeadersMiddleware, ...)
    app.add_middleware(RedisRateLimitMiddleware, ...)
    app.add_middleware(BlogVisitStatsMiddleware)
    app.add_middleware(CORSMiddleware, ...)
    app.add_middleware(GZipMiddleware, ...)
    
    # 注册路由
    app.include_router(auth.router)
    app.include_router(fund.router)
    app.include_router(market.router)
    # ... 其他路由
    
    # 全局异常处理
    @app.exception_handler(Exception)
    async def _global_exception_handler(...)
```

> **Gin vs FastAPI 的关键差异**：
> - FastAPI 用装饰器 `@router.get("/path")` 注册路由，Gin 在 router 文件里手动 `r.GET("/path", handler)`
> - FastAPI 路由文件同时写处理函数，Go 把路由注册（router）和处理函数（handler）分离

### 1.7 验证：跑起来

```bash
go run cmd/server/main.go
# 应该看到：
# INFO 数据库已连接 max_open=20
# INFO Redis 已连接 addr=localhost:6379
# INFO 应用已装配 port=8080
# INFO 服务启动 addr=:8080
```

**如果数据库连不上怎么办？**
先注释掉 `db.InitGORM` 和 `db.InitRedis` 的调用，先让 Gin Engine 能跑起来，
后面把数据库配通了再打开。建议：

```go
// 开发初期先跳过数据库
// db.InitGORM(cfg.DB)
// db.InitRedis(cfg.Redis)
// 等后面需要真正读写数据时再打开
```

---

## Step 2：Router 骨架

> **这步做什么**：一口气创建 10 个路由文件，每个只写 // TODO 占位。
> 对应 Python routers/ 目录下的 10 个文件。

### 2.1 创建 Handlers 聚合结构

先建 handler 的"总线"——后面所有 handler 都往这里装：

```go
// internal/handler/handler.go
package handler

// Handlers 聚合所有 handler，供 router 调用
type Handlers struct {
    Health       *HealthHandler
    Auth         *AuthHandler
    Fund         *FundHandler
    Market       *MarketHandler
    User         *UserHandler
    Admin        *AdminHandler
    Version      *VersionHandler
    AgentRequest *AgentRequestHandler
    Jcti         *JctiHandler
    Public       *PublicHandler
}

func NewHandlers() *Handlers {
    return &Handlers{
        Health: NewHealthHandler(),
        // 其他先让它们 nil，后面逐步补上
    }
}
```

### 2.2 创建 router 入口

```go
// internal/router/router.go
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, h *handler.Handlers) {
    api := r.Group("/api")
    
    // 10 个子路由，每个对应 Python 的 routers/*.py
    registerHealth(api, h.Health)       // routers/health.py
    registerAuth(api, h.Auth)            // routers/auth.py
    registerFund(api, h.Fund)            // routers/fund.py
    registerMarket(api, h.Market)        // routers/market.py
    registerUser(api, h.User)            // routers/user.py
    registerAdmin(api, h.Admin)          // routers/admin.py
    registerVersion(api, h.Version)      // routers/version.py
    registerAgentRequest(api, h.AgentRequest) // routers/agent_request.py
    registerJcti(api, h.Jcti)            // routers/jcti.py
    registerPublic(api, h.Public)        // routers/public.py
}
```

### 2.3 先写一个完整的 router（health）

```go
// internal/router/health.go
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

func registerHealth(rg *gin.RouterGroup, h *handler.HealthHandler) {
    // Python routers/health.py 里定义了两个路由：
    // GET /api/health
    // GET /api/health/redis
    
    g := rg.Group("/health")
    g.GET("", h.Ping)          // GET /api/health
    g.GET("/redis", h.Redis)   // GET /api/health/redis
}
```

### 2.4 其他路由全部占位

```go
// internal/router/auth.go
package router

import (
    "huahua-service/internal/handler"
    "github.com/gin-gonic/gin"
)

func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler) {
    g := rg.Group("/auth")
    // TODO: 后面逐步打开注册行
    // g.POST("/send-code",        h.SendEmailCode)
    // g.POST("/register",         h.Register)
    // g.POST("/token",            h.Login)
    // g.POST("/login-by-email",   h.LoginByEmail)
    // g.POST("/logout",           h.Logout)
    // g.GET("/me",                h.Me)
    // ...
}
```

其余 8 个文件同理，先都建好，函数内部的注册行全部 `// TODO` 注释掉。

### 2.5 在 app.go 里启用路由注册

回到 `internal/app/app.go`，加上：

```go
import "huahua-service/internal/router"

func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    
    // 创建 handler 实例
    h := handler.NewHandlers()
    // 注册路由
    router.Register(r, h)
    
    return r
}
```

现在编译会报错，因为很多 handler 还没有具体实现（返回 nil）。
别担心——Step 3 开始一个个补齐。

---

## Step 3：不依赖外部 API 的 handler 先做

> **这步做什么**：先写不需要调外部接口的模块——health、version、auth（注册登录）。
> 这些"自成一体"，写完就能测通，建立信心。

**对应 Python 文件**：`routers/health.py`、`routers/version.py`、`routers/auth.py`

### 3.1 Health handler —— 最简单的例子

**Python 原版**：[routers/health.py](Python backend → routers/health.py)：

```python
@router.get("/health")
async def health(response: Response):
    db = await run_in_threadpool(_check_db)
    redis_status = await _check_redis()
    components = {"db": db, "redis": redis_status}
    has_error = any(v == "error" for v in components.values())
    status = "error" if has_error else ("degraded" if "fallback" in components.values() else "ok")
    if has_error:
        response.status_code = 503
    return {"status": status, "components": components}
```

**Go 实现：**

```go
// internal/handler/health.go
package handler

import (
    "context"
    "net/http"
    "time"

    "huahua-service/internal/db"

    "github.com/gin-gonic/gin"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
    return &HealthHandler{}
}

// Ping 对应 Python health() —— 检查 db 和 redis
func (h *HealthHandler) Ping(c *gin.Context) {
    // 检查数据库
    dbStatus := "ok"
    if db.GORM != nil {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
        defer cancel()
        var result int
        if err := db.GORM.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
            dbStatus = "error"
        }
    }

    // 检查 Redis
    redisStatus := "ok"
    if db.Redis != nil {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
        defer cancel()
        if err := db.Redis.Ping(ctx).Err(); err != nil {
            redisStatus = "error"
        }
    }

    components := gin.H{"db": dbStatus, "redis": redisStatus}

    status := "ok"
    statusCode := http.StatusOK
    if dbStatus == "error" || redisStatus == "error" {
        status = "error"
        statusCode = http.StatusServiceUnavailable  // 503
    } else if dbStatus == "fallback" || redisStatus == "fallback" {
        status = "degraded"
    }

    c.JSON(statusCode, gin.H{
        "status":     status,
        "components": components,
    })
}

func (h *HealthHandler) Redis(c *gin.Context) {
    // 对应 GET /api/health/redis —— 返回 Redis 详情
    c.JSON(http.StatusOK, gin.H{"status": "Redis 诊断（待实现）"})
}
```

**救：在 app.go 的 New 函数里把 router 注册调用加上**

```go
// internal/app/app.go
func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    
    h := handler.NewHandlers()
    router.Register(r, h)
    
    return r
}
```

**验证**：

```bash
go run cmd/server/main.go
# 另一个终端执行：
curl http://localhost:8080/api/health
# 期望输出（如果没连数据库，会显示 error）：
# {"components":{"db":"error","redis":"ok"},"status":"error"}
```

### 3.2 Auth handler —— 注册登录全套

这是最复杂的一个模块，**拆成子步骤完成**。

#### 3.2a Schema —— 请求和响应结构

**Python 原版**：[schemas.py](Python backend → schemas.py) 的 `UserCreate`、`TokenResponse` 等

```go
// internal/schema/auth.go
package schema

// RegisterRequest 对应 Python schemas.py 的 UserCreate
type RegisterRequest struct {
    Username        string `json:"username" binding:"required,min=4,max=20"`
    Password        string `json:"password" binding:"required,min=8"`
    Email           string `json:"email" binding:"required,email"`
    VerificationCode string `json:"verification_code" binding:"required"`
    Nickname        string `json:"nickname" binding:"max=16"`
    InviteCode      string `json:"invite_code"`
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

> `binding:"required,min=4,max=20"` 是 Gin 内置的校验标签。
> Gin 底层用 `validator/v10`，跟 Pydantic 的 `Field(..., pattern=...)` 功能类似。

#### 3.2b Security —— JWT 工具

**Python 原版**：[security.py](Python backend → security.py) 中的 JWT 部分 + `routers/auth.py` 的 `create_access_token`

```go
// internal/security/jwt.go
package security

import (
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
    Secret   string
    ExpireIn time.Duration // 如 7*24*time.Hour
}

type Claims struct {
    Sub string `json:"sub"` // username 作为 subject
    jwt.RegisteredClaims
}

// GenerateToken 生成 JWT token
// Python 对应：jwt_encode({"sub": username}, SECRET_KEY, algorithm="HS256")
func GenerateToken(cfg JWTConfig, username string) (string, error) {
    claims := Claims{
        Sub: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.ExpireIn)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(cfg.Secret))
}

// ParseToken 验证并解析 JWT
func ParseToken(secret, tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, 
        func(token *jwt.Token) (interface{}, error) {
            return []byte(secret), nil
        })
    if err != nil {
        return nil, err
    }
    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, jwt.ErrSignatureInvalid
    }
    return claims, nil
}
```

**对照 Python 的 JWT 部分**：

```python
# Python routers/auth.py 中的 JWT 函数
def create_access_token(data: dict):
    to_encode = data.copy()
    to_encode.update({
        "exp": datetime.now(timezone.utc).replace(tzinfo=None) + 
               timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    })
    return jwt_encode(to_encode, settings.SECRET_KEY, algorithm=settings.ALGORITHM)
```

#### 3.2c Bcrypt 密码哈希

Python 用 `passlib` 库，Go 用 `golang.org/x/crypto/bcrypt`：

```go
// internal/security/bcrypt.go
package security

import "golang.org/x/crypto/bcrypt"

// HashPassword 对应 Python get_password_hash()
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

// CheckPassword 对应 Python verify_password()
func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**对照 Python**：

```python
# Python routers/auth.py
from passlib.context import CryptContext
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")

def verify_password(plain, hashed): return pwd_context.verify(plain, hashed)
def get_password_hash(password): return pwd_context.hash(password)
```

**如何安装 bcrypt 依赖**：

```bash
go get golang.org/x/crypto/bcrypt
```

#### 3.2d Middleware —— Auth 中间件

对比 Python 的 [`get_current_user` 依赖](Python backend → routers/auth.py 行 200+)：

```python
# Python —— FastAPI 的 Depends
async def get_current_user(
    request: Request,
    token_header: Optional[str] = Depends(oauth2_scheme),
    db: Session = Depends(get_db)
):
    # 支持 AgentToken 和 Bearer JWT 两种鉴权
    auth_header = request.headers.get("Authorization", "")
    
    if auth_header.startswith("AgentToken "):
        # AgentToken 模式——查 agent_tokens 表
        ...
        return user
    
    # Bearer JWT 模式
    token = token_header or request.cookies.get("access_token")
    ...
    return user
```

**Go 中间件**——每次请求进入时，先过中间件验证身份，把用户信息存到 Context：

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

// AuthRequired 返回一个 Gin 中间件函数
// 对应 Python get_current_user Depends
func AuthRequired(jwtCfg security.JWTConfig, userRepo repository.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        
        if strings.HasPrefix(authHeader, "AgentToken ") {
            // AgentToken 模式——查数据库验证
            // 简单起见，这一步先返回 401
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "AgentToken 未实现"})
            return
        }
        
        // Bearer JWT 模式
        token := ""
        if strings.HasPrefix(authHeader, "Bearer ") {
            token = authHeader[7:]  // 去掉 "Bearer "
        } else {
            // 尝试从 cookie 读取
            token, _ = c.Cookie("access_token")
        }
        
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "未登录"})
            return
        }
        
        // 解析 JWT
        claims, err := security.ParseToken(jwtCfg.Secret, token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Token 无效或已过期"})
            return
        }
        
        // 从数据库查用户
        user, err := userRepo.GetByUsername(c.Request.Context(), claims.Sub)
        if err != nil || user == nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "用户不存在"})
            return
        }
        
        // 把用户信息存到 context——后续 handler 通过 c.Get("user") 取
        c.Set("user", user)
        c.Next()  // 继续执行后续的 handler
    }
}
```

#### 3.2e Auth service —— 真正的业务逻辑

Python 的用户注册是写在 `routers/auth.py` 里的（~150 行），Go 必须抽到 service 层：

```go
// internal/service/auth/auth_service.go
package auth

import (
    "context"
    "errors"

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

// Register 用户注册
// 对照 Python routers/auth.py 的 register() 函数——去掉数据库操作后就是纯业务逻辑
func (s *AuthService) Register(ctx context.Context, req schema.RegisterRequest) (*model.User, string, error) {
    // 1. 校验用户名是否已存在
    existing, err := s.userRepo.GetByUsername(ctx, req.Username)
    if err != nil {
        return nil, "", err
    }
    if existing != nil {
        return nil, "", errors.New("用户名已被注册")
    }

    // 2. 校验邮箱是否已存在
    existingEmail, err := s.userRepo.GetByEmail(ctx, req.Email)
    if err != nil {
        return nil, "", err
    }
    if existingEmail != nil {
        return nil, "", errors.New("邮箱已被注册")
    }

    // 3. 加密密码
    hashedPassword, err := security.HashPassword(req.Password)
    if err != nil {
        return nil, "", err
    }

    // 4. 创建用户
    user := &model.User{
        Username:       req.Username,
        Email:          req.Email,
        HashedPassword: hashedPassword,
        Nickname:       req.Nickname,
        Avatar:         "https://api.dicebear.com/7.x/adventurer/svg",  // Python 默认头像
        UID:            generateUID(),  // 8位数字 UID
    }

    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, "", err
    }

    // 5. 生成 JWT
    token, err := security.GenerateToken(s.jwtCfg, user.Username)
    if err != nil {
        return nil, "", err
    }

    return user, token, nil
}

// Login 对应 Python 的 login() 函数
func (s *AuthService) Login(ctx context.Context, username, password string) (*model.User, string, error) {
    user, err := s.userRepo.GetByUsername(ctx, username)
    if err != nil {
        return nil, "", err
    }
    if user == nil {
        return nil, "", errors.New("用户名或密码错误")
    }

    if !security.CheckPassword(password, user.HashedPassword) {
        return nil, "", errors.New("用户名或密码错误")
    }

    token, err := security.GenerateToken(s.jwtCfg, user.Username)
    if err != nil {
        return nil, "", err
    }

    return user, token, nil
}

// generateUID 生成 8 位数字 UID（对应 Python models.py 的 generate_uid）
func generateUID() string {
    // 简单实现——可以用 crypto/rand 改进
    return fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
}
```

> **对照 Python**：Python 的注册逻辑在 `routers/auth.py` 的 `register()` 函数里（~120 行），
> 包含了校验验证码、检查邮箱黑名单、创建用户、发放邀请奖励等一系列逻辑。
> Go service 层做的事情一模一样，只是把 `db.query(User).filter(...).first()` 换成了 `s.userRepo.GetByUsername(ctx, ...)`。

#### 3.2f Auth handler —— 接 HTTP 请求

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
        AccessToken: token,
        TokenType:   "bearer",
        UserID:      user.ID,
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
        AccessToken: token,
        TokenType:   "bearer",
        UserID:      user.ID,
    })
}
```

#### 3.2g 串联：在 handler.NewHandlers 里注入依赖

回到 `handler/handler.go`，因为 AuthHandler 需要 AuthService，
所以 NewHandlers 需要接收参数：

```go
// internal/handler/handler.go
package handler

import "huahua-service/internal/service/auth"

type Handlers struct {
    Health       *HealthHandler
    Auth         *AuthHandler  // 现在不再是 nil 了
    // ... 其他
}

// NewHandlers 现在需要注入依赖
func NewHandlers(authService *auth.AuthService) *Handlers {
    return &Handlers{
        Health: NewHealthHandler(),
        Auth:   NewAuthHandler(authService),
    }
}
```

然后在 `app.go` 里创建依赖链：

```go
// internal/app/app.go（概览）
func New(cfg *config.Config) *gin.Engine {
    // 1. 创建 repository
    userRepo := repository.NewUserRepository(db.GORM)
    
    // 2. 创建 service
    jwtCfg := security.JWTConfig{
        Secret:   cfg.SecretKey,
        ExpireIn: 7 * 24 * time.Hour,
    }
    authService := auth.NewAuthService(db.GORM, userRepo, jwtCfg)
    
    // 3. 创建 handler
    h := handler.NewHandlers(authService)
    
    // 4. 注册路由
    r := gin.New()
    r.Use(gin.Recovery())
    router.Register(r, h)
    
    return r
}
```

#### 3.2h 打开路由注册

回到 `internal/router/auth.go`，把 TODO 替换为：

```go
func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler) {
    g := rg.Group("/auth")
    g.POST("/register", h.Register)
    g.POST("/token", h.Login)
    // 后续再加：/me, /send-code, /logout ...
}
```

**验证**：

```bash
go run cmd/server/main.go &
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123","email":"test@test.com","verification_code":"000000"}'
# 期望：
# {"access_token":"eyJ...","token_type":"bearer","user_id":1}
```

---

## Step 4：核心数据源封装 (akshare)

> **这步做什么**：写 Go 的 HTTP 客户端，替代 Python 的 akshare 库。
> akshare 本质是包装了东方财富、腾讯财经等网站的 HTTP API。

**Python 原版**：[services/akshare_service.py](Python backend → services/akshare_service.py)（最长文件，~400+ 行）

### 4.1 理解 akshare 是干什么的

akshare 是一个 Python 财经数据库，本质就是发 HTTP 请求到各个财经网站。
比如获取基金实时估值的 URL：`https://fundgz.1234567.com.cn/js/000001.js`
返回的是 `jsonpgz({"fundcode":"000001","gsz":1.234,...});` 格式。

### 4.2 实现 Go HTTP 客户端

```go
// internal/service/akshare/client.go
package akshare

import (
    "fmt"
    "io"
    "net/http"
    "time"
)

// Client akshare HTTP 客户端
// Python 里用 httpx.AsyncClient，Go 用 net/http.Client
type Client struct {
    httpClient *http.Client
}

func NewClient() *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: 10 * time.Second,
        },
    }
}

// get 通用 GET 请求（小写开头，包内使用）
func (c *Client) get(url string) ([]byte, error) {
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
    req.Header.Set("Referer", "https://finance.eastmoney.com/")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("请求失败: %w", err)
    }
    defer resp.Body.Close()  // 重要！不关会造成连接泄漏

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }

    return io.ReadAll(resp.Body)
}
```

### 4.3 实现基金实时估值接口

**Python 原版**：akshare_service.py 中通过 httpx 请求东方财富 fundgz：

```python
# Python akshare_service.py（约）
async def fetch_fund_estimate(code: str):
    url = f"https://fundgz.1234567.com.cn/js/{code}.js"
    async with httpx.AsyncClient() as client:
        resp = await client.get(url, headers={"User-Agent": "..."})
        data = resp.text
        # 返回 jsonpgz({"fundcode":"000001",...}) 格式
        json_str = data[8:-2]  # 去掉 jsonpgz( 和 );
        return json.loads(json_str)
```

**Go 实现**：

```go
// internal/service/akshare/estimate.go
package akshare

import (
    "encoding/json"
    "fmt"
    "strings"
)

// FundEstimate 实时估值——对应东方财富 fundgz 接口返回
type FundEstimate struct {
    FundCode        string  `json:"fundcode"`
    Name            string  `json:"name"`
    Estimate        float64 `json:"gsz"`        // 实时估值
    EstimatePercent float64 `json:"gszzl"`      // 涨跌幅 %
    LastNav         float64 `json:"dwjz"`        // 昨日净值
    EstimateTime    string  `json:"gztime"`      // 估值时间
}

// GetFundEstimate 获取指定基金的实时估值
// fundCode 如 "000001"
func (c *Client) GetFundEstimate(fundCode string) (*FundEstimate, error) {
    url := fmt.Sprintf("https://fundgz.1234567.com.cn/js/%s.js", fundCode)
    body, err := c.get(url)
    if err != nil {
        return nil, fmt.Errorf("获取估值失败 %s: %w", fundCode, err)
    }

    // 去掉 "jsonpgz(" 前缀和 ");" 后缀
    text := string(body)
    if !strings.HasPrefix(text, "jsonpgz(") || !strings.HasSuffix(text, ");") {
        return nil, fmt.Errorf("unexpected response format for %s", fundCode)
    }
    jsonStr := text[8 : len(text)-2]

    var est FundEstimate
    if err := json.Unmarshal([]byte(jsonStr), &est); err != nil {
        return nil, fmt.Errorf("解析估值失败 %s: %w", fundCode, err)
    }

    return &est, nil
}
```

### 4.4 写单元测试 mock 上游

```go
// internal/service/akshare/estimate_test.go
package akshare

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetFundEstimate(t *testing.T) {
    // 创建 mock HTTP 服务器，模拟东方财富返回
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`jsonpgz({"fundcode":"000001","name":"测试基金","gsz":1.234,"gszzl":0.56,"dwjz":1.227,"gztime":"2024-01-15 15:00"});`))
    }))
    defer mockServer.Close()

    // 创建一个 client 但指向 mockServer 的 URL
    client := &Client{
        httpClient: &http.Client{},
    }

    // 测试——但这里 get 的 URL 写死在函数里了
    // 所以更好的做法是把 fundgz URL 变成可配置的
    t.Log("mock 服务器已启动:", mockServer.URL)
    // 见下方改进：把 URL 作为参数传入 GetFundEstimate
}
```

> **测试技巧**：写 HTTP 客户端时，把 base URL 设计成可配置的，这样测试时可以指向 mock 服务器。

改进的客户端版本：

```go
type Client struct {
    httpClient  *http.Client
    FundGZURL   string  // 可配置的基金估值 URL 模板，默认 "https://fundgz.1234567.com.cn/js/%s.js"
}
```

```bash
# 运行测试
go test ./internal/service/akshare/
```

---

## Step 5：基础业务 (Fund Service)

> **这步做什么**：把 akshare 的数据 + 数据库信息组合成业务对象返回给前端。
> 对应 Python 的 `services/fund_service.py` + `routers/fund.py`

### 5.1 Fund Service

**Python 原版**：[services/fund_service.py](Python backend → services/fund_service.py) ~300 行

```go
// internal/service/fund/fund_service.go
package fund

import (
    "context"
    "fmt"

    "huahua-service/internal/model"
    "huahua-service/internal/repository"
    "huahua-service/internal/service/akshare"

    "github.com/shopspring/decimal"  // 用 decimal 避免 float64 精度问题
)

type FundService struct {
    akshareClient *akshare.Client
    fundRepo      repository.FundBasicInfoRepository
    navRepo       repository.FundNavRepository
}

func NewFundService(akshareClient *akshare.Client, fundRepo repository.FundBasicInfoRepository, navRepo repository.FundNavRepository) *FundService {
    return &FundService{
        akshareClient: akshareClient,
        fundRepo:      fundRepo,
        navRepo:       navRepo,
    }
}

// GetEstimate 获取单只基金实时估值（含数据库补全信息）
// 对应 Python FundService.get_estimates() 方法
func (s *FundService) GetEstimate(ctx context.Context, code string) (*FundEstimateResult, error) {
    // 1. 从 akshare 拿实时数据
    raw, err := s.akshareClient.GetFundEstimate(code)
    if err != nil {
        return nil, fmt.Errorf("获取基金 %s 估值失败: %w", code, err)
    }

    // 2. 从数据库查基金基本信息（名称、类型等）
    info, _ := s.fundRepo.GetByCode(ctx, code)
    
    result := &FundEstimateResult{
        FundCode: raw.FundCode,
        Estimate: decimal.NewFromFloat(raw.Estimate),
        Percent:  decimal.NewFromFloat(raw.EstimatePercent),
        Time:     raw.EstimateTime,
    }
    
    if info != nil {
        result.Name = info.Name
        result.FundType = info.Type
    } else {
        result.Name = raw.Name
    }

    return result, nil
}

// BatchGetEstimates 批量获取估值
// 对应 Python FundService.get_estimates(codes) 列表版本
func (s *FundService) BatchGetEstimates(ctx context.Context, codes []string) ([]*FundEstimateResult, error) {
    results := make([]*FundEstimateResult, 0, len(codes))
    for _, code := range codes {
        est, err := s.GetEstimate(ctx, code)
        if err != nil {
            // 单个失败不影响其他——跳过
            continue
        }
        results = append(results, est)
    }
    return results, nil
}

type FundEstimateResult struct {
    FundCode string          `json:"fund_code"`
    Name     string          `json:"name"`
    FundType string          `json:"fund_type,omitempty"`
    Estimate decimal.Decimal `json:"estimate"`
    Percent  decimal.Decimal `json:"percent"`
    Time     string          `json:"time"`
}
```

### 5.2 Fund Repository

```go
// internal/repository/fund_basic_info_repo.go
package repository

import (
    "context"
    
    "huahua-service/internal/model"
    
    "gorm.io/gorm"
)

type FundBasicInfoRepository interface {
    GetByCode(ctx context.Context, code string) (*model.FundBasicInfo, error)
    FindAll(ctx context.Context) ([]model.FundBasicInfo, error)
}

type fundBasicInfoRepo struct {
    db *gorm.DB
}

func NewFundBasicInfoRepository(db *gorm.DB) FundBasicInfoRepository {
    return &fundBasicInfoRepo{db: db}
}

func (r *fundBasicInfoRepo) GetByCode(ctx context.Context, code string) (*model.FundBasicInfo, error) {
    var info model.FundBasicInfo
    err := r.db.WithContext(ctx).Where("code = ?", code).First(&info).Error
    if err != nil {
        return nil, err
    }
    return &info, nil
}

func (r *fundBasicInfoRepo) FindAll(ctx context.Context) ([]model.FundBasicInfo, error) {
    var funds []model.FundBasicInfo
    err := r.db.WithContext(ctx).Find(&funds).Error
    return funds, err
}
```

### 5.3 Fund Handler

```go
// internal/handler/fund.go
package handler

import (
    "net/http"

    "huahua-service/internal/service/fund"

    "github.com/gin-gonic/gin"
)

type FundHandler struct {
    fundService *fund.FundService
}

func NewFundHandler(fundService *fund.FundService) *FundHandler {
    return &FundHandler{fundService: fundService}
}

// BatchEstimate 对应 Python fund.py 的 api_estimate()
// POST /api/estimate/batch  {codes: ["000001","000002"]}
func (h *FundHandler) BatchEstimate(c *gin.Context) {
    var req struct {
        Codes []string `json:"codes" binding:"required,max=50"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
        return
    }

    results, err := h.fundService.BatchGetEstimates(c.Request.Context(), req.Codes)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "data":      results,
        "truncated": false,
        "limit":     50,
    })
}
```

> **对照 Python 代码**：
> ```python
> # routers/fund.py 中的 api_estimate
> @router.post("/estimate/batch")
> async def api_estimate(payload: EstimateRequest, request: Request):
>     codes = payload.codes
>     result = await FundService.get_estimates(codes)
>     return {"data": result, "truncated": False, "limit": 50}
> ```
> 是不是一模一样？只是 `ShouldBindJSON` 替代了 `EstimateRequest`，`c.JSON` 替代了 `return`。

### 5.4 打开路由

```go
// internal/router/fund.go
func registerFund(rg *gin.RouterGroup, h *handler.FundHandler) {
    g := rg.Group("")
    g.POST("/estimate/batch", h.BatchEstimate)
    g.GET("/fund/:code/estimate", h.SingleEstimate)  // 逐步实现
    // ...
}
```

---

## Step 6：handler/fund.go / market.go 接入

到这步你已经掌握了"写一个完整的 handler"的所有套路。这步就是**重复这个模式**，把 Step 4~5 写的
全部接入路由。

**检查清单**：
- [ ] `handler/fund.go` 中 `BatchEstimate` 已实现并通过 curl 验证
- [ ] `handler/fund.go` 中 `SingleEstimate`（GET /api/fund/:code/estimate）已实现
- [ ] `handler/fund.go` 中 `History`（GET /api/history/:code）已实现
- [ ] `handler/market.go` 中各接口已实现（指数行情、板块排行等）
- [ ] 所有 repository 方法已补全
- [ ] 所有 router 文件中的 TODO 已打开

**验证**：

```bash
# 批量估值
curl -X POST http://localhost:8080/api/estimate/batch \
  -H "Content-Type: application/json" \
  -d '{"codes":["000001","110011"]}'
# 应返回包含估值数据的数组
```

---

## Step 7：AI 与用户数据

> **这步做什么**：实现 AI 调用（OpenRouter + Gemini 降级）和用户数据同步。

**Python 原版**：[services/ai_service.py](Python backend → services/ai_service.py)

### 7.1 Go 里发 HTTP POST 请求调用 AI

```go
// internal/service/ai/ai_service.go
package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type AIService struct {
    openRouterKey string
    geminiKey     string
    httpClient    *http.Client
}

func NewAIService(openRouterKey, geminiKey string) *AIService {
    return &AIService{
        httpClient: &http.Client{
            Timeout: 60 * time.Second,  // AI 响应慢，60s 超时
        },
        openRouterKey: openRouterKey,
        geminiKey:     geminiKey,
    }
}

// Chat 发送对话给 AI（先试 OpenRouter，失败则降级到 Gemini）
// 对应 Python ai_service.py 的 chat_completion 方法
func (s *AIService) Chat(ctx context.Context, messages []Message) (string, error) {
    // 先试 OpenRouter
    result, err := s.chatWithOpenRouter(ctx, messages)
    if err == nil {
        return result, nil
    }
    
    // 降级到 Gemini
    return s.chatWithGemini(ctx, messages)
}

type Message struct {
    Role    string `json:"role"`    // "user" 或 "assistant"
    Content string `json:"content"`
}

type chatRequest struct {
    Model    string      `json:"model"`
    Messages []Message    `json:"messages"`
}

type chatResponse struct {
    Choices []struct {
        Message Message `json:"message"`
    } `json:"choices"`
}

func (s *AIService) chatWithOpenRouter(ctx context.Context, messages []Message) (string, error) {
    if s.openRouterKey == "" {
        return "", fmt.Errorf("OpenRouter key 未配置")
    }

    body := chatRequest{
        Model:    "openai/gpt-4o",
        Messages: messages,
    }
    jsonBody, _ := json.Marshal(body)

    req, err := http.NewRequestWithContext(ctx, "POST",
        "https://openrouter.ai/api/v1/chat/completions",
        bytes.NewReader(jsonBody))
    if err != nil {
        return "", err
    }

    req.Header.Set("Authorization", "Bearer "+s.openRouterKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := s.httpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("OpenRouter 请求失败: %w", err)
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    var chatResp chatResponse
    if err := json.Unmarshal(respBody, &chatResp); err != nil {
        return "", err
    }

    if len(chatResp.Choices) == 0 {
        return "", fmt.Errorf("AI 未返回结果")
    }

    return chatResp.Choices[0].Message.Content, nil
}

func (s *AIService) chatWithGemini(ctx context.Context, messages []Message) (string, error) {
    // Gemini API 格式略不同，但模式一样：构造请求 → 发 POST → 解析响应
    // 留作练习
    return "", fmt.Errorf("Gemini 暂未实现")
}
```

**对照 Python**：

```python
# Python ai_service.py（简化）
class AIService:
    async def chat_completion(self, messages, model=None):
        model = model or settings.OPENROUTER_MODEL
        # 1. 尝试 OpenRouter
        try:
            return await self._call_openrouter(messages, model)
        except Exception:
            # 2. 失败降级 Gemini
            return await self._call_gemini(messages)
    
    async def _call_openrouter(self, messages, model):
        headers = {
            "Authorization": f"Bearer {settings.OPENROUTER_API_KEY}",
            "Content-Type": "application/json",
        }
        body = {"model": model, "messages": [m.dict() for m in messages]}
        async with httpx.AsyncClient() as client:
            resp = await client.post(
                f"{settings.OPENROUTER_BASE_URL}/chat/completions",
                headers=headers, json=body,
                timeout=httpx.Timeout(settings.OPENROUTER_READ_TIMEOUT),
            )
            return resp.json()["choices"][0]["message"]["content"]
```

> **注意**：Go 的 `http.Client` 不支持自动超时重试，Python 的 `httpx` 也不支持。
> 这里用 `defer resp.Body.Close()` 确保连接关闭（Go 新手最容易忘这行）。

---

## Step 8：夜估 (Night Estimate)

> **这步做什么**：实现"夜间净值校准"——用当日净值平滑更新历史估值。

**Python 原版**：[services/calibration_service.py](Python backend → services/calibration_service.py) + [night_estimate_service.py](Python backend → services/night_estimate_service.py)

### 8.1 业务理解

夜估公式（简化）：
```
校准后估值 = 旧估值 × λ + 当天净值 × (1 - λ)
其中 λ = 0.85（平滑系数）
```

### 8.2 实现

```go
// internal/service/calibration/calibration_service.go
package calibration

import (
    "context"
    "time"

    "huahua-service/internal/repository"
)

const (
    Lambda = 0.85  // Python 原版 λ=0.85
)

type CalibrationService struct {
    calibRepo repository.CalibrationRepository
    fundRepo  repository.FundBasicInfoRepository
    navRepo   repository.FundNavRepository
}

// RunCalibration 执行全部基金的夜估校准
// 对应 Python calibration_service.py 的 run_calibration()
func (s *CalibrationService) RunCalibration(ctx context.Context, date time.Time) error {
    // 获取所有基金
    funds, err := s.fundRepo.FindAll(ctx)
    if err != nil {
        return err
    }

    for _, fund := range funds {
        // 取当天净值
        nav, err := s.navRepo.GetByDate(ctx, fund.Code, date)
        if err != nil {
            continue  // 净值还没出来，跳过
        }

        // 取上次校准估值
        lastEst, _ := s.calibRepo.GetLatest(ctx, fund.Code)
        
        var adjusted float64
        if lastEst == nil {
            // 冷启动：直接用当天净值
            adjusted = nav.Nav
        } else {
            // 平滑公式：旧估值 × λ + 新净值 × (1-λ)
            adjusted = lastEst.Value*Lambda + nav.Nav*(1-Lambda)
        }

        // 保存校准结果
        if err := s.calibRepo.Save(ctx, fund.Code, date, adjusted); err != nil {
            return err
        }
    }

    return nil
}
```

**对照 Python**：

```python
# Python calibration_service.py
class CalibrationService:
    @staticmethod
    def run_calibration(session, date: date) -> None:
        funds = session.query(FundBasicInfo).all()
        for fund in funds:
            nav = session.query(FundNav).filter(
                FundNav.fund_code == fund.code,
                FundNav.date == date.isoformat()
            ).first()
            if not nav:
                continue
            
            last = session.query(CalibrationData).filter(
                CalibrationData.fund_code == fund.code
            ).order_by(CalibrationData.date.desc()).first()
            
            if last is None:
                adjusted = float(nav.nav)
            else:
                adjusted = last.adjusted_value * 0.85 + float(nav.nav) * 0.15
            
            session.add(CalibrationData(
                fund_code=fund.code,
                date=date,
                adjusted_value=adjusted,
            ))
        session.commit()
```

---

## Step 9：后台任务

> **这步做什么**：写一个后台循环，定时刷新基金数据、执行夜估。

**Python 原版**：[tasks/background_refresh.py](Python backend → tasks/background_refresh.py)

### 9.1 Go 实现后台任务

```go
// internal/task/background.go
package task

import (
    "context"
    "log/slog"
    "time"

    "huahua-service/internal/service/calibration"
)

type TaskRunner struct {
    calibService *calibration.CalibrationService
    interval     time.Duration
}

func NewTaskRunner(calibService *calibration.CalibrationService) *TaskRunner {
    return &TaskRunner{
        calibService: calibService,
        interval:     5 * time.Minute,  // 每 5 分钟跑一次
    }
}

// Run 启动后台循环（在 goroutine 里调用）
// 对应 Python background_refresh.py 的 background_refresh_loop()
func (r *TaskRunner) Run(ctx context.Context) {
    slog.Info("后台任务启动")

    ticker := time.NewTicker(r.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            r.refresh(ctx)
        case <-ctx.Done():
            slog.Info("后台任务停止")
            return
        }
    }
}

func (r *TaskRunner) refresh(ctx context.Context) {
    now := time.Now()
    
    // 只在交易日执行
    if !isTradingDay(now) {
        return
    }

    // 22:00 左右执行夜估
    if now.Hour() == 22 && now.Minute() < 10 {
        slog.Info("执行夜估校准...")
        if err := r.calibService.RunCalibration(ctx, now); err != nil {
            slog.Error("夜估失败", "error", err)
        }
    }

    // 盘中刷新基金排行、板块数据（略）
}

func isTradingDay(t time.Time) bool {
    // 简单判断：周末不是交易日
    if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
        return false
    }
    // 完整实现需要查询交易日历数据库
    return true
}
```

### 9.2 嵌入 main.go

```go
// 在 cmd/server/main.go 中
bgCtx, bgCancel := context.WithCancel(context.Background())
defer bgCancel()

taskRunner := task.NewTaskRunner(calibService)
go taskRunner.Run(bgCtx)

// 收到 SIGTERM 时先停 HTTP 再停后台
<-quit
srv.Shutdown(ctx)
bgCancel()  // 通知后台任务停止
```

**对照 Python**：

```python
# Python lifespan.py —— 用 asyncio 创建后台任务
@asynccontextmanager
async def lifespan(app: FastAPI):
    # 启动
    bg_task = asyncio.create_task(background_refresh_loop())
    yield
    # 停止
    bg_task.cancel()
    await asyncio.gather(bg_task, return_exceptions=True)
```

---

## Step 10：Agent 相关

> **这步做什么**：支持 `Authorization: AgentToken xxx` 鉴权，让 MCP/AI agent 访问用户数据。

**Python 原版**：[routers/auth.py](Python backend → routers/auth.py) 的 `_is_agent_token_allowed`、`get_current_user` 的 AgentToken 分支

AgentToken 的核心思路：
1. 前端生成一个随机 token，SHA256 哈希后存数据库（明文仅展示一次）
2. 客户端请求时带 `Authorization: AgentToken <raw_token>`
3. 服务端取到 raw_token，SHA256 后去数据库查
4. 查到的记录里有 scope（权限范围），检查当前路径是否允许

```go
// internal/middleware/auth.go 中补充 AgentToken 分支
func handleAgentToken(c *gin.Context, authHeader string, agentTokenRepo repository.AgentTokenRepository) {
    rawToken := strings.TrimPrefix(authHeader, "AgentToken ")
    tokenHash := sha256Hash(rawToken)  // SHA256(raw_token)

    agentTok, err := agentTokenRepo.GetByHash(c.Request.Context(), tokenHash)
    if err != nil || agentTok == nil {
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "无效的 Agent Token"})
        return
    }

    // 检查 scope 是否允许当前路径
    if !isAgentTokenAllowed(c.Request.Method, c.Request.URL.Path, agentTok.Scope) {
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "Agent Token 无权访问"})
        return
    }

    // 更新 last_used
    agentTokenRepo.UpdateLastUsed(c.Request.Context(), agentTok.ID, c.ClientIP())

    // 查对应用户并存入 context
    userRepo := getFromContext(c, "userRepo")  // 需要设计如何传递依赖
    user, _ := userRepo.GetByID(c.Request.Context(), agentTok.UserID)
    c.Set("user", user)
    c.Next()
}

func sha256Hash(s string) string {
    h := sha256.Sum256([]byte(s))
    return hex.EncodeToString(h[:])
}
```

---

## Step 11：JCTI / 邀请裂变 / 截图导入

这些属于业务"尾部和特色功能"，模式完全相同：`handler → service → repository`。

### 邀请裂变关键——事务 + SELECT FOR UPDATE

这是 Go 对比 Python 最容易出错的地方。Python SQLAlchemy 用 `db.commit()` 手动提交，
GORM 用 `db.Transaction(func(tx *gorm.DB) error { ... })`：

```go
// 在 repository 里提供带事务的方法
func (r *userRepo) LockByIDForUpdateTx(ctx context.Context, tx *gorm.DB, id int64) (*model.User, error) {
    var user model.User
    err := tx.WithContext(ctx).
        Raw("SELECT * FROM users WHERE id = ? FOR UPDATE", id).
        First(&user).Error
    // 或使用 GORM 的 Set 方法
    // tx.Set("gorm:query_option", "FOR UPDATE").First(&user, id)
    return &user, err
}

// 在 service 中使用事务
func (s *InviteService) ApplyInviteReward(ctx context.Context, inviterID, newUserID int64) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 1. FOR UPDATE 锁行
        inviter, err := s.userRepo.LockByIDForUpdateTx(ctx, tx, inviterID)
        if err != nil {
            return err
        }
        
        // 2. 在锁保护下统计已奖励次数
        count, err := s.userRepo.CountInvitedByTx(ctx, tx, inviter.ID)
        if err != nil {
            return err
        }
        
        // 3. 如果没到上限，给奖励
        if count < 20 {
            inviter.VIPExpiresAt = inviter.VIPExpiresAt.Add(5 * 24 * time.Hour)
            if err := s.userRepo.UpdateTx(ctx, tx, inviter); err != nil {
                return err
            }
        }
        
        return nil  // commit
    })
}
```

**对照 Python**：

```python
# Python routers/auth.py 中的注册邀请奖励
# 用 with_for_update() 锁行
inviter_locked = db.query(User).filter(
    User.id == inviter.id
).with_for_update().first()

reward_count = db.query(User).filter(
    User.invited_by == inviter_locked.id
).count()
if reward_count < MAX_INVITE_REWARD_TIMES:
    _extend_vip(inviter_locked, INVITE_REWARD_DAYS)
```

---

## Step 12：静态文件 & SPA fallback

**Python 原版**：[core/app_factory.py](Python backend → core/app_factory.py) 的后半部分

```go
// internal/app/app.go 中追加
func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())
    
    // 托管上传目录（对应 Python /static/uploads）
    r.Static("/static/uploads", "./data/uploads")
    r.Static("/static/buy", "./data/buy")
    
    // SPA 回退
    r.NoRoute(func(c *gin.Context) {
        // API 路径返回 404 JSON
        if strings.HasPrefix(c.Request.URL.Path, "/api/") {
            c.JSON(http.StatusNotFound, gin.H{"detail": "API endpoint not found"})
            return
        }
        // 其他所有路径返回 index.html（SPA 兜底）
        c.File("./dist/index.html")
    })
    
    return r
}
```

> **注意**：Gin 的 `NoRoute` 只匹配没注册的路径。如果有注册了 `GET /api/health`，那访问 `/api/health` 就不会走到 NoRoute。

---

## Step 13：端到端联调

**这步的核心**：对比 Python 原版响应，保证**字段名和大小写完全一致**。

### 验证脚本

```bash
# 1. 健康检查
curl http://localhost:8080/api/health
# 对比 Python：{"status":"ok","components":{"db":"ok","redis":"ok"}}
# Go 必须返回完全一样的字段名和大小写

# 2. 注册
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"Pass1234","email":"test@test.com","verification_code":"000000"}'

# 3. 登录
curl -X POST http://localhost:8080/api/auth/token \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"Pass1234"}'
# 对比 Python 响应：{"token_type":"bearer","access_token":"eyJ..."}

# 4. 用 token 请求基金估值
TOKEN="eyJ..."
curl http://localhost:8080/api/estimate/batch \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"codes":["000001","110011"]}'
# 对比 Python 格式之 {"data":[...],"truncated":false,"limit":50}
```

### 字段名对齐检查清单

```go
// 每次返回结构体后，检查 json tag 是否与 Python 一致：
type MyResponse struct {
    // 如果 Python 返回 "fund_code"，那 tag 必须是 `json:"fund_code"`
    // 如果写成 `json:"fundCode"`，前端就崩了
    FundCode string `json:"fund_code"`
}
```

---

## 附录：Go 新手必坑 & 推荐学习路径

### 最常犯的 5 个错误

**1. 忘记检查 error**
```go
// 错
result, _ := doSomething()  // _ 把错误吞了

// 对
result, err := doSomething()
if err != nil {
    log.Printf("出错: %v", err)
    return nil, err
}
```

**2. 忘了 defer resp.Body.Close()，导致连接泄漏**
```go
resp, _ := http.Get(url)
defer resp.Body.Close()  // 这行必须写完 get 立刻就写！
```

**3. goroutine 闭包捕获循环变量**
```go
// 错
for _, code := range codes {
    go func() {
        process(code)  // 所有 goroutine 读到的是最后一个 code！
    }()
}

// 对（Go 1.22+ 已修复，但建议养成好习惯）
for _, code := range codes {
    c := code
    go func() {
        process(c)
    }()
}
```

**4. 结构体字段小写导致 JSON 序列化失败**
```go
// 错
type User struct {
    name string  // 小写——json.Marshal 输出 {}
}

// 对
type User struct {
    Name string `json:"name"`  // 大写 + json tag
}
```

**5. 没把结构体指针传给 GORM**
```go
// 错
var user model.User
db.First(user, 1)  // 传了值不是指针，GORM 没法改 user

// 对
db.First(&user, 1)  // 一定要传指针
```

### 推荐学习路径

```
第 1 天  Step 1: 让 Gin 跑起来
第 2 天  Step 2: 所有路由文件建好
第 3 天  Step 3: Health handler（最快的正向反馈）
第 4-7 天 Step 3: Auth handler（最复杂的单体模块）
第 8 天  Step 4: akshare 客户端 + 一个接口
第 9 天  Step 5: Fund service + handler
第 10 天 Step 6: 接入所有剩余路由
第 11 天 Step 7: AI 调用
第 12 天 Step 8-9: 夜估 + 后台任务
第 13 天 Step 10-11: Agent Token + 邀请裂变
第 14 天 Step 12-13: 静态文件 + 联调
```

### 常用 Go 命令

```bash
# 运行
go run cmd/server/main.go

# 编译
go build -o huahua-server cmd/server/main.go

# 测试
go test ./...                    # 全部测试
go test -v ./internal/handler/   # 某个包测试（详细输出）

# 依赖
go mod tidy     # 清理/下载依赖
go get github.com/xxx/xxx  # 安装一个新依赖

# 代码检查
go vet ./...    # 静态检查
```

### 依赖安装清单

随着项目推进，你需要跑这些 `go get`：

```bash
# 框架
go get github.com/gin-gonic/gin

# 数据库
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get github.com/redis/go-redis/v9

# JWT
go get github.com/golang-jwt/jwt/v5

# 密码哈希
go get golang.org/x/crypto/bcrypt

# 精确小数计算（基金价格用）
go get github.com/shopspring/decimal

# 配置（读 .env）
go get github.com/joho/godotenv
```

每次跑完 `go get`，`go.mod` 和 `go.sum` 会自动更新。

---

> **最后的话**：
>
> 这份指南把 README.md 第 17 条的 13 个步骤展开成了每一步的具体代码和思路。
> 每一步都给出了 Python 原版代码对照，你可以打开原文件边看边翻译。
>
> 别怕犯错——Go 编译器会在你运行前就告诉你哪里错了。
> 碰到编译错误先看错误信息，然后搜索，再不懂就来问我。
>
> 写完这个项目，你不仅学会了 Go + Gin 的分层架构，
> 也深入理解了"花花日记"这个基金投资助手的业务逻辑。
