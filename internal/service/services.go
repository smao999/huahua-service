package service

import (
	akshareSvc "huahua-service/internal/service/akshare"
	"huahua-service/internal/service/auth"
	fundSvc "huahua-service/internal/service/fund"
)

type Services struct {
	Auth    *auth.AuthService
	Akshare *akshareSvc.Client
	Fund    *fundSvc.Service
}
