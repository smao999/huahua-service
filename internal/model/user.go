package model

import "time"

type User struct {
	ID             int64      `gorm:"primaryKey;autoIncrement;comment:主键"`
	Username       string     `gorm:"uniqueIndex;size:50;comment:用户名"`
	Email          string     `gorm:"uniqueIndex;size:100;comment:邮箱"`
	HashedPassword string     `gorm:"size:255;column:hashed_password;comment:密码哈希(bcrypt)"`
	Avatar         string     `gorm:"size:255;default:'';comment:头像URL"`
	Nickname       string     `gorm:"size:50;default:'';comment:昵称"`
	UID            string     `gorm:"uniqueIndex;size:32;comment:用户UID(8位数字)"`
	InvitedBy      *int64     `gorm:"column:invited_by;comment:邀请人用户ID"`
	VIPExpiresAt   time.Time  `gorm:"column:vip_expires_at;comment:VIP到期时间"`
	ProExpiresAt   *time.Time `gorm:"column:pro_expires_at;comment:高级会员到期时间"`
	IsAdmin        bool       `gorm:"column:is_admin;default:false;comment:是否管理员"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;comment:创建时间"`
}

func (User) TableName() string { return "users" }
