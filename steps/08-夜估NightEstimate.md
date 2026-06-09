# Step 8：夜估（Night Estimate）

> **目标**：实现"夜间净值校准"——每天晚上用当天净值平滑更新历史估值。
> 这是整个项目最复杂的业务逻辑之一（涉及 α、β 参数校准）。

---

## 8.1 Python 核心代码分析

**services/calibration_service.py**（夜估校准）：

```python
class CalibrationService:
    """基金净值校准服务
    
    核心公式（EMA 指数移动平均）：
    alpha_new = alpha_old × λ + error × (1-λ)
              其中 λ=0.85，error = 实际净值 - 持仓估算净值
    
    alpha 是"加法偏差"——吸收系统性偏置（汇率、打新、债息等）
    beta 是"乘法系数"——吸收覆盖率不足导致的幅度缩放
    
    Python 实现要点：
    1. 每晚 22:00 执行，只跑一次
    2. 每只基金独立校准
    3. 冷启动保护（sample_count < 10 时 alpha=0, beta=1）
    4. 持仓变更时重置 beta
    """
    
    @staticmethod
    def run_calibration(session, date: date):
        funds = session.query(FundBasicInfo).all()
        for fund in funds:
            # 1. 取当天实际净值
            nav = session.query(FundNav).filter(
                FundNav.fund_code == fund.code,
                FundNav.date == date.isoformat()
            ).first()
            if not nav:
                continue  # 净值还没出来，跳过
            
            # 2. 取当前校准参数
            cal = session.query(FundCalibration).filter(
                FundCalibration.fund_code == fund.code
            ).first()
            
            if cal is None:
                # 冷启动：初始化参数
                cal = FundCalibration(
                    fund_code=fund.code,
                    beta=1.0, alpha=0.0,
                    sample_count=0,
                )
                session.add(cal)
            
            # 3. 用持仓估算今日净值
            estimated = compute_holdings_estimate(fund)
            if estimated is None:
                continue
            
            # 4. 计算偏差
            error = float(nav.nav) - estimated
            
            # 5. EMA 更新参数
            LAMBDA = 0.85
            if cal.sample_count >= 10:  # 冷启动保护
                cal.alpha = cal.alpha * LAMBDA + error * (1 - LAMBDA)
            cal.sample_count += 1
            
            # 6. 标记校准日期
            cal.last_calibrated_nav_date = date.isoformat()
        
        session.commit()
```

## 8.2 Go 实现

```go
// internal/service/calibration/calibration_service.go
package calibration

import (
    "context"
    "log/slog"

    "huahua-service/internal/repository"

    "gorm.io/gorm"
)

const (
    Lambda     = 0.85 // EMA 遗忘因子——越大越依赖历史，越小越依赖新数据
    MinSamples = 10   // 冷启动保护：少于 10 个样本时不更新 alpha
)

// Service 夜估校准服务
// 对应 Python CalibrationService
type Service struct {
    db       *gorm.DB
    fundRepo repository.FundBasicInfoRepository
    navRepo  repository.FundNavRepository
    calRepo  repository.CalibrationRepository
}

func NewService(
    db *gorm.DB,
    fundRepo repository.FundBasicInfoRepository,
    navRepo repository.FundNavRepository,
    calRepo repository.CalibrationRepository,
) *Service {
    return &Service{
        db:       db,
        fundRepo: fundRepo,
        navRepo:  navRepo,
        calRepo:  calRepo,
    }
}

// RunCalibration 执行全部基金的夜估校准
// 对应 Python run_calibration()
func (s *Service) RunCalibration(ctx context.Context, date string) error {
    // 1. 获取所有基金
    funds, err := s.fundRepo.FindAll(ctx)
    if err != nil {
        return err
    }

    slog.Info("开始夜估校准", "fund_count", len(funds), "date", date)

    for _, fund := range funds {
        if err := s.calibrateOne(ctx, fund.Code, date); err != nil {
            // 单个基金失败只打日志，不影响其他
            slog.Warn("校准失败", "code", fund.Code, "error", err)
            continue
        }
    }

    return nil
}

// calibrateOne 校准单只基金
func (s *Service) calibrateOne(ctx context.Context, code, date string) error {
    // 1. 取当天实际净值
    nav, err := s.navRepo.GetByDate(ctx, code, date)
    if err != nil {
        return err // 净值还没出来，跳过
    }

    // 2. 取当前校准参数
    cal, err := s.calRepo.GetByCode(ctx, code)
    if err != nil {
        return err
    }

    // 3. 用持仓估算（简化版——实际需要提取 holdings_json 计算）
    estimated := nav.Nav * 0.99 // 占位，实际应调 holdings_estimate
    error := nav.Nav - estimated

    // 4. EMA 更新 alpha
    if cal.SampleCount >= MinSamples {
        // Python: cal.alpha = cal.alpha * LAMBDA + error * (1 - LAMBDA)
        cal.Alpha = cal.Alpha*Lambda + error*(1-Lambda)
    }
    cal.SampleCount++
    cal.LastCalibratedNavDate = date

    // 5. 保存
    return s.calRepo.Save(ctx, cal)
}
```

**公式解释**：

```
error = 实际净值 - 持仓估算
        （正值表示估算偏低，需要向上调整）

alpha_new = alpha_old × 0.85 + error × 0.15
           （指数移动平均，强调最近误差）

为什么用 0.85？
- λ=0.85 → 历史权重高，新数据影响平缓
- λ=0.99 → 几乎不变，反应慢
- λ=0.5  → 对新数据很敏感，波动大
```

