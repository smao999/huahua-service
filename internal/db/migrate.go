package db

import (
	"huahua-service/internal/model"
	"log"
)

// AutoMigrate 自动创建/更新所有数据库表
// 只创建不存在的表和列，不会删除已有结构
func AutoMigrate() {
	err := GORM.AutoMigrate(
		&model.User{},
		&model.UserData{},
		&model.FundNav{},
		&model.FundSector{},
		&model.AppVersion{},
		&model.FundBasicInfo{},
		&model.AgentToken{},
		&model.AgentRequest{},
		&model.FundCalibration{},
		&model.SystemNotice{},
		&model.SystemKV{},
	)
	if err != nil {
		panic("AutoMigrate 失败: " + err.Error())
	}
	log.Println("数据库表迁移完成")
}
