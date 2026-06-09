# Step 3：不依赖外部 API 的 handler

> **目标**：先写"自成一体"的模块——health、version、auth（含注册登录 + JWT 中间件）、admin、public。
> 它们不依赖 akshare（财经数据）和 AI，写完就能测通。

---

## 3.1 health handler —— 最简单、最快得到正向反馈

### Python 源码

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（38 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
# routers/health.py — 健康检查

from fastapi import APIRouter, Response
from sqlalchemy import text
from starlette.concurrency import run_in_threadpool
from cache import cache
from database import SessionLocal

router = APIRouter(prefix="/api", tags=["health"])

@router.get("/health")
async def health(response: Response):
    # 检查数据库
    db = "ok"
    try:
        with SessionLocal() as session:
            session.execute(text("SELECT 1"))
    except Exception as e:
        db = "error"
    
    # 检查 Redis
    redis_status = "ok"
    try:
        await cache.set("_health:ping", "1", ttl=10)
        val = await cache.get("_health:ping")
        if val != "1":
            redis_status = "fallback"
    except Exception:
        redis_status = "error"
    
    components = {"db": db, "redis": redis_status}
    has_error = any(v == "error" for v in components.values())
    status = "error" if has_error else ("degraded" if "fallback" in components.values() else "ok")
    
    if has_error:
        response.status_code = 503
    
    return {"status": status, "components": components}
```
</details>


**Python 这段代码做了什么**：
1. 连接数据库执行 `SELECT 1` 检查数据库是否活着
2. 写 Redis key 再读回来检查 Redis 是否活着
3. 根据结果返回状态：ok（全部正常）/ degraded（降级运行）/ error（不可用）
4. 有 error 时返回 HTTP 503

### Go 实现

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （101 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

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

// HealthHandler 健康检查处理器
// 对应 Python routers/health.py
type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
    return &HealthHandler{}
}

// Ping 健康检查
// GET /api/health
// 对应 Python health() 函数
func (h *HealthHandler) Ping(c *gin.Context) {
    // Python 的 gin.H 是 map[string]interface{} 的快捷写法
    components := gin.H{}
    
    // ── 检查数据库 ──
    // Python: session.execute(text("SELECT 1"))
    dbStatus := "ok"
    if db.GORM != nil {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
        defer cancel()
        
        var result int
        if err := db.GORM.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
            dbStatus = "error"
        }
    }
    components["db"] = dbStatus
    
    // ── 检查 Redis ──
    // Python: await cache.set("_health:ping", "1") + await cache.get("_health:ping")
    redisStatus := "ok"
    if db.Redis != nil {
        ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
        defer cancel()
        
        if err := db.Redis.Set(ctx, "_health:ping", "1", 10*time.Second).Err(); err != nil {
            redisStatus = "error"
        } else {
            val, err := db.Redis.Get(ctx, "_health:ping").Result()
            if err != nil || val != "1" {
                redisStatus = "fallback"
            }
        }
    }
    components["redis"] = redisStatus
    
    // ── 组装响应 ──
    // Python: {"status": "ok"/"degraded"/"error", "components": {"db": ..., "redis": ...}}
    hasError := dbStatus == "error" || redisStatus == "error"
    isDegraded := dbStatus == "fallback" || redisStatus == "fallback"
    
    statusCode := http.StatusOK
    status := "ok"
    if hasError {
        status = "error"
        statusCode = http.StatusServiceUnavailable  // 503
    } else if isDegraded {
        status = "degraded"
    }
    
    c.JSON(statusCode, gin.H{
        "status":     status,
        "components": components,
    })
}

// Redis Redis 连接池诊断
// GET /api/health/redis
// 对应 Python health_redis()
func (h *HealthHandler) Redis(c *gin.Context) {
    if db.Redis == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Redis not connected"})
        return
    }
    
    // Redis INFO client 命令
    // Python: await cache.redis.info("clients")
    info, err := db.Redis.Info(c.Request.Context(), "clients").Result()
    if err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "clients_info": info,
    })
}
```
</details>


**健康检查 handler 能先跑通的原理**：
- 这个 handler 不需要访问数据库！`db.GORM` 和 `db.Redis` 这步可以跳过
- 你只要把 Gin 服务起起来，不连数据库也能返回 503 状态码
- 这是最快能看到"我的 Go 服务真的在工作了"的正向反馈

