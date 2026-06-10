package app

import (
	"huahua-service/internal/config"
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"
	"huahua-service/internal/router"

	"github.com/gin-gonic/gin"
)

func New(cfg *config.Config) *gin.Engine {
	r := gin.New()

	r.Use(middleware.CORS())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())

	h := handler.NewHandlers()
	router.Register(r, h)

	return r
}
