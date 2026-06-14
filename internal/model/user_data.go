package model

// UserData 用户基金持仓备份 → user_data 表
type UserData struct {
	ID        int64  `gorm:"primaryKey;autoIncrement;comment:主键"`
	UserID    int64  `gorm:"column:user_id;uniqueIndex;comment:用户ID"`
	DataJSON  string `gorm:"column:data_json;type:text;comment:基金持仓JSON"`
	DataEtag  string `gorm:"column:data_etag;size:32;comment:数据变更校验码"`
	UpdatedAt string `gorm:"column:updated_at;size:30;comment:更新时间"`
}

func (UserData) TableName() string {
	return "user_data"
}
