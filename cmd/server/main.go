package main

import (
	"huahua-service/internal/config"
	"huahua-service/internal/db"
	"huahua-service/internal/handler"
	"huahua-service/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// 初始化数据库连接
	//db.InitGorm(cfg.DB)
	//db.InitRedis(cfg.Redis)
	defer db.Close()

	r := gin.Default()
	h := handler.NewHealthHandler()
	router.Register(r, h)

	r.Run(":" + cfg.Port)
}
