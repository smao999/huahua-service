# Step 5：基础业务（Fund Service）

> **目标**：用 akshare 客户端 + 数据库构建 Fund Service，完成"获取基金估值"的业务编排。

---

## 5.1 Python 源码：routers/fund.py 是怎么写基金的

Python 把基金相关路由分散在两个地方：

**routers/fund.py**（处理 HTTP 请求、调 service）：

```python
from fastapi import APIRouter, Depends, HTTPException
from schemas import EstimateRequest
from services.fund_service import FundService
from security import check_rate_limit, get_client_ip, is_valid_fund_code

router = APIRouter(prefix="/api", tags=["fund"])

@router.post("/estimate/batch")
async def api_estimate(payload: EstimateRequest, request: Request):
    """批量获取基金实时估值"""
    codes = payload.codes  # ["000001", "110011", ...]
    try:
        result = await FundService.get_estimates(codes)
        return {"data": result, "truncated": False, "limit": 50}
    except Exception as e:
        return JSONResponse(500, content={"detail": "处理失败"})

@router.get("/fund/{code}")
async def api_detail(code: str):
    """基金详情（含净值、费率、持仓、胜率表）"""
    code = await _ensure_known_fund_code(code)
    detail = await FundService.get_fund_detail_full(code)
    
    # 顺带算胜率表（弱依赖，失败降级）
    win_rate = await win_rate_service.get_or_compute(code, confirm_days=1)
    
    return {**detail, "winRateTable": win_rate}
```

**services/fund_service.py**（业务逻辑）：

```python
class FundService:
    """基金业务逻辑类"""
    
    @staticmethod
    async def get_estimates(codes: list[str]) -> list[dict]:
        """批量获取实时估值"""
        estimates = []
        for code in codes:
            try:
                est = await AkshareService.get_estimate(code)
                estimates.append({
                    "fund_code": code,
                    "name": est["name"],
                    "estimate": float(est["gsz"]),
                    "changePercent": float(est["gszzl"]),
                    "time": est["gztime"],
                })
            except Exception:
                continue  # 单个基金失败不影响其他
        return estimates
    
    @staticmethod
    async def get_fund_detail_full(code: str) -> dict:
        """获取基金完整详情"""
        # 1. 查数据库基本信息
        info = db.query(FundBasicInfo).filter(FundBasicInfo.code == code).first()
        
        # 2. 从 akshare 拿实时估值
        estimate = await AkshareService.get_estimate(code)
        
        # 3. 查历史净值
        history = await AkshareService.fetch_fund_history_akshare(code)
        
        # 4. 组装结果
        return {
            "code": code,
            "name": info.name if info else estimate["name"],
            "type": info.type if info else "",
            "estimate": float(estimate["gsz"]),
            "changePercent": float(estimate["gszzl"]),
            "history": history[:30],  # 最近 30 条
            "fees": json.loads(info.fees_json) if info and info.fees_json else {},
        }
```

---

## 5.2 理解 Go 的 Service 层

Go 的 Service 层是"业务编排"的场所，它：
- 调 akshare client 获取上游数据
- 调 repository 查数据库
- 组合、计算、缓存，返回结果
- 不处理 HTTP 请求（那是 handler 的事）
- 不直接写 SQL（那是 repository 的事）

## 5.3 Go 实现

### 5.3.1 FundBasicInfo Repository

```go
// internal/repository/fund_basic_info_repo.go
package repository

import (
    "context"
    "huahua-service/internal/model"
    "gorm.io/gorm"
)

// FundBasicInfoRepository 基金基础信息接口
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

// GetByCode 按基金代码查基本信息
// 对应 Python: db.query(FundBasicInfo).filter(FundBasicInfo.code == code).first()
func (r *fundBasicInfoRepo) GetByCode(ctx context.Context, code string) (*model.FundBasicInfo, error) {
    var info model.FundBasicInfo
    err := r.db.WithContext(ctx).Where("code = ?", code).First(&info).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil  // 查不到返回 nil——不是错误
        }
        return nil, err
    }
    return &info, nil
}

// FindAll 获取所有基金
func (r *fundBasicInfoRepo) FindAll(ctx context.Context) ([]model.FundBasicInfo, error) {
    var funds []model.FundBasicInfo
    err := r.db.WithContext(ctx).Find(&funds).Error
    return funds, err
}
```

### 5.3.2 FundNav Repository

