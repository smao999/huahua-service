package model

// FundNav 基金历史净值 → fund_navs 表
type FundNav struct {
	ID       int64   `gorm:"primaryKey;autoIncrement;comment:主键"`
	FundCode string  `gorm:"column:fund_code;index;size:6;comment:基金代码"`
	Date     string  `gorm:"index;size:10;comment:净值日期(YYYY-MM-DD)"`
	Nav      float64 `gorm:"type:numeric(10,4);comment:单位净值"`
	Change   float64 `gorm:"type:numeric(7,4);comment:涨跌幅(%)"`
}

func (FundNav) TableName() string {
	return "fund_navs"
}
