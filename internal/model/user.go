package model

import "time"

type User struct {
	ID             int64      `gorm:"primaryKey;autoIncrement"`
	Username       string     `gorm:"uniqueIndex;size:50"`
	Email          string     `gorm:"uniqueIndex;size:100"`
	HashedPassword string     `gorm:"size:255;column:hashed_password"`
	Avatar         string     `gorm:"size:255;default:''"`
	Nickname       string     `gorm:"size:50;default:''"`
	UID            string     `gorm:"uniqueIndex;size:32"`
	InvitedBy      *int64     `gorm:"column:invited_by"`
	VIPExpiresAt   time.Time  `gorm:"column:vip_expires_at"`
	ProExpiresAt   *time.Time `gorm:"column:pro_expires_at"`
	IsAdmin        bool       `gorm:"column:is_admin;default:false"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }
