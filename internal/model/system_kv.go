package model

import "time"

// SystemKV 系统键值对 → system_kv 表
type SystemKV struct {
	Key       string    `gorm:"primaryKey;size:120;comment:键名"`
	ValueJSON string    `gorm:"column:value_json;type:text;comment:值(JSON)"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;comment:更新时间"`
}

func (SystemKV) TableName() string {
	return "system_kv"
}