---

## 3.2 Auth handler —— 最关键、最复杂的模块

Auth 模块包含注册、登录、JWT、密码哈希、邮箱验证码、AgentToken……多个子功能。
我们把 Python 的 `routers/auth.py`（~300 行）拆成 Go 的多个文件来实现。

### 3.2.1 Schema —— 请求/响应数据结构

**Python 的 schemas.py**（Pydantic 模型）：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（22 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
# schemas.py — 请求和响应的数据结构定义

from pydantic import BaseModel, Field, EmailStr

# ── 注册请求 ──
class UserCreate(BaseModel):
    username: str = Field(..., pattern=r"^[a-zA-Z0-9]{4,20}$")
    password: str = Field(..., min_length=8)
    email: EmailStr
    verification_code: str
    nickname: str | None = Field(None, max_length=16)
    invite_code: str | None = None

# ── 登录请求 ──
class LoginRequest(BaseModel):
    username: str
    password: str

# ── 令牌响应 ──
class TokenResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
```
</details>


**Pydantic 的 `Field` 参数含义**：
- `pattern`：正则表达式校验
- `min_length` / `max_length`：长度限制
- `default` / `default=None`：默认值 / 可选值
- `EmailStr`：自动校验邮箱格式

**Go 实现 —— `json` tag 和 `binding` tag**：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （27 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/schema/auth.go
package schema

// RegisterRequest 对应 Python 的 UserCreate
// json:"username" — 告诉 Gin JSON 反序列化时字段名是 "username"
// binding:"required,min=4,max=20" — Gin 自动校验：必填、4-20 字符
type RegisterRequest struct {
    Username         string `json:"username" binding:"required,min=4,max=20"`
    Password         string `json:"password" binding:"required,min=8"`
    Email            string `json:"email" binding:"required,email"`
    VerificationCode string `json:"verification_code" binding:"required"`
    Nickname         string `json:"nickname"`
    InviteCode       string `json:"invite_code"`
}

// LoginRequest 对应 Python 的 LoginRequest
type LoginRequest struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

// TokenResponse 对应 Python 的 TokenResponse
type TokenResponse struct {
    AccessToken string `json:"access_token"`    // Python: access_token
    TokenType   string `json:"token_type"`      // Python: token_type
    UserID      int64  `json:"user_id"`
}
```
</details>


**Go binding tag 速查**：

| tag | 作用 | 对应 Pydantic |
|---|---|---|
| `required` | 必填 | `Field(required=True)` |
| `min=4` | 最少 4 字符 | `min_length=4` |
| `max=20` | 最多 20 字符 | `max_length=20` |
| `email` | 邮箱格式 | `EmailStr` |
| `len=6` | 固定 6 字符 | `pattern="^.{6}$"` |

### 3.2.2 JWT 工具

**Python JWT 部分**（写在 `routers/auth.py` 中）：

```python
# Python JWT 工具函数
from jwt import encode as jwt_encode, decode as jwt_decode

def create_access_token(data: dict):
    """生成 JWT token"""
    to_encode = data.copy()
    to_encode.update({
        "exp": datetime.now(timezone.utc) + timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    })
    return jwt_encode(to_encode, settings.SECRET_KEY, algorithm=settings.ALGORITHM)
```

**Python 的 JWT 函数做了什么**：
1. 接收一个字典（如 `{"sub": "testuser"}`）
2. 加上过期时间 `exp`
3. 用 SECRET_KEY 和 HS256 算法签名
4. 返回字符串格式的 JWT token

**Go 实现**：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （60 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/security/jwt.go
package security

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// JWTConfig JWT 配置
// 从 config.Config 传进来
type JWTConfig struct {
    Secret   string        // 签名密钥
    ExpireIn time.Duration // 过期时长（7 天 = 7*24*time.Hour）
}

// Claims JWT 载荷（payload）里存的数据
type Claims struct {
    Sub string `json:"sub"`  // 用户名作为 subject
    jwt.RegisteredClaims     // 嵌入标准字段（ExpiresAt, IssuedAt 等）
}

