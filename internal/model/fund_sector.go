package model

import "time"

// FundSector 基金行业分类 → fund_sectors 表
type FundSector struct {
	ID        int64     `gorm:"primaryKey;autoIncrement;comment:主键"`
	FundCode  string    `gorm:"column:fund_code;uniqueIndex;size:6;comment:基金代码"`
	Sector    string    `gorm:"size:50;comment:行业分类"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;comment:更新时间"`
}

func (FundSector) TableName() string {
	return "fund_sectors"
}
