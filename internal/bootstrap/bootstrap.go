package bootstrap

import (
	"os"
	"time"

	"huahua-service/internal/config"
	"huahua-service/internal/db"
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"
	"huahua-service/internal/security"
	"huahua-service/internal/service"
	akshareSvc "huahua-service/internal/service/akshare"
	"huahua-service/internal/service/auth"
	fundSvc "huahua-service/internal/service/fund"
)

// Dependencies 所有依赖的容器
type Dependencies struct {
	Handlers *handler.Handlers
	Midware  *middleware.Middleware
}

// Init 初始化所有依赖——这是整个项目唯一的"接线"位置
func Init(cfg *config.Config) *Dependencies {
	// 基础配置
	jwtCfg := security.JWTConfig{
		Secret: cfg.SecretKey,
		Expire: time.Duration(cfg.AccessTokenExpireHours) * time.Hour,
	}
	// Service
	svcs := &service.Services{
		Auth:    auth.NewAuthService(db.GORM, jwtCfg),
		Akshare: initAkshare(),
	}
	svcs.Fund = fundSvc.NewService(svcs.Akshare)
	// Middleware
	mw := &middleware.Middleware{
		AuthRequired: middleware.AuthRequired(jwtCfg, svcs.Auth),
	}
	// Handler
	handlers := &handler.Handlers{
		Health:       handler.NewHealthHandler(),
		Fund:         handler.NewFundHandler(svcs.Fund),
		Market:       handler.NewMarketHandler(),
		User:         handler.NewUserHandler(),
		Admin:        handler.NewAdminHandler(),
		Version:      handler.NewVersionHandler(),
		AgentRequest: handler.NewAgentRequestHandler(),
		Jcti:         handler.NewJctiHandler(),
		Public:       handler.NewPublicHandler(),
		Auth:         handler.NewAuthHandler(svcs.Auth),
	}
	return &Dependencies{Handlers: handlers, Midware: mw}
}

// initAkshare 连接 Python akshare gRPC 服务。
// 读取 AKSHARE_GRPC_ADDR 环境变量，默认 localhost:9800。
func initAkshare() *akshareSvc.Client {
	addr := os.Getenv("AKSHARE_GRPC_ADDR")
	if addr == "" {
		addr = "localhost:9800"
	}
	client, err := akshareSvc.NewClient(addr)
	if err != nil {
		return nil
	}
	return client
}
