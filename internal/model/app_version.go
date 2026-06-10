package model

import "time"

// AppVersion 应用版本管理 → app_versions 表
type AppVersion struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	Version     string    `gorm:"uniqueIndex;size:20"` // "1.6"
	Changelog   string    `gorm:"type:text;default:''"`
	DownloadURL string    `gorm:"column:download_url;size:500;default:''"`
	IsMandatory bool      `gorm:"column:is_mandatory;default:false"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AppVersion) TableName() string {
	return "app_versions"
}
