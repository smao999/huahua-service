package model

import "time"

// FundCalibration 夜估校准参数 → fund_calibrations 表
type FundCalibration struct {
	FundCode      string    `gorm:"column:fund_code;primaryKey;size:6"`
	Beta          float64   `gorm:"type:numeric(10,6);default:1.0"`
	Alpha         float64   `gorm:"type:numeric(10,6);default:0.0"`
	SampleCount   int       `gorm:"column:sample_count;default:0"`
	CoverageRatio float64   `gorm:"column:coverage_ratio;type:numeric(7,2);default:0.0"`
	HoldingsHash  string    `gorm:"column:holdings_hash;size:16"`
	LastUpdated   time.Time `gorm:"column:last_updated;autoUpdateTime"`
}

func (FundCalibration) TableName() string {
	return "fund_calibrations"
}