// GenerateToken 生成 JWT token
// 对应 Python create_access_token({"sub": username})
func GenerateToken(cfg JWTConfig, username string) (string, error) {
    claims := Claims{
        Sub: username,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.ExpireIn)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "huahua",
        },
    }

    // jwt.NewWithClaims + SignedString = 签名
    // Python 的 jwt_encode(data, key, algorithm="HS256")
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(cfg.Secret))
}

// ParseToken 解析和验证 JWT token
// 对应 Python jwt_decode(token, SECRET_KEY, algorithms=["HS256"])
func ParseToken(secret, tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, 
        func(token *jwt.Token) (interface{}, error) {
            // 验证签名算法是 HS256
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
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
</details>


### 3.2.3 Bcrypt 密码哈希

**Python**（写在 `routers/auth.py` 中）：

```python
from passlib.context import CryptContext
pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto")

def verify_password(plain, hashed): return pwd_context.verify(plain, hashed)
def get_password_hash(password): return pwd_context.hash(password)
```

**Go**：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （20 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/security/bcrypt.go
package security

import "golang.org/x/crypto/bcrypt"

// HashPassword 密码哈希
// 对应 Python get_password_hash()
func HashPassword(password string) (string, error) {
    // bcrypt.DefaultCost = 10
    // cost 越大越安全但越慢，Python passlib 默认也是 10
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

// CheckPassword 验证密码
// 对应 Python verify_password()
func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```
</details>


### 3.2.4 Auth 中间件

**Python `get_current_user`**（FastAPI 依赖注入）：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（50 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
from fastapi.security import OAuth2PasswordBearer

oauth2_scheme = OAuth2PasswordBearer(tokenUrl="/api/auth/token", auto_error=False)

async def get_current_user(
    request: Request,
    token_header: Optional[str] = Depends(oauth2_scheme),
    db: Session = Depends(get_db)
):
    """从请求中提取当前登录用户。
    
    两种鉴权方式：
    1. Authorization: Bearer <JWT> （浏览器/手机 App）
    2. Authorization: AgentToken <raw_token> （AI Agent）
    3. 从 cookie access_token 读取（Web 浏览器）
    """
    auth_header = request.headers.get("Authorization", "")
    
    # ── AgentToken 模式 ──
    if auth_header.startswith("AgentToken "):
        raw_token = auth_header[len("AgentToken "):]
        token_hash = hashlib.sha256(raw_token.encode()).hexdigest()
        agent_tok = db.query(AgentToken).filter(
            AgentToken.token_hash == token_hash
        ).first()
        if not agent_tok:
            raise HTTPException(401, "无效的 Agent Token")
        # 检查 scope 权限
        if not _is_agent_token_allowed(request.method, request.url.path, agent_tok.scope):
            raise HTTPException(403, "Agent Token 无权访问")
        return db.query(User).filter(User.id == agent_tok.user_id).first()
    
    # ── JWT 模式 ──
    token = token_header or request.cookies.get("access_token")
    if not token:
        raise HTTPException(401)
    
    try:
        payload = jwt_decode(token, settings.SECRET_KEY, algorithms=[settings.ALGORITHM])
        username = payload.get("sub")
        if username is None:
            raise HTTPException(401)
    except JWTError:
        raise HTTPException(401)
    
    user = db.query(User).filter(User.username == username).first()
    if user is None:
        raise HTTPException(401)
    
    return user
```
</details>


**Go 中间件**：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （84 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

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

// AuthRequired 创建认证中间件
// 用法：gin router 里用 auth := g.Group("", mw.AuthRequired())
// 对应 Python 的 get_current_user 依赖
func AuthRequired(jwtCfg security.JWTConfig, userRepo repository.UserRepository) gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")

        // ── AgentToken 模式 ──
        if strings.HasPrefix(authHeader, "AgentToken ") {
            handleAgentToken(c, authHeader, userRepo)
            return
        }

        // ── Bearer JWT 模式 ──
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

        // 解析 JWT —— 对应 Python jwt_decode
        claims, err := security.ParseToken(jwtCfg.Secret, token)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Token 无效或已过期"})
            return
        }

        // 查用户 — 对应 Python db.query(User).filter(...).first()
        user, err := userRepo.GetByUsername(c.Request.Context(), claims.Sub)
        if err != nil || user == nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "用户不存在"})
            return
        }

        // 把用户信息存到 Gin Context —— 后续 handler 用 c.Get("user") 取
        // 对应 Python 的 request.state.user
        c.Set("user", user)
        c.Next()
    }
}

