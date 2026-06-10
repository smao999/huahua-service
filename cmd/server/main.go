package main

import (
	"huahua-service/internal/app"
	"huahua-service/internal/config"
	"huahua-service/internal/db"
)

func main() {
	cfg := config.Load()

	// 初始化数据库连接
	db.InitGorm(cfg.DB)
	db.InitRedis(cfg.Redis)
	defer db.Close()

	r := app.New(cfg)

	r.Run(":" + cfg.Port)
}