```go
// internal/repository/fund_nav_repo.go
package repository

import (
    "context"
    "time"
    "huahua-service/internal/model"
    "gorm.io/gorm"
)

type FundNavRepository interface {
    GetByCode(ctx context.Context, code string, limit int) ([]model.FundNav, error)
    GetByDate(ctx context.Context, code string, date time.Time) (*model.FundNav, error)
}

type fundNavRepo struct {
    db *gorm.DB
}

func NewFundNavRepository(db *gorm.DB) FundNavRepository {
    return &fundNavRepo{db: db}
}

func (r *fundNavRepo) GetByCode(ctx context.Context, code string, limit int) ([]model.FundNav, error) {
    var navs []model.FundNav
    err := r.db.WithContext(ctx).
        Where("fund_code = ?", code).
        Order("date DESC").
        Limit(limit).
        Find(&navs).Error
    return navs, err
}

func (r *fundNavRepo) GetByDate(ctx context.Context, code string, date time.Time) (*model.FundNav, error) {
    dateStr := date.Format("2006-01-02")
    var nav model.FundNav
    err := r.db.WithContext(ctx).
        Where("fund_code = ? AND date = ?", code, dateStr).
        First(&nav).Error
    if err != nil {
        return nil, err
    }
    return &nav, nil
}
```

### 5.3.3 Fund Service

```go
// internal/service/fund/fund_service.go
package fund

import (
    "context"
    "fmt"

    "huahua-service/internal/repository"
    "huahua-service/internal/service/akshare"
)

// FundService 基金业务编排
// 对应 Python services/fund_service.py 的 FundService 类
type FundService struct {
    akshareClient *akshare.Client
    fundRepo      repository.FundBasicInfoRepository
    navRepo       repository.FundNavRepository
}

func NewFundService(
    akshareClient *akshare.Client,
    fundRepo repository.FundBasicInfoRepository,
    navRepo repository.FundNavRepository,
) *FundService {
    return &FundService{
        akshareClient: akshareClient,
        fundRepo:      fundRepo,
        navRepo:       navRepo,
    }
}

// EstimateResult 估值结果
// json tag 必须和 Python 返回的字段名一模一样！
type EstimateResult struct {
    FundCode       string  `json:"fund_code"`
    Name           string  `json:"name"`
    Estimate       float64 `json:"estimate"`
    ChangePercent  float64 `json:"changePercent"`  // Python 返回 changePercent
    Time           string  `json:"time"`
}

// GetEstimates 批量获取实时估值
// 对应 Python FundService.get_estimates()
//
// Python 代码对照：
//   for code in codes:
//       try:
//           est = await AkshareService.get_estimate(code)
//           estimates.append({...})
//       except Exception:
//           continue
func (s *FundService) GetEstimates(ctx context.Context, codes []string) ([]EstimateResult, error) {
    results := make([]EstimateResult, 0, len(codes))

    for _, code := range codes {
        // 1. 调 akshare 拿实时数据
        est, err := s.akshareClient.GetFundEstimate(code)
        if err != nil {
            continue // 单个失败不影响其他
        }

        // 2. 从数据库补全基金信息
        info, _ := s.fundRepo.GetByCode(ctx, code)

        result := EstimateResult{
            FundCode:      est.FundCode,
            Name:          est.Name,
            Estimate:      est.Estimate,
            ChangePercent: est.EstimatePercent,
            Time:          est.EstimateTime,
        }

        // Python 的 info.name if info else estimate["name"]
        if info != nil {
            result.Name = info.Name
        }

        results = append(results, result)
    }

    return results, nil
}

// FundDetailResult 基金详情（用于 GET /api/fund/{code}）
type FundDetailResult struct {
    Code          string            `json:"code"`
    Name          string            `json:"name"`
    Type          string            `json:"type"`
    Estimate      float64           `json:"estimate"`
    ChangePercent float64           `json:"changePercent"`
    Fees          map[string]interface{} `json:"fees"`
    WinRateTable  interface{}       `json:"winRateTable,omitempty"`
    WinRateStatus string            `json:"winRateStatus,omitempty"`
}

// GetFundDetail 获取基金完整详情
// 对应 Python get_fund_detail_full()
func (s *FundService) GetFundDetail(ctx context.Context, code string) (*FundDetailResult, error) {
    // 1. 查数据库基本信息
    info, _ := s.fundRepo.GetByCode(ctx, code)

    // 2. 拿实时估值
    est, err := s.akshareClient.GetFundEstimate(code)
    if err != nil {
        return nil, fmt.Errorf("获取估值失败: %w", err)
    }

    result := &FundDetailResult{
        Code:          code,
        Name:          est.Name,
        Estimate:      est.Estimate,
        ChangePercent: est.EstimatePercent,
        Fees:          make(map[string]interface{}),
    }

    if info != nil {
        result.Name = info.Name
        result.Type = info.Type
        // 解析费率 JSON（Python 的 json.loads(info.fees_json) if info.fees_json else {}）
        if info.FeesJSON != "" {
            // 用 json.Unmarshal 解析 info.FeesJSON 到 result.Fees
        }
    }

    return result, nil
}
```

### 5.3.4 Fund Handler

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