// handleAgentToken AgentToken 鉴权（简化版，Step 10 详细展开）
func handleAgentToken(c *gin.Context, authHeader string, userRepo repository.UserRepository) {
    c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "AgentToken 暂未实现"}}
}

// AdminRequired 管理员中间件
// 对应 Python get_current_admin()
func AdminRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        user, exists := c.Get("user")
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "未登录"})
            return
        }
        
        u := user.(*model.User)  // 类型断言：把 interface{} 转回 *model.User
        if !u.IsAdmin {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "需要管理员权限"})
            return
        }
        c.Next()
    }
}
```
</details>


### 3.2.5 Auth Service

**Python 的函数风格的业务逻辑**（写在 `routers/auth.py` 中，非类）：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（34 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
# Python 的注册逻辑（在 routers/auth.py 的 register() 函数中）
# Python 把业务逻辑和路由处理混在一起写，Go 必须分离

@router.post("/register", response_model=UserOut)
async def register(request: Request, user_in: UserCreate, db: Session = Depends(get_db)):
    # 1. IP 频率限制
    client_ip = get_client_ip(request)
    await _check_auth_rate_limit(f"ip:{client_ip}:register", limit=3, period=86400)
    
    # 2. 验证邮箱验证码
    if not await MailService.verify_code(user_in.email, user_in.verification_code, "reg"):
        raise HTTPException(400, "验证码错误或已过期")
    
    # 3. 检查用户名是否已有
    if db.query(User).filter(User.username == user_in.username).first():
        raise HTTPException(400, "Username already registered")
    if db.query(User).filter(User.email == user_in.email).first():
        raise HTTPException(400, "Email already registered")
    
    # 4. 创建用户
    new_user = User(
        username=user_in.username,
        email=user_in.email,
        hashed_password=get_password_hash(user_in.password),
        nickname=user_in.nickname or user_in.username,
        uid=generate_uid(),
    )
    db.add(new_user)
    db.commit()
    db.refresh(new_user)
    
    # 5. 生成 JWT 返回
    token = create_access_token({"sub": new_user.username})
    return UserOut.model_validate(new_user)
```
</details>


**Go 的 Service 层**：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （119 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

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

// AuthService 认证业务逻辑
// Python 是直接写在 router 函数里的，Go 要求抽象到 Service 层
type AuthService struct {
    db       *gorm.DB
    userRepo repository.UserRepository
    jwtCfg   security.JWTConfig
}

func NewAuthService(db *gorm.DB, userRepo repository.UserRepository, jwtCfg security.JWTConfig) *AuthService {
    return &AuthService{db: db, userRepo: userRepo, jwtCfg: jwtCfg}
}

// Register 注册新用户
// 对应 Python routers/auth.py 的 register()
//
// 对比 Python 代码：
// Python 直接在函数里写 db.query(User).filter(...) 
// Go 调 userRepo.GetByUsername(ctx, ...) —— 数据库操作转到 repository 层
func (s *AuthService) Register(ctx context.Context, req schema.RegisterRequest) (*model.User, string, error) {
    // 1. 检查用户名和邮箱是否已存在
    // Python: db.query(User).filter(User.username == user_in.username).first()
    existing, err := s.userRepo.GetByUsername(ctx, req.Username)
    if err != nil {
        return nil, "", err
    }
    if existing != nil {
        return nil, "", errors.New("用户名已被注册")
    }
    
    // Python: db.query(User).filter(User.email == user_in.email).first()
    existingEmail, err := s.userRepo.GetByEmail(ctx, req.Email)
    if err != nil {
        return nil, "", err
    }
    if existingEmail != nil {
        return nil, "", errors.New("邮箱已被注册")
    }

    // 2. 密码哈希
    // Python: get_password_hash(user_in.password)
    hashedPwd, err := security.HashPassword(req.Password)
    if err != nil {
        return nil, "", err
    }

    // 3. 创建用户对象
    user := &model.User{
        Username:       req.Username,
        Email:          req.Email,
        HashedPassword: hashedPwd,
        Nickname:       req.Nickname,
        Avatar:         "https://api.dicebear.com/7.x/adventurer/svg",
        UID:            generateUID(),  // 8位数字
    }

    // Python: db.add(new_user); db.commit()
    if err := s.userRepo.Create(ctx, user); err != nil {
        return nil, "", err
    }

    // 4. 生成 JWT
    // Python: create_access_token({"sub": new_user.username})
    token, err := security.GenerateToken(s.jwtCfg, user.Username)
    if err != nil {
        return nil, "", err
    }

    return user, token, nil
}

