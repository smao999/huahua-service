package auth

import (
	"context"
	"errors"
	"fmt"
	"huahua-service/internal/model"
	"huahua-service/internal/schema"
	"huahua-service/internal/security"
	"time"

	"gorm.io/gorm"
)

type AuthService struct {
	db     *gorm.DB
	jwtCfg security.JWTConfig
}

func NewAuthService(db *gorm.DB, jwtCfg security.JWTConfig) *AuthService {
	return &AuthService{db: db, jwtCfg: jwtCfg}
}

func (s *AuthService) Register(ctx context.Context, req schema.RegisterRequest) (*model.User, string, error) {
	var existing model.User
	if err := s.db.WithContext(ctx).Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, "", errors.New("用户名已被注册")
	}
	hashedPwd, _ := security.HashPassword(req.Password)

	user := &model.User{
		Username:       req.Username,
		Email:          req.Email,
		HashedPassword: hashedPwd,
		Nickname:       req.Username,
		UID:            fmt.Sprintf("%08d", time.Now().UnixNano()%100000000),
	}
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, "", err
	}
	token, _ := security.GenerateToken(s.jwtCfg, req.Username)
	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, req schema.LoginRequest) (*model.User, string, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}
	if !security.CheckPasswordHash(req.Password, user.HashedPassword) {
		return nil, "", errors.New("用户名或密码错误")
	}
	token, _ := security.GenerateToken(s.jwtCfg, user.Username)
	return &user, token, nil
}

func (s *AuthService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}
