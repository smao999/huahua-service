package model

import "time"

// AgentRequest Agent 发起的交易请求 → agent_requests 表
type AgentRequest struct {
	ID         string    `gorm:"primaryKey;size:64;comment:请求ID(UUID)"`
	UserID     int64     `gorm:"column:user_id;index;comment:用户ID"`
	ActionType string    `gorm:"column:action_type;size:10;comment:操作类型(BUY/SELL)"`
	Payload    string    `gorm:"type:text;comment:请求载荷(JSON)"`
	Status     string    `gorm:"size:20;default:'PENDING';comment:状态(PENDING/PROCESSED/DISMISSED)"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
}

func (AgentRequest) TableName() string {
	return "agent_requests"
}