// Login 用户登录
// 对应 Python login()
func (s *AuthService) Login(ctx context.Context, username, password string) (*model.User, string, error) {
    // Python: db.query(User).filter(User.username == form_data.username).first()
    user, err := s.userRepo.GetByUsername(ctx, username)
    if err != nil {
        return nil, "", err
    }
    if user == nil {
        return nil, "", errors.New("用户名或密码错误")
    }

    // Python: verify_password(password, user.hashed_password)
    if !security.CheckPassword(password, user.HashedPassword) {
        return nil, "", errors.New("用户名或密码错误")
    }

    // Python: create_access_token({"sub": user.username})
    token, err := security.GenerateToken(s.jwtCfg, user.Username)
    if err != nil {
        return nil, "", err
    }

    return user, token, nil
}

// generateUID 生成 8 位数字 UID
// 对应 Python models.py 的 generate_uid()
func generateUID() string {
    // 简单实现：用纳秒时间戳后 8 位
    return fmt.Sprintf("%08d", time.Now().UnixNano()%100000000)
}
```
</details>


### 3.2.6 Auth Handler

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （78 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

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

// Register POST /api/auth/register
// 对应 Python register()
func (h *AuthHandler) Register(c *gin.Context) {
    var req schema.RegisterRequest
    // Python: user_in: UserCreate = Depends() ← Pydantic 自动校验
    // Go: c.ShouldBindJSON(&req) ← 手动绑定，绑定失败返回 400
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
        return
    }

    user, token, err := h.authService.Register(c.Request.Context(), req)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
        return
    }

    // Python: return UserOut.model_validate(new_user)
    // Go: 手动组装响应
    c.JSON(http.StatusOK, schema.TokenResponse{
        AccessToken: token,
        TokenType:   "bearer",
        UserID:      user.ID,
    })
}

// Login POST /api/auth/token
// 对应 Python login()
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

// Me GET /api/auth/me
// 对应 Python read_users_me()
func (h *AuthHandler) Me(c *gin.Context) {
    user, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"detail": "未登录"})
        return
    }
    c.JSON(http.StatusOK, user)
}
```
</details>


### 3.2.7 串联：在 app.go 注入所有依赖

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （53 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/app/app.go
package app

import (
    "time"

    "huahua-service/internal/config"
    "huahua-service/internal/db"
    "huahua-service/internal/handler"
    "huahua-service/internal/middleware"
    "huahua-service/internal/repository"
    "huahua-service/internal/router"
    "huahua-service/internal/security"
    "huahua-service/internal/service/auth"

    "github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())

    // ===== 创建所有依赖（DI 注入） =====
    // Python 用全局变量 + Depends()，Go 手动构造并传递

    // Repository 层
    userRepo := repository.NewUserRepository(db.GORM)

    // Security 配置
    jwtCfg := security.JWTConfig{
        Secret:   cfg.SecretKey,
        ExpireIn: 7 * 24 * time.Hour,  // 7 天
    }

    // Service 层
    authService := auth.NewAuthService(db.GORM, userRepo, jwtCfg)

    // Middleware
    mw := &middleware.Middleware{
        AuthRequired: middleware.AuthRequired(jwtCfg, userRepo),
        AdminRequired: middleware.AdminRequired(),
    }

    // Handler 层
    h := handler.NewHandlers(
        handler.WithAuth(authService),
    )

    // Router 层
    router.Register(r, h)

    return r
}
```
</details>


### 3.2.8 更新 handler.NewHandlers

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （35 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/handler/handler.go
package handler

import "huahua-service/internal/service/auth"

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

// HandlersOption 函数选项模式——方便逐步注入依赖
type HandlersOption func(*Handlers)

func WithAuth(s *auth.AuthService) HandlersOption {
    return func(h *Handlers) { h.Auth = NewAuthHandler(s) }
}

func NewHandlers(opts ...HandlersOption) *Handlers {
    h := &Handlers{
        Health: NewHealthHandler(),
        // 其他默认为 nil——用到时再注入
    }
    for _, opt := range opts {
        opt(h)
    }
    return h
}
```
</details>


