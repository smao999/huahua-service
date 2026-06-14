package schema

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=4,max=20" cn:"用户名"`
	Password string `json:"password" binding:"required,min=8,max=20" cn:"密码"`
	Email    string `json:"email" binding:"required,email" cn:"邮箱"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required,min=4,max=20" cn:"用户名"`
	Password string `json:"password" binding:"required,min=8,max=20" cn:"密码"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	UId         string `json:"uid"`
}
