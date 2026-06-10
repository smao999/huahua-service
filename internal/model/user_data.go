package model

// UserData 用户基金持仓备份 → user_data 表
type UserData struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	UserID    int64  `gorm:"column:user_id;uniqueIndex"`
	DataJSON  string `gorm:"column:data_json;type:text"` // 基金持仓 JSON
	DataEtag  string `gorm:"column:data_etag;size:32"`   // 数据变更校验码
	UpdatedAt string `gorm:"column:updated_at;size:30"`  // 更新时间字符串
}

func (UserData) TableName() string {
	return "user_data"
}
