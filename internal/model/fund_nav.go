package model

// FundNav 基金历史净值 → fund_navs 表
type FundNav struct {
	ID       int64   `gorm:"primaryKey;autoIncrement"`
	FundCode string  `gorm:"column:fund_code;index;size:6"`
	Date     string  `gorm:"index;size:10"`      // "2024-01-15"
	Nav      float64 `gorm:"type:numeric(10,4)"` // 单位净值
	Change   float64 `gorm:"type:numeric(7,4)"`  // 涨跌幅 %
}

func (FundNav) TableName() string {
	return "fund_navs"
}
