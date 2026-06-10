package model

import "time"

// FundSector 基金行业分类 → fund_sectors 表
type FundSector struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	FundCode  string    `gorm:"column:fund_code;uniqueIndex;size:6"`
	Sector    string    `gorm:"size:50"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (FundSector) TableName() string {
	return "fund_sectors"
}
