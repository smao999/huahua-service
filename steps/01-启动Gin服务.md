# Step 1：启动一个最简 Gin 服务

> **目标**：不连数据库、不读配置文件，先让 Gin 跑起来，能看到 HTTP 响应。
> 这是最快获得正向反馈的一步——你马上就能用 curl 访问自己的服务。

---

## 1.1 先理清依赖顺序

**之前版本**的 Step 1 一口气把 config / db / model / cache 全写了，问题是：
- PostgreSQL 还没装好怎么办？
- Redis 没启动怎么办？
- 这些配置项看不明白怎么办？

**正确做法**：先让一个空的 Gin 跑起来，哪怕只返回 `Hello Gin`，你也有"我能跑了"的信心。
后面再逐步加配置、加数据库、加业务。

---

## 1.2 创建 main.go——启动入口

```go
// cmd/server/main.go
package main

import (
    "log"
    "net/http"

    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()  // 带 Logger 和 Recovery 中间件的默认引擎
    r.GET("/ping", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"message": "pong"})
    })
    
    log.Println("服务启动在 :8080")
    r.Run(":8080")
}
```

**这行代码做了什么**：
- `gin.Default()`：创建一个 Gin 引擎，自带两个中间件（Logger 打印请求日志，Recovery 捕获 panic）
- `r.GET("/ping", handler)`：注册一个 GET 路由
- `r.Run(":8080")`：监听 8080 端口，阻塞等待请求

---

## 1.3 验证

```bash
# 1. 安装依赖（第一次会下载 Gin 框架）
cd /Users/xqs/Documents/datasource/huahua-service
go mod tidy

# 2. 运行
go run cmd/server/main.go

# 3. 另一个终端请求
curl http://localhost:8080/ping
# 输出：{"message":"pong"}
```

**成功了吗？** 如果看到 `pong`，恭喜——你的第一个 Gin 服务跑起来了！

**注意**：`gin.Default()` 自带 Logger 中间件，终端会打印：
```
[GIN] 2026/06/09 - 10:00:00 | 200 | 0s | ::1 | GET "/ping"
```

---

## 1.4 升级：改成健康检查模式

把 `/ping` 改成 `/api/health`，准备迎接后续步骤的扩展：

```go
// cmd/server/main.go
package main

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()
    
    // 健康检查——最简单的 handler，不依赖任何外部服务
    r.GET("/api/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "ok",
        })
    })
    
    r.Run(":8080")
}
```

验证：
```bash
curl http://localhost:8080/api/health
# {"status":"ok"}
```

---

## 1.5 项目结构检查

这步只需要这些文件：

```
huahua-service/
├── cmd/server/main.go     # ← 刚写的启动入口
├── go.mod                 # ← go mod init 时自动生成
├── go.sum                 # ← go mod tidy 时自动生成
```

所有其他文件夹（internal/、pkg/ 等）暂时用不上——后面每步逐步增加。

---

## 1.6 对比之前版本

旧版 Step 1 一口气写了 8 个文件 + 1714 行内容。
这步只写 1 个文件 + ~30 行代码——但你已经能看到输出。

**这就是渐进式开发的精髓**：每一步都能验证，每一步向前一小步。

---

## 本步总结

- ✅ Gin 服务跑起来了
- ✅ /api/health 返回正常
- ✅ 0 个外部依赖（不需要数据库、不需要 Redis）

**下一步**：Step 2 —— 做好看的项目结构，把 handler 拆到单独的文件里。
