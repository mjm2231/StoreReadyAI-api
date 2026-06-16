package dto

import "strings"

// RegisterRequest 后台管理员注册请求。
//
// 说明：
// 1. 当前用于后台管理员账号创建/初始化场景；
// 2. password / confirm_password 由 service 层继续做一致性校验；
// 3. is_super_admin 采用 0/1 表达，后续可继续收敛到更明确的角色体系。
type RegisterRequest struct {
	TenantID        uint64 `json:"tenant_id"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
	Nickname        string `json:"nickname"`
	Email           string `json:"email"`
	Mobile          string `json:"mobile"`
	Avatar          string `json:"avatar"`
	Remark          string `json:"remark"`
	IsSuperAdmin    uint8  `json:"is_super_admin"`
}

func (r RegisterRequest) Normalize() RegisterRequest {
	r.Username = strings.TrimSpace(r.Username)
	r.Password = strings.TrimSpace(r.Password)
	r.ConfirmPassword = strings.TrimSpace(r.ConfirmPassword)
	r.Nickname = strings.TrimSpace(r.Nickname)
	r.Email = strings.TrimSpace(r.Email)
	r.Mobile = strings.TrimSpace(r.Mobile)
	r.Avatar = strings.TrimSpace(r.Avatar)
	r.Remark = strings.TrimSpace(r.Remark)
	if r.IsSuperAdmin > 0 {
		r.IsSuperAdmin = 1
	}
	return r
}

// LoginRequest 后台管理员登录请求。
type LoginRequest struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (r LoginRequest) Normalize() LoginRequest {
	r.Username = strings.TrimSpace(r.Username)
	r.Password = strings.TrimSpace(r.Password)
	return r
}

// RefreshTokenRequest 刷新后台管理员访问令牌请求。
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (r RefreshTokenRequest) Normalize() RefreshTokenRequest {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	return r
}

// LogoutRequest 后台管理员退出登录请求。
//
// 说明：
// 1. 当前支持显式传 refresh_token 做单设备退出；
// 2. 若为空，可由上层结合当前登录态做兜底处理。
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r LogoutRequest) Normalize() LogoutRequest {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	return r
}

// ChangePasswordRequest 后台管理员修改密码请求。
type ChangePasswordRequest struct {
	TenantID        uint64 `json:"tenant_id"`
	OldPassword     string `json:"old_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

func (r ChangePasswordRequest) Normalize() ChangePasswordRequest {
	r.OldPassword = strings.TrimSpace(r.OldPassword)
	r.NewPassword = strings.TrimSpace(r.NewPassword)
	r.ConfirmPassword = strings.TrimSpace(r.ConfirmPassword)
	return r
}

// LoginResponse 后台管理员登录响应。
type LoginResponse struct {
	AdminUser AdminUserProfile `json:"admin_user"`
	Token     TokenPair        `json:"token"`
}

// RegisterResponse 后台管理员注册响应。
type RegisterResponse struct {
	AdminUser AdminUserProfile `json:"admin_user"`
}

// RefreshTokenResponse 刷新令牌响应。
type RefreshTokenResponse struct {
	Token TokenPair `json:"token"`
}

// TokenPair 后台管理员令牌对。
type TokenPair struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  uint64 `json:"access_token_expires_at"`
	RefreshTokenExpiresAt uint64 `json:"refresh_token_expires_at"`
	TokenType             string `json:"token_type"`
}

// AdminUserProfile 后台管理员资料。
type AdminUserProfile struct {
	TenantID     uint64   `json:"tenant_id"`
	ID           uint64   `json:"id"`
	Username     string   `json:"username"`
	Nickname     string   `json:"nickname"`
	Email        string   `json:"email"`
	Mobile       string   `json:"mobile"`
	Avatar       string   `json:"avatar"`
	Status       uint8    `json:"status"`
	IsSuperAdmin uint8    `json:"is_super_admin"`
	Roles        []string `json:"roles,omitempty"`
	LastLoginAt  uint64   `json:"last_login_at"`
	LastLoginIP  string   `json:"last_login_ip"`
	Remark       string   `json:"remark"`
	CreatedAt    uint64   `json:"created_at"`
	UpdatedAt    uint64   `json:"updated_at"`
}
