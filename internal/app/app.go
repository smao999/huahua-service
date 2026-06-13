package app

import (
	"huahua-service/internal/bootstrap"
	"huahua-service/internal/config"
	"huahua-service/internal/middleware"
	"huahua-service/internal/router"

	"github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
	r := gin.New()

	r.Use(middleware.CORS())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())

	deps := bootstrap.Init(cfg)
	router.Register(r, deps.Handlers, deps.Midware)

	return r
}
