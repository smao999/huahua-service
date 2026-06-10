package router

import (
	"huahua-service/internal/handler"

	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, h *handler.Handlers) {
	api := r.Group("/api")
	registerHealth(api, h.Health)
	registerFund(api, h.Fund)
	registerMarket(api, h.Market)
	registerUser(api, h.User)
	registerAdmin(api, h.Admin)
	registerVersion(api, h.Version)
	registerAgentRequest(api, h.AgentRequest)
	registerJcti(api, h.Jcti)
	registerPublic(api, h.Public)
}
