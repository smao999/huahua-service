package repository

import "gorm.io/gorm"

// Repos 聚合全部 Repository，启动时创建一次，各处按需取用
type Repos struct {
	User UserRepository
	// 后续新模块：
	// FundBasicInfo FundBasicInfoRepository
	// FundNav       FundNavRepository
	// AgentToken    AgentTokenRepository
}

func NewRepos(db *gorm.DB) *Repos {
	return &Repos{
		User: NewUserRepository(db),
		// FundBasicInfo: NewFundBasicInfoRepository(db),
		// FundNav:       NewFundNavRepository(db),
	}
}
