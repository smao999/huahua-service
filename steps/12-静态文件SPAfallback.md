# Step 12：静态文件 & SPA fallback

> **目标**：让 Go 服务同时托管前端静态文件，并实现 SPA 单页应用的路由回退。

---

## 12.1 Python 源码

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 python（27 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```python
# core/app_factory.py 中
import os

# 静态文件目录
_UPLOADS_DIR = os.path.join(settings.DATA_DIR, "uploads")  # 用户上传文件
_BUY_DIR = os.path.join(settings.DATA_DIR, "buy")          # 购买记录
_FRONTEND_DIST = os.path.join(os.path.dirname(os.path.dirname(__file__)), "dist")

# 确保目录存在
for _d in [_UPLOADS_DIR, _BUY_DIR]:
    os.makedirs(_d, exist_ok=True)

# 挂载静态文件
app.mount("/static/uploads", StaticFiles(directory=_UPLOADS_DIR), name="static_uploads")
app.mount("/static/buy", StaticFiles(directory=_BUY_DIR), name="static_buy")

# 前端 SPA 回退
if os.path.isdir(_FRONTEND_DIST):
    app.mount("/assets", StaticFiles(directory=os.path.join(_FRONTEND_DIST, "assets")))
    
    @app.get("/{full_path:path}")
    def spa_fallback(full_path: str):
        # /api/ 开头的未知路径返回 JSON 404
        if full_path.startswith("api/"):
            return JSONResponse(404, {"detail": "API endpoint not found"})
        # 其他所有路径返回 index.html（让前端路由处理）
        return FileResponse(os.path.join(_FRONTEND_DIST, "index.html"))
```
</details>


---

## 12.2 Go 实现

<details>
<summary>

┌════════════════════════════════════════════════┐
│  📂 go    （52 行）                               │
│  ─────────────────────────────────────────────  │
│  👆 点击此处展开 / 收起                              │
└════════════════════════════════════════════════┘
</summary>

```go
// internal/app/app.go 中追加
import (
    "net/http"
    "os"
    "strings"
    "github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
    r := gin.New()
    r.Use(gin.Recovery())

    // 1. 静态文件：上传目录
    // 对应 Python app.mount("/static/uploads", ...)
    uploadDir := cfg.DataDir + "/uploads"
    os.MkdirAll(uploadDir, 0755)
    r.Static("/static/uploads", uploadDir)

    // 对应 Python /static/buy
    buyDir := cfg.DataDir + "/buy"
    os.MkdirAll(buyDir, 0755)
    r.Static("/static/buy", buyDir)

    // 2. 注册 API 路由
    h := handler.NewHandlers()
    router.Register(r, h)

    // 3. SPA 回退（== Python 的 spa_fallback）
    // 必须放在路由注册之后——Gin 的 NoRoute 只匹配没被注册的路由
    r.NoRoute(func(c *gin.Context) {
        path := c.Request.URL.Path
        
        // API 404 → JSON
        if strings.HasPrefix(path, "/api/") {
            c.JSON(http.StatusNotFound, gin.H{"detail": "API endpoint not found"})
            return
        }
        
        // 前端静态文件 → index.html（SPA 兜底）
        // Python: FileResponse(os.path.join(_FRONTEND_DIST, "index.html"))
        indexFile := "./dist/index.html"
        if _, err := os.Stat(indexFile); err == nil {
            c.File(indexFile)
            return
        }
        
        // 连 index.html 都没有 → 404
        c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
    })

    return r
}
```
</details>


**Gin NoRoute 的行为**：
- 只匹配**没有被任何路由注册**的路径
- 如果 `GET /api/health` 在路由里注册了，访问 `/api/health` **不会**走到 NoRoute
- 访问 `/api/nonexistent` 才会走到 NoRoute
- 访问 `/some-unknown-page`（前端路径）也走到 NoRoute → 返回 index.html → 前端路由接管

---

## 本级小结

- ✅ `r.Static("/static/uploads", dir)` 托管上传文件
- ✅ `r.NoRoute` 实现 SPA 回退
- ✅ /api/ 前缀的 404 返回 JSON，其他返回 index.html