---

## 3.3 在 router/auth.go 中打开路由

```go
// internal/router/auth.go
func registerAuth(rg *gin.RouterGroup, h *handler.AuthHandler) {
    g := rg.Group("/auth")
    
    // 无需认证的路由
    g.POST("/register", h.Register)  // POST /api/auth/register
    g.POST("/token", h.Login)        // POST /api/auth/token
    
    // 需要认证的路由（后面加中间件）
    // auth := g.Group("", mw.AuthRequired)
    // auth.GET("/me", h.Me)
}
```

---

## 3.4 验证

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 bash  （20 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```bash
# 1. 编译
go build ./...

# 2. 启动（如果连不上数据库，先注释掉 db.InitGORM 和 db.InitRedis）
go run cmd/server/main.go &

# 3. 测试注册
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"Pass1234","email":"test@test.com","verification_code":"000000"}'
# 期望响应：{"access_token":"eyJ...","token_type":"bearer","user_id":1}

# 4. 测试登录
curl -X POST http://localhost:8080/api/auth/token \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"Pass1234"}'

# 5. 测试健康检查
curl http://localhost:8080/api/health
# 期望：{"components":{"db":"ok","redis":"ok"},"status":"ok"}
```
</details>


---

## 本级小结

- ✅ Health handler：数据库 + Redis 健康检查
- ✅ Schema：注册/登录请求响应结构
- ✅ JWT 工具：生成和解析 JWT token
- ✅ Bcrypt 工具：密码哈希和验证
- ✅ Auth 中间件：Bearer JWT 鉴权
- ✅ Auth Service：注册和登录业务逻辑
- ✅ Auth Handler：HTTP 请求入口

**下一步**：Step 4 —— 封装核心数据源（akshare HTTP 客户端）。

---

## 附录：Go 中 interface{} 和类型断言

Auth middleware 里用到了 `c.Set("user", user)` 和后面的 `user.(*model.User)`：

```go
// 存
c.Set("user", user)  // user 是 *model.User 类型

// 取——要"断言"回原类型
// v, ok := x.(T) 尝试把 interface{} 转换成 T 类型
u, ok := c.Get("user")
if !ok {
    // context 里没有 user
}
user, ok := u.(*model.User)  // 断言为 *model.User
if !ok {
    // 类型不对
}
```

**为什么需要断言**？Go 的 `c.Get` 返回 `interface{}`（可以装任何类型），
取出来时要告诉编译器"我把它当成 *model.User 用"。
这类似于 Python 的 `isinstance()` 检查。

---

## 补充 1：MailService——邮箱验证码服务（从 Step 14 移入）

Auth handler 的注册/登录流程中需要发邮箱验证码。Python 用 `fastapi-mail` 库，Go 用 `gomail`。

### Python 源码

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（26 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
# services/mail_service.py
class MailService:
    @staticmethod
    async def send_verification_code(email: str, type_prefix: str = "reg"):
        # 1. 检查频率：同一邮箱同一类型 2 分钟冷却
        rate_key = f"mail:rate:{type_prefix}:{email}"
        if await cache.get(rate_key):
            return False, "发送太频繁"
        
        # 2. 生成 6 位安全随机码
        code = str(secrets.randbelow(900000) + 100000)
        
        # 3. 存 Redis：verify:reg:test@test.com = 123456, TTL=600s
        verify_key = f"verify:{type_prefix}:{email}"
        await cache.set(verify_key, code, ttl=600)
        await cache.set(rate_key, "1", ttl=120)
        
        # 4. 发 SMTP 邮件
        message = MessageSchema(subject="验证码", recipients=[email], body=html)
        fm = FastMail(ConnectionConfig(...))
        await fm.send_message(message)
    
    @staticmethod
    async def verify_code(email, code, type_prefix="reg"):
        stored = await cache.get(f"verify:{type_prefix}:{email}")
        return str(stored) == str(code)  # 匹配后删除（一次性）
