package model

import "time"

type FundBasicInfo struct {
	Code         string    `gorm:"primaryKey;size:6;comment:基金代码"`
	Name         string    `gorm:"size:100;comment:基金名称"`
	Type         string    `gorm:"size:20;comment:基金类型(股票型/混合型等)"`
	ConfirmDays  int       `gorm:"column:confirm_days;default:1;comment:确认天数"`
	FeesJSON     string    `gorm:"column:fees_json;type:text;comment:费率JSON"`
	HoldingsJSON string    `gorm:"column:holdings_json;type:text;comment:持仓JSON"`
	IndustryJSON string    `gorm:"column:industry_json;type:text;comment:行业分布JSON"`
	Sector       string    `gorm:"size:50;default:'';comment:行业板块"`
	EquityPct    *float64  `gorm:"column:equity_pct;comment:股票仓位(%)"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime;comment:更新时间"`
}

func (FundBasicInfo) TableName() string { return "fund_basic_info" }