// BatchEstimate POST /api/estimate/batch
// 对应 Python api_estimate()
func (h *FundHandler) BatchEstimate(c *gin.Context) {
    var req struct {
        Codes []string `json:"codes" binding:"required,max=50"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
        return
    }

    results, err := h.fundService.GetEstimates(c.Request.Context(), req.Codes)
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

// Detail GET /api/fund/:code
func (h *FundHandler) Detail(c *gin.Context) {
    code := c.Param("code")
    if len(code) != 6 {
        c.JSON(http.StatusBadRequest, gin.H{"detail": "基金代码格式无效"})
        return
    }

    detail, err := h.fundService.GetFundDetail(c.Request.Context(), code)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"detail": err.Error()})
        return
    }

    c.JSON(http.StatusOK, detail)
}
```

### 5.3.5 更新路由注册

```go
// internal/router/fund.go
func registerFund(rg *gin.RouterGroup, h *handler.FundHandler) {
    g := rg.Group("")
    g.POST("/estimate/batch", h.BatchEstimate)    // POST /api/estimate/batch
    g.GET("/fund/:code", h.Detail)                 // GET /api/fund/:code
}
```

---

## 5.4 验证

```bash
# 批量估值
curl -X POST http://localhost:8080/api/estimate/batch \
  -H "Content-Type: application/json" \
  -d '{"codes":["000001","110011"]}'
# 期望响应（Python 格式完全一致）：
# {"data":[{"fund_code":"000001","name":"上证...","estimate":1.234,"changePercent":0.56,"time":"15:00"}],"truncated":false,"limit":50}

# 基金详情
curl http://localhost:8080/api/fund/000001
```

---

## 本级小结

- ✅ FundBasicInfo Repository：按代码查基金基本信息
- ✅ FundNav Repository：查历史净值
- ✅ Fund Service：组合 akshare + 数据库，返回估值结果
- ✅ Fund Handler：处理 HTTP 请求
- ✅ 路由注册：打开 /api/estimate/batch 和 /api/fund/:code

**下一步**：Step 6 —— 把 Fund 和 Market 的所有 handler 接入路由。

---

## 补充：WinRateService——基金胜率表（从 Step 14 移入）

基金详情页需要展示"胜率表"（T+20 乖离率胜率）。

### Python 源码

```python
# services/win_rate_service.py
def get_or_compute(fund_code, confirm_days=1):
    # 1. 取历史净值（最近 5 年）
    navs = db.query(FundNav).filter(FundNav.fund_code == fund_code).order_by(FundNav.date.desc()).limit(1250).all()
    # 2. 计算 MA20、MA60
    # 3. 计算 Bias20 = (close - MA20) / MA20 × 100
    # 4. 按 Bias20 分桶（每 2% 一档）
    # 5. 统计每档后续 T+20 上涨概率
    # 6. Redis 缓存：fund:winrate:v2:{code}:d{confirm_days}
```

### Go 实现

```go
// internal/service/winrate/service.go — 新建文件
package winrate

import (
    "huahua-service/internal/repository"
)

type Service struct {
    navRepo repository.FundNavRepository
}

func NewService(navRepo repository.FundNavRepository) *Service {
    return &Service{navRepo: navRepo}
}

// Result 胜率表结果
type Result struct {
    WinRate     float64 `json:"winRate"`     // 当前 Bias20 对应的历史胜率
    TotalDays   int     `json:"totalDays"`   // 总样本数
    UpDays      int     `json:"upDays"`      // 上涨天数
    DownDays    int     `json:"downDays"`    // 下跌天数
    AvgReturn   float64 `json:"avgReturn"`   // 平均收益 %
}

// GetWinRate 计算基金胜率
func (s *Service) GetWinRate(code string, confirmDays int) (*Result, error) {
    // 1. 取最近 1250 条历史净值
    records, err := s.navRepo.GetRecent(code, 1250)
    if err != nil { return nil, err }
    if len(records) < 30 { return nil, nil }  // 样本不足

    // 2. 计算
    prices := extractPrices(records)
    winRates := computeBiasWinRates(prices, 20)

    // 3. 找到当前 Bias20 对应的胜率
    currentBias := bias20(prices)
    result := findWinRate(winRates, currentBias)

    return result, nil
}

// 辅助函数（略，纯数值计算，跟 Python 一样）
func extractPrices(navs interface{}) []float64 { return nil }
func bias20(prices []float64) float64 { return 0 }
func computeBiasWinRates(prices []float64, window int) map[float64]float64 { return nil }
func findWinRate(winRates map[float64]float64, bias float64) *Result { return nil }
```

### 在 Fund Detail 中接入

```go
// 在 FundService.GetFundDetail 中补充
winRate, err := s.winRateService.GetWinRate(code, confirmDays)
if err != nil {
    result.WinRateStatus = "unavailable"
} else if winRate == nil {
    result.WinRateStatus = "computing"
} else {
    result.WinRateTable = winRate
    result.WinRateStatus = "ready"
}
```