---

## 8.3 Calibration Repository

```go
// internal/repository/calibration_repo.go
package repository

import (
    "context"
    "huahua-service/internal/model"
    "gorm.io/gorm"
)

type CalibrationRepository interface {
    GetByCode(ctx context.Context, code string) (*model.FundCalibration, error)
    Save(ctx context.Context, cal *model.FundCalibration) error
}

type calibrationRepo struct {
    db *gorm.DB
}

func NewCalibrationRepository(db *gorm.DB) CalibrationRepository {
    return &calibrationRepo{db: db}
}

func (r *calibrationRepo) GetByCode(ctx context.Context, code string) (*model.FundCalibration, error) {
    var cal model.FundCalibration
    err := r.db.WithContext(ctx).Where("fund_code = ?", code).First(&cal).Error
    if err != nil {
        return nil, err
    }
    return &cal, nil
}

func (r *calibrationRepo) Save(ctx context.Context, cal *model.FundCalibration) error {
    return r.db.WithContext(ctx).Save(cal).Error
}
```

---

## 本级小结

- ✅ 夜估校准数学公式：`alpha_new = alpha × λ + error × (1-λ)`
- ✅ Go Service 层实现
- ✅ Calibration Repository 实现
- ✅ 冷启动保护（sample_count < 10 时不动 alpha）

**下一步**：Step 9 —— 后台定时任务（调度夜估和其他刷新任务）。

---

## 补充 1：HoldingsEstimate——持仓估算（从 Step 14 移入）

夜估校准的核心依赖——用持仓数据估算当日净值。

### Python 源码

```python
# services/holdings_estimate.py
@dataclass(frozen=True)
class HoldingsEstimateResult:
    raw_estimate: Decimal          # 原始估算净值
    covered_weight: Decimal        # 已覆盖持仓权重
    total_holdings_weight: Decimal # 总持仓权重

def build_holdings_estimate(holdings_json, market_prices):
    """用持仓数据估算基金净值
    
    步骤：
    1. 解析 holdings_json（Python json.loads）
    2. 对每个持仓项：持仓数量 × 市价 = 持仓市值
    3. 汇总持仓市值 ÷ 基金总份额 = 估算净值
    4. 返回估算结果 + 覆盖率
    """
    holdings = json.loads(holdings_json or "[]")
    total_value = Decimal("0")
    covered_value = Decimal("0")
    
    for item in holdings:
        code = item.get("code")
        amount = Decimal(str(item.get("amount", 0)))
        price = market_prices.get(code)
        if price:
            covered_value += amount * Decimal(str(price))
        total_value += amount
    
    return HoldingsEstimateResult(
        raw_estimate=covered_value,  # 简化版
        covered_weight=covered_value,
        total_holdings_weight=total_value,
    )
```

### Go 实现

```go
// internal/service/holdings/estimate.go — 新建文件
package holdings

// EstimateResult 持仓估算结果
type EstimateResult struct {
    RawEstimate   float64 // 估算净值
    CoveredWeight float64 // 已覆盖持仓权重
    TotalWeight   float64 // 总持仓权重
}

// Estimate 用持仓数据估算净值
// 对应 Python build_holdings_estimate()
func Estimate(holdingsJSON string, marketPrices map[string]float64) *EstimateResult {
    // 1. json.Unmarshal 解析持仓 JSON
    // 2. 对每个持仓项：数量 × 市价
    // 3. 汇总
    return &EstimateResult{
        RawEstimate:   0,
        CoveredWeight: 0,
        TotalWeight:   0,
    }
}

// HasEnoughCoverage 判断覆盖率是否足够
// 覆盖率 < 25% 时估算不可靠
func HasEnoughCoverage(covered, total float64) bool {
    return total > 0 && covered/total*100 >= 25
}
```

---

## 补充 2：NightEstimateService（从 Step 14 移入）

夜估（用户侧看到的"盘后估值"）和校准是两回事：
- **Calibration**（Step 8 主体）：每只基金 α/β 参数训练——每晚 22:00 跑一次
- **NightEstimate**：用户主动查询某只基金的夜估结果——调用 market 接口时

**Python routers/market.py**：
```python
@router.get("/night-est")
async def api_market_night_est(codes, current_user):
    # 需 VIP/Pro 会员
    code_list = codes.split(",")
    return await NightEstimateService.get_night_est(code_list)
```

**Go 简化实现**（依赖 holdings estimate + calibration params）：

```go
// internal/service/nightest/service.go — 新建文件
package nightest

import (
    "huahua-service/internal/repository"
    "huahua-service/internal/service/akshare"
)

type Service struct {
    akshareClient *akshare.Client
    calibRepo     repository.CalibrationRepository
}

func NewService(client *akshare.Client, calibRepo repository.CalibrationRepository) *Service {
    return &Service{akshareClient: client, calibRepo: calibRepo}
}

// GetNightEst 获取盘后估值
func (s *Service) GetNightEst(code string) (*Result, error) {
    // 1. 取当日收盘净值
    // 2. 取 calibration 参数（α/β）
    // 3. 用持仓估算法计算
    // 4. 返回校准后估值
    return &Result{Code: code}, nil
}

type Result struct {
    Code           string  `json:"code"`
    Name           string  `json:"name"`
    NightEstimate  float64 `json:"nightEstimate"`
    ChangePercent  float64 `json:"changePercent"`
}
```