```
</details>


### Go 实现

依赖安装：`go get gopkg.in/gomail.v2`

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （67 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/service/mail/mail_service.go — 新建文件
package mail

import (
    "context"
    "crypto/rand"
    "fmt"
    "math/big"
    "time"

    "huahua-service/internal/config"
    "huahua-service/pkg/cache"

    "gopkg.in/gomail.v2"
)

type Service struct {
    cfg   config.MailConfig
    cache *cache.Cache
}

func NewService(cfg config.MailConfig, cache *cache.Cache) *Service {
    return &Service{cfg: cfg, cache: cache}
}

// SendVerificationCode 发送验证码
// typePrefix: reg(注册) / reset(重置密码) / bind(绑定) / agent(AgentToken)
func (s *Service) SendVerificationCode(ctx context.Context, email, typePrefix string) error {
    // 1. 检查频率
    rateKey := fmt.Sprintf("mail:rate:%s:%s", typePrefix, email)
    if v, _ := s.cache.Get(ctx, rateKey); v != "" {
        return fmt.Errorf("发送太频繁，请2分钟后再试")
    }
    // 2. 生成 6 位随机码
    code, _ := generateCode(6)
    // 3. 存 Redis
    verifyKey := fmt.Sprintf("verify:%s:%s", typePrefix, email)
    s.cache.Set(ctx, verifyKey, code, 10*time.Minute)
    s.cache.Set(ctx, rateKey, "1", 2*time.Minute)
    // 4. 发 SMTP 邮件
    m := gomail.NewMessage()
    m.SetHeader("From", s.cfg.From)
    m.SetHeader("To", email)
    m.SetHeader("Subject", "花花日记 邮箱验证")
    m.SetBody("text/html", fmt.Sprintf(
        `<div><h2>验证码</h2><p>验证码：<strong>%s</strong><br>10分钟内有效</p></div>`, code))
    d := gomail.NewDialer(s.cfg.Host, s.cfg.Port, s.cfg.Username, s.cfg.Password)
    return d.DialAndSend(m)
}

// VerifyCode 校验验证码（一次性）
func (s *Service) VerifyCode(ctx context.Context, email, code, typePrefix string) bool {
    key := fmt.Sprintf("verify:%s:%s", typePrefix, email)
    stored, _ := s.cache.Get(ctx, key)
    if stored != code { return false }
    s.cache.Del(ctx, key) // 用完即删
    return true
}

func generateCode(length int) (string, error) {
    code := make([]byte, length)
    for i := range code {
        n, _ := rand.Int(rand.Reader, big.NewInt(10))
        code[i] = byte('0' + n.Int64())
    }
    return string(code), nil
}
```
</details>


---

## 补充 2：全量中间件（从 Step 14 移入）

### 2.1 中间件聚合结构

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （27 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/middleware/middleware.go — 新建/覆盖
package middleware

import (
    "huahua-service/internal/repository"
    "huahua-service/internal/security"
    "github.com/gin-gonic/gin"
)

type Middleware struct {
    AuthRequired  gin.HandlerFunc
    AdminRequired gin.HandlerFunc
    CORS          gin.HandlerFunc
    RateLimit     gin.HandlerFunc
    GZip          gin.HandlerFunc
    Logger        gin.HandlerFunc
}

func New(jwtCfg security.JWTConfig, userRepo repository.UserRepository, agentTokenRepo repository.AgentTokenRepository) *Middleware {
    return &Middleware{
        AuthRequired:  AuthRequired(jwtCfg, userRepo, agentTokenRepo),
        AdminRequired: AdminRequired(),
        CORS:          CORS(),
        RateLimit:     RateLimit(1200, 60),
        Logger:        RequestLogger(),
    }
}
```
</details>


### 2.2 CORS 中间件

**Python 代码**：`app.add_middleware(CORSMiddleware, allow_origins=..., allow_methods=["GET","POST","PUT","DELETE","OPTIONS"])`

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （18 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/middleware/cors.go — 新建文件
package middleware

import (
    "time"
    "github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
    return cors.New(cors.Config{
        AllowOrigins:     []string{"*"}, // 生产环境改为具体域名
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
        AllowHeaders:     []string{"Content-Type", "Authorization", "X-Client-Platform", "Cache-Control"},
        AllowCredentials: true,
        MaxAge:           12 * time.Hour,
    })
}
```
</details>


