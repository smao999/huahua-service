package handler

// Handlers 聚合所有 Handler，供 router 层统一注入
type Handlers struct {
	Health       *HealthHandler
	Fund         *FundHandler
	Market       *MarketHandler
	User         *UserHandler
	Admin        *AdminHandler
	Version      *VersionHandler
	AgentRequest *AgentRequestHandler
	Jcti         *JctiHandler
	Public       *PublicHandler
}

// NewHandlers 创建所有 Handler 实例
func NewHandlers() *Handlers {
	return &Handlers{
		Health:       NewHealthHandler(),
		Fund:         NewFundHandler(),
		Market:       NewMarketHandler(),
		User:         NewUserHandler(),
		Admin:        NewAdminHandler(),
		Version:      NewVersionHandler(),
		AgentRequest: NewAgentRequestHandler(),
		Jcti:         NewJctiHandler(),
		Public:       NewPublicHandler(),
	}
}
