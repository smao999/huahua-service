<h1 align="center">🌸 HuaHua Service</h1>

<p align="center">
  <em>HuaHua — 后端 API 服务</em>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-blue?logo=go" alt="Go 1.25">
  <img src="https://img.shields.io/badge/Gin-1.12-green?logo=gin" alt="Gin 1.12">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="License">
</p>

基于 **Go 1.25+** 和 **Gin 框架**构建的 REST API 后端服务，集成 PostgreSQL + Redis + AI + JWT，涵盖用户认证、数据同步、AI 对话、后台任务、文件管理、权限控制等业务场景。

## 功能

| 模块 | 功能 |
|---|---|
| **用户系统** | 注册登录（JWT / bcrypt）、邮箱验证、密码重置、头像上传、VIP 会员体系 |
| **数据查询** | 实时数据、历史数据、批量查询、搜索、排行榜、分时数据 |
| **数据同步** | 用户数据多端同步、ETag 增量校验、导入导出 |
| **AI 集成** | OpenAI 兼容接口 / Gemini 双引擎、自动降级、截图识别、文本分析 |
| **权限体系** | JWT + AgentToken 双模鉴权、Scope 细粒度权限、管理员 RBAC |
| **后台任务** | 定时数据刷新、Redis 分布式领导锁、心跳续约、智能调度 |
| **社区功能** | 弹幕评论、用户评测、系统通知广播 |
| **管理后台** | 用户管理、会员操作、版本发布、系统通知、数据统计 |
| **第三方集成** | SMTP 邮件、外部 API 封装、令牌桶限速 |

## 技术栈

| 层次 | 技术 |
|---|---|
| **语言** | Go 1.25+ |
| **框架** | Gin 1.12 |
| **数据库** | PostgreSQL + GORM |
| **缓存** | Redis + 进程内 LRU 降级 |
| **认证** | JWT + bcrypt + SHA256 Token |
| **AI** | OpenRouter / Gemini 双引擎 |
| **邮件** | SMTP（gomail.v2） |
| **日志** | log/slog 结构化日志 |
| **中间件** | CORS、全局限流（Redis 滑窗 + 本地降级）、请求日志、访问统计 |

## 项目结构

```
cmd/
└── server/
    └── main.go              # 启动入口
internal/
├── app/                     # Gin Engine 装配
├── bootstrap/               # 依赖注入装配
├── config/                  # 配置管理
├── db/                      # 数据库连接
├── errors/                  # 业务错误码
├── handler/                 # HTTP 处理器
├── middleware/               # 中间件
├── model/                   # 数据模型（11 张表）
├── repository/              # 数据库操作层
├── router/                  # 路由注册（10 组）
├── schema/                  # 请求/响应结构
├── security/                # 安全工具
├── service/                 # 业务逻辑层
└── task/                    # 后台任务
pkg/
├── cache/                   # 缓存层
├── httpx/                   # HTTP 客户端
├── ipx/                     # IP 提取
├── leader/                  # 分布式锁
├── ratelimit/               # 限流器
└── response/                # 统一响应格式
migrations/                  # 数据库迁移
steps/                       # 开发指南
```

## 快速开始

### 前置条件

- Go 1.25+
- PostgreSQL 15+
- Redis 7+

### 配置

创建 `.env`：

```env
PORT=8080
DATABASE_URL=postgres://user:pass@localhost:5432/huahuadb?sslmode=disable
REDIS_ADDR=localhost:6379
SECRET_KEY=your-random-secret-key
```

### 启动

```bash
# 安装依赖
go mod tidy

# 运行
go run cmd/server/main.go

# 验证
curl http://localhost:8080/api/health

# 构建
go build -o huahua-server cmd/server/main.go
```

## 分层架构

```
router → handler → service → repository → gorm.DB
  ↓         ↘          ↘           ↘
schema    errors/    pkg/*      pkg/cache/...
```

- router：只做路由注册
- handler：参数绑定 + 调 service + 返回响应
- service：业务编排 + 缓存 + 协调
- repository：唯一操作数据库的层
- pkg：通用工具包，不引用 internal

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 健康检查 |
| POST | `/api/auth/register` | 注册 |
| POST | `/api/auth/token` | 登录 |
| GET | `/api/auth/me` | 用户信息 |
| GET | `/api/version` | 版本 |
| POST | `/api/jcti/analyze` | 评测 |
| GET/POST/PUT | `/api/agent/*` | Agent 接口 |
| GET | `/api/public/blog-stats` | 统计 |
| POST | `/api/admin/activate_vip` | 管理 |
| GET | `/api/admin/users` | 用户管理 |

完整路由见 `internal/router/`。

## License

[MIT](LICENSE)
