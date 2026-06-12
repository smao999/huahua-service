package service

import "huahua-service/internal/service/auth"

type Services struct {
	Auth *auth.AuthService
}

func NewServices(authService *auth.AuthService) *Services {
	return &Services{
		Auth: authService,
	}
}
