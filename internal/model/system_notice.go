package model

import "time"

// SystemNotice 系统通知 → system_notices 表
type SystemNotice struct {
	ID        string    `gorm:"primaryKey;size:64"`
	Title     string    `gorm:"size:200"`
	Content   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (SystemNotice) TableName() string {
	return "system_notices"
}
