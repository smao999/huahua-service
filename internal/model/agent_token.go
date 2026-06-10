package model

import "time"

// AgentToken AI Agent 令牌 → agent_tokens 表
type AgentToken struct {
	ID         int64      `gorm:"primaryKey;autoIncrement"`
	UserID     int64      `gorm:"column:user_id;index"`
	TokenHash  string     `gorm:"column:token_hash;uniqueIndex;size:64"`
	Name       string     `gorm:"size:50;default:'Agent Token'"`
	Scope      string     `gorm:"size:100;default:'agent:full'"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
	LastUsedAt *time.Time `gorm:"column:last_used_at"`
	LastUsedIP string     `gorm:"column:last_used_ip;size:50"`
	ExpiresAt  *time.Time `gorm:"column:expires_at"`
}

func (AgentToken) TableName() string {
	return "agent_tokens"
}