安装：`go get github.com/gin-contrib/cors`

### 2.3 全局限流中间件

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （34 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/middleware/ratelimit.go — 新建文件
package middleware

import (
    "net/http"
    "huahua-service/internal/db"
    "huahua-service/pkg/ratelimit"
    "github.com/gin-gonic/gin"
)

// RateLimit 全局 API 限流
func RateLimit(calls, period int) gin.HandlerFunc {
    redisLimiter := ratelimit.NewRedisLimiter(db.Redis, calls, period)
    localLimiter := ratelimit.NewLocalLimiter(calls, period)

    return func(c *gin.Context) {
        path := c.Request.URL.Path
        // 只限 /api/，跳过 OPTIONS
        if len(path) < 4 || path[:4] != "/api" || c.Request.Method == "OPTIONS" {
            c.Next(); return
        }
        key := c.ClientIP()
        allowed, err := redisLimiter.Allow(c.Request.Context(), key)
        if err == nil && !allowed {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"detail": "请求过于频繁"})
            return
        }
        if err != nil && !localLimiter.Allow(key) {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"detail": "请求过于频繁"})
            return
        }
        c.Next()
    }
}
```
</details>


### 2.4 请求日志中间件

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （18 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/middleware/logger.go — 新建文件
package middleware

import (
    "log/slog"
    "time"
    "github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        c.Next()
        slog.Info("request", "method", c.Request.Method, "path", path,
            "status", c.Writer.Status(), "latency", time.Since(start).String(), "ip", c.ClientIP())
    }
}
```
</details>


### 2.5 中间件装配顺序（在 app.go 中）

必须跟 Python 的顺序一致：

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （17 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/app/app.go
func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    deps := bootstrap.Init(cfg)
    mw := deps.Middleware

    // 中间件顺序（和 Python 完全一致）
    r.Use(mw.Logger)        // 1. 请求日志
    r.Use(gin.Recovery())   // 2. Recovery（Python 的 exception_handler）
    r.Use(mw.CORS)          // 3. CORS
    r.Use(mw.RateLimit)     // 4. 全局限流
                           // 5. BlogStats（见下方补充3）
                           // 6. GZip

    router.Register(r, deps.Handlers, mw)
    return r
}
```
</details>


### 2.6 BlogStats 中间件

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （36 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/middleware/blogstats.go — 新建文件
package middleware

import (
    "crypto/sha256"
    "fmt"
    "time"
    "huahua-service/internal/db"
    "github.com/gin-gonic/gin"
)

func BlogStats() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if db.Redis == nil { return }
        path := c.Request.URL.Path
        // 只记录 GET /api/ 路径，跳过自身和热路径
        if c.Request.Method != "GET" || !startsWith(path, "/api/") || path == "/api/public/blog-stats" { return }
        hotPaths := []string{"/api/estimate/batch", "/api/history/", "/api/fund/today-timeline/"}
        for _, hp := range hotPaths { if startsWith(path, hp) { return } }

        ctx := c.Request.Context()
        today := time.Now().Format("2006-01-02")
        db.Redis.Incr(ctx, fmt.Sprintf("blog_stats:%s:visit_count", today))
        db.Redis.Expire(ctx, fmt.Sprintf("blog_stats:%s:visit_count", today), 72*time.Hour)
        // 唯一访客（SHA256 IP+UA）
        raw := c.ClientIP() + "|" + c.GetHeader("User-Agent")
        hash := fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
        db.Redis.SAdd(ctx, fmt.Sprintf("blog_stats:%s:visitor_users", today), hash)
        db.Redis.Expire(ctx, fmt.Sprintf("blog_stats:%s:visitor_users", today), 72*time.Hour)
    }
}

func startsWith(s, prefix string) bool {
    return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
```
</details>

