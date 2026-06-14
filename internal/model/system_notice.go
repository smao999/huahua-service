package model

import "time"

// SystemNotice 系统通知 → system_notices 表
type SystemNotice struct {
	ID        string    `gorm:"primaryKey;size:64;comment:通知ID(UUID)"`
	Title     string    `gorm:"size:200;comment:通知标题"`
	Content   string    `gorm:"type:text;comment:通知内容"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
}

func (SystemNotice) TableName() string {
	return "system_notices"
}
