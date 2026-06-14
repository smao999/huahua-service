package schema

// BatchEstimateRequest POST /api/estimate/batch
type BatchEstimateRequest struct {
	Codes []string `json:"codes" binding:"required,min=1,max=50"`
}

// EstimateItem 批量估值单条结果
type EstimateItem struct {
	FundCode      string  `json:"fund_code"`
	Name          string  `json:"name"`
	Estimate      float64 `json:"estimate"`
	ChangePercent float64 `json:"changePercent"`
	Time          string  `json:"time"`
}

// NavItem 历史净值单条
type NavItem struct {
	Date   string  `json:"date"`
	Nav    float64 `json:"nav"`
	Change float64 `json:"change"`
}

// FundSearchItem 搜索命中项
type FundSearchItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"`
}
