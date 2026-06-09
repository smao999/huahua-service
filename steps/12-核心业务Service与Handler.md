# Step 12：核心业务 Service + Handler

> **目标**：实现 FundService、MarketService 等业务逻辑，并注册对应的 HTTP 路由。
> 参考 `steps-v1/05-基础业务FundService.md` + `steps-v1/06-handler接入.md`。

---

## 12.1 FundService——业务编排

```go
// internal/service/fund/fund_service.go
package fund

import (
    "context"
    "huahua-service/internal/repository"
    "huahua-service/internal/service/akshare"
)

type FundService struct {
    akshare *akshare.Client
    fundRepo repository.FundBasicInfoRepository
    navRepo  repository.FundNavRepository
}

func NewService(client *akshare.Client, fundRepo repository.FundBasicInfoRepository, navRepo repository.FundNavRepository) *FundService {
    return &FundService{akshare: client, fundRepo: fundRepo, navRepo: navRepo}
}

type EstimateResult struct {
    FundCode      string  `json:"fund_code"`
    Name          string  `json:"name"`
    Estimate      float64 `json:"estimate"`
    ChangePercent float64 `json:"changePercent"`
    Time          string  `json:"time"`
}

func (s *FundService) GetEstimate(ctx context.Context, code string) (*EstimateResult, error) {
    raw, err := s.akshare.GetFundEstimate(code)
    if err != nil {
        return nil, err
    }
    info, _ := s.fundRepo.GetByCode(ctx, code)
    name := raw.Name
    if info != nil {
        name = info.Name
    }
    return &EstimateResult{
        FundCode: raw.FundCode, Name: name,
        Estimate: raw.Estimate, ChangePercent: raw.EstimatePercent, Time: raw.EstimateTime,
    }, nil
}
```

## 12.2 Fund Handler

```go
// internal/handler/fund.go
package handler

import (
    "net/http"
    "huahua-service/internal/service/fund"
    "github.com/gin-gonic/gin"
)

type FundHandler struct {
    service *fund.FundService
}

func NewFundHandler(service *fund.FundService) *FundHandler {
    return &FundHandler{service: service}
}

func (h *FundHandler) Detail(c *gin.Context) {
    code := c.Param("code")
    result, err := h.service.GetEstimate(c.Request.Context(), code)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
        return
    }
    c.JSON(http.StatusOK, result)
}
```

## 12.3 注册路由

```go
// internal/router/fund.go
func registerFund(rg *gin.RouterGroup, h *handler.FundHandler) {
    g := rg.Group("")
    g.GET("/fund/:code", h.Detail)  // GET /api/fund/:code
}
```

## 12.4 在 bootstrap 中装配

```go
// internal/bootstrap/bootstrap.go（新增）
package bootstrap

import (
    "huahua-service/internal/db"
    "huahua-service/internal/handler"
    "huahua-service/internal/repository"
    "huahua-service/internal/service/akshare"
    "huahua-service/internal/service/fund"
)

type Dependencies struct {
    Handlers   *handler.Handlers
}

func Init() *Dependencies {
    userRepo := repository.NewUserRepository(db.GORM)
    fundRepo := repository.NewFundBasicInfoRepository(db.GORM)
    navRepo := repository.NewFundNavRepository(db.GORM)
    akshareClient := akshare.NewClient()
    fundService := fund.NewService(akshareClient, fundRepo, navRepo)
    
    h := handler.NewHandlers(
        handler.WithFund(fundService),
    )
    return &Dependencies{Handlers: h}
}
```

## 12.5 验证

```bash
curl http://localhost:8080/api/fund/000001
```

---

## 本步总结

- ✅ FundService（业务编排）
- ✅ FundHandler（HTTP 处理）
- ✅ Bootstrap 依赖装配
- ✅ 路由注册
