package model

import "time"

// SystemKV 系统键值对 → system_kv 表
type SystemKV struct {
	Key       string    `gorm:"primaryKey;size:120"`
	ValueJSON string    `gorm:"column:value_json;type:text"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SystemKV) TableName() string {
	return "system_kv"
}
