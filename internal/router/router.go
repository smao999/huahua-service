package router

import (
	"huahua-service/internal/handler"
	"huahua-service/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, h *handler.Handlers, mw *middleware.Middleware) {
	api := r.Group("/api")
	registerHealth(api, h.Health)
	registerFund(api, h.Fund)
	registerMarket(api, h.Market)
	registerUser(api, h.User, mw)
	registerAdmin(api, h.Admin, mw)
	registerVersion(api, h.Version)
	registerAgentRequest(api, h.AgentRequest, mw)
	registerJcti(api, h.Jcti, mw)
	registerPublic(api, h.Public)
	registerAuth(api, h.Auth)
}
