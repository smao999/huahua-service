package model

import "time"

// AgentRequest Agent 发起的交易请求 → agent_requests 表
type AgentRequest struct {
	ID         string    `gorm:"primaryKey;size:64"`
	UserID     int64     `gorm:"column:user_id;index"`
	ActionType string    `gorm:"column:action_type;size:10"` // "BUY" / "SELL"
	Payload    string    `gorm:"type:text"`                  // JSON
	Status     string    `gorm:"size:20;default:'PENDING'"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (AgentRequest) TableName() string {
	return "agent_requests"
}
