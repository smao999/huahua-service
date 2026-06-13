package bootstrap

import (
	"time"

	"huahua-service/internal/config"
	"huahua-service/internal/db"
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"
	"huahua-service/internal/repository"
	"huahua-service/internal/security"
	"huahua-service/internal/service"
	"huahua-service/internal/service/auth"
)

// Dependencies 所有依赖的容器
type Dependencies struct {
	Handlers *handler.Handlers
	Midware  *middleware.Middleware
}

// Init 初始化所有依赖——这是整个项目唯一的"接线"位置
// 所有 Repository → Service → Handler 的创建顺序在这里一目了然
func Init(cfg *config.Config) *Dependencies {
	// 基础配置
	jwtCfg := security.JWTConfig{
		Secret: cfg.SecretKey,
		Expire: 7 * 24 * time.Hour,
	}
	// Repository
	repos := repository.NewRepos(db.GORM)
	// Service
	svcs := &service.Services{
		Auth: auth.NewAuthService(db.GORM, repos.User, jwtCfg),
		// Fund: fund.NewFundService(repos.FundBasicInfo, repos.FundNav),
	}
	// Middleware
	mw := &middleware.Middleware{
		AuthRequired: middleware.AuthRequired(jwtCfg, repos.User),
	}
	// Handler
	handlers := &handler.Handlers{
		Health:       handler.NewHealthHandler(),
		Fund:         handler.NewFundHandler(),
		Market:       handler.NewMarketHandler(),
		User:         handler.NewUserHandler(),
		Admin:        handler.NewAdminHandler(),
		Version:      handler.NewVersionHandler(),
		AgentRequest: handler.NewAgentRequestHandler(),
		Jcti:         handler.NewJctiHandler(),
		Public:       handler.NewPublicHandler(),
		Auth:         handler.NewAuthHandler(svcs.Auth),
		// Fund:      handler.NewFundHandler(svcs.Fund),
	}
	return &Dependencies{Handlers: handlers, Midware: mw}
}
