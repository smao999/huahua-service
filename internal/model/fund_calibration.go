package model

import "time"

// FundCalibration 夜估校准参数 → fund_calibrations 表
type FundCalibration struct {
	FundCode      string    `gorm:"column:fund_code;primaryKey;size:6;comment:基金代码"`
	Beta          float64   `gorm:"type:numeric(10,6);default:1.0;comment:贝塔系数(市场敏感度)"`
	Alpha         float64   `gorm:"type:numeric(10,6);default:0.0;comment:阿尔法系数(超额收益)"`
	SampleCount   int       `gorm:"column:sample_count;default:0;comment:样本数量"`
	CoverageRatio float64   `gorm:"column:coverage_ratio;type:numeric(7,2);default:0.0;comment:持仓覆盖率(%)"`
	HoldingsHash  string    `gorm:"column:holdings_hash;size:16;comment:持仓数据哈希"`
	LastUpdated   time.Time `gorm:"column:last_updated;autoUpdateTime;comment:最后更新时间"`
}

func (FundCalibration) TableName() string {
	return "fund_calibrations"
}
