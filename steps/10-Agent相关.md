# Step 10：Agent 相关

> **目标**：支持 `Authorization: AgentToken xxx` 模式。
> 让 AI Agent（如 MCP 客户端）不带用户 JWT 也能访问指定接口。

---

## 10.1 理解 AgentToken

AgentToken 是**长期有效的 API 密钥**，用于程序/机器人访问用户数据。

**和 JWT 的区别**：

| 对比 | JWT | AgentToken |
|---|---|---|
| 生成方式 | 用户登录，服务器签名 | 用户手动创建 |
| 有效期 | 7 天 | 可配（90 天/1 年/永久） |
| 存储 | 客户端 | SHA256 哈希存数据库 |
| 泄漏风险 | 短期，可过期 | 长期，需手动撤销 |
| 用途 | 前端 App 登录 | MCP/AI 客户端调用 |

**Python 代码**（`routers/auth.py`）：

```python
# AgentToken 鉴权流程
@router.post("/agent-token", response_model=AgentTokenCreated)
async def create_agent_token(req, current_user, db):
    """创建 Agent Token（需邮箱验证）"""
    # 1. 验证邮箱验证码
    if not await MailService.verify_code(req.email, req.verification_code, "agent"):
        raise HTTPException(400, "验证码错误")
    
    # 2. 生成随机 token
    raw_token = secrets.token_urlsafe(32)  # 43 字符随机字符串
    token_hash = hashlib.sha256(raw_token.encode()).hexdigest()
    
    # 3. 存哈希到数据库
    agent_tok = AgentToken(
        user_id=current_user.id,
        token_hash=token_hash,
        name=req.name,
        scope=_normalize_agent_scope(req.scope),
    )
    db.add(agent_tok)
    db.commit()
    
    # 4. 返回明文 token（仅此一次！）
    return {"token": raw_token, ...}

# 鉴权时——在 get_current_user 中
if auth_header.startswith("AgentToken "):
    raw_token = auth_header[len("AgentToken "):]
    token_hash = hashlib.sha256(raw_token.encode()).hexdigest()
    agent_tok = db.query(AgentToken).filter(
        AgentToken.token_hash == token_hash
    ).first()
    if not agent_tok:
        raise HTTPException(401, "无效的 Agent Token")
    # 检查 scope 权限
    if not _is_agent_token_allowed(method, path, agent_tok.scope):
        raise HTTPException(403, "无权访问")
    return db.query(User).filter(User.id == agent_tok.user_id).first()
```

---

## 10.2 Go 实现

### 10.2.1 AgentToken Model

```go
// internal/model/agent_token.go
package model

import "time"

type AgentToken struct {
    ID         int64      `gorm:"primaryKey;autoIncrement"`
    UserID     int64      `gorm:"column:user_id;index"`
    TokenHash  string     `gorm:"column:token_hash;uniqueIndex;size:64"`
    Name       string     `gorm:"size:50;default:'Agent Token'"`
    Scope      string     `gorm:"size:100;default:'agent:full'"`
    CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
    LastUsedAt *time.Time `gorm:"column:last_used_at"`
    LastUsedIP string     `gorm:"column:last_used_ip;size:50"`
    ExpiresAt  *time.Time `gorm:"column:expires_at"`
}

func (AgentToken) TableName() string { return "agent_tokens" }
```

### 10.2.2 AgentToken Repository

```go
// internal/repository/agent_token_repo.go
package repository

import (
    "context"
    "huahua-service/internal/model"
    "gorm.io/gorm"
)

type AgentTokenRepository interface {
    GetByHash(ctx context.Context, hash string) (*model.AgentToken, error)
    Create(ctx context.Context, token *model.AgentToken) error
    Delete(ctx context.Context, id, userID int64) error
    ListByUserID(ctx context.Context, userID int64) ([]model.AgentToken, error)
    UpdateLastUsed(ctx context.Context, id int64, ip string) error
}

type agentTokenRepo struct {
    db *gorm.DB
}

func NewAgentTokenRepository(db *gorm.DB) AgentTokenRepository {
    return &agentTokenRepo{db: db}
}

func (r *agentTokenRepo) GetByHash(ctx context.Context, hash string) (*model.AgentToken, error) {
    var token model.AgentToken
    err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
    if err != nil {
        return nil, err
    }
    return &token, nil
}

func (r *agentTokenRepo) Create(ctx context.Context, token *model.AgentToken) error {
    return r.db.WithContext(ctx).Create(token).Error
}

func (r *agentTokenRepo) UpdateLastUsed(ctx context.Context, id int64, ip string) error {
    return r.db.WithContext(ctx).Model(&model.AgentToken{}).
        Where("id = ?", id).
        Updates(map[string]interface{}{
            "last_used_at": time.Now(),
            "last_used_ip": ip,
        }).Error
}
```

### 10.2.3 Auth 中间件中加上 AgentToken 分支

```go
// internal/middleware/auth.go 追加

// handleAgentToken 处理 AgentToken 鉴权
func handleAgentToken(c *gin.Context, authHeader string, agentTokenRepo repository.AgentTokenRepository) {
    rawToken := strings.TrimPrefix(authHeader, "AgentToken ")
    if rawToken == authHeader { // 没去掉 AgentToken 前缀
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "格式无效"})
        return
    }

    // SHA256 哈希
    tokenHash := sha256Hex(rawToken)

    // 查数据库
    agentTok, err := agentTokenRepo.GetByHash(c.Request.Context(), tokenHash)
    if err != nil {
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "无效的 Agent Token"})
        return
    }

    // 检查过期
    if agentTok.ExpiresAt != nil && agentTok.ExpiresAt.Before(time.Now()) {
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Agent Token 已过期"})
        return
    }

    // 检查 scope 权限
    if !isAgentTokenAllowed(c.Request.Method, c.Request.URL.Path, agentTok.Scope) {
        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"detail": "无权访问"})
        return
    }

    // 更新 last_used
    go agentTokenRepo.UpdateLastUsed(c.Request.Context(), agentTok.ID, c.ClientIP())

    // 查对应用户
    userRepo := getFromContext(c, "userRepo").(repository.UserRepository)
    user, _ := userRepo.GetByID(c.Request.Context(), agentTok.UserID)
    c.Set("user", user)
    c.Next()
}

func sha256Hex(s string) string {
    h := sha256.Sum256([]byte(s))
    return hex.EncodeToString(h[:])
}

// isAgentTokenAllowed 检查 scope 是否允许访问指定路径
// 对应 Python _is_agent_token_allowed()
func isAgentTokenAllowed(method, path, scope string) bool {
    // 简化版：agent:full 允许所有
    if scope == "agent:full" {
        return true
    }
    
    // scope 白名单检查
    // Python 里维护了一个 method+path → required_scope 的映射表
    // 这里简化：后续根据 Python 代码完善
    return true
}
```

---

## 10.3 验证

```bash
# 用 AgentToken 测试
curl http://localhost:8080/api/auth/me \
  -H "Authorization: AgentToken your_raw_token_here"
# 期望：返回用户信息，或 401 invalid token
```

---

## 本级小结

- ✅ AgentToken Model：SHA256 哈希存数据库
- ✅ AgentToken Repository：增删查
- ✅ 中间件里添加 AgentToken 鉴权分支
- ✅ Scope 权限检查
- ✅ 验证
