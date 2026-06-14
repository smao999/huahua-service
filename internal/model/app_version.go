package model

import "time"

// AppVersion 应用版本管理 → app_versions 表
type AppVersion struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;comment:主键"`
	Version     string    `gorm:"uniqueIndex;size:20;comment:版本号"`
	Changelog   string    `gorm:"type:text;default:'';comment:更新日志"`
	DownloadURL string    `gorm:"column:download_url;size:500;default:'';comment:下载地址"`
	IsMandatory bool      `gorm:"column:is_mandatory;default:false;comment:是否强制更新"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
}

func (AppVersion) TableName() string {
	return "app_versions"
}
