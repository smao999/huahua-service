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
	Auth         *AuthHandler
}
