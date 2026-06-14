package model

import "time"

// AgentToken AI Agent 令牌 → agent_tokens 表
type AgentToken struct {
	ID         int64      `gorm:"primaryKey;autoIncrement;comment:主键"`
	UserID     int64      `gorm:"column:user_id;index;comment:用户ID"`
	TokenHash  string     `gorm:"column:token_hash;uniqueIndex;size:64;comment:令牌哈希(SHA256)"`
	Name       string     `gorm:"size:50;default:'Agent Token';comment:令牌名称"`
	Scope      string     `gorm:"size:100;default:'agent:full';comment:权限范围"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
	LastUsedAt *time.Time `gorm:"column:last_used_at;comment:最后使用时间"`
	LastUsedIP string     `gorm:"column:last_used_ip;size:50;comment:最后使用IP"`
	ExpiresAt  *time.Time `gorm:"column:expires_at;comment:过期时间"`
}

func (AgentToken) TableName() string {
	return "agent_tokens"
}
