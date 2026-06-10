package model

import "time"

type FundBasicInfo struct {
	Code         string    `gorm:"primaryKey;size:6"`
	Name         string    `gorm:"size:100"`
	Type         string    `gorm:"size:20"`
	ConfirmDays  int       `gorm:"column:confirm_days;default:1"`
	FeesJSON     string    `gorm:"column:fees_json;type:text"`
	HoldingsJSON string    `gorm:"column:holdings_json;type:text"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (FundBasicInfo) TableName() string { return "fund_basic_info" }
