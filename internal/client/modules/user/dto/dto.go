package dto

import (
	"storeready_ai/internal/contracts/entitlement"
	"storeready_ai/internal/contracts/user"
)

// DTO（Data Transfer Object）用于：
// 1) HTTP 入参/出参（JSON）
// 2) Service 层与 Handler 层之间的数据结构约定
// 注意：DTO 不要直接复用 GORM Model，避免把数据库字段/标签泄漏到接口层。

// ---- 通用响应封装（可按你项目统一的响应结构调整） ----
type Resp[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data,omitempty"`
}

// ---- Token ----

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
	RefreshToken string `json:"refresh_token,omitempty"`
	RefreshExpIn int64  `json:"refresh_expires_in,omitempty"` // seconds
}

// ---- Firebase 登录 ----

// FirebaseLoginReq Firebase 登录请求。
// 客户端通过 Firebase SDK 登录后，将 id_token 传给后端。
type FirebaseLoginReq struct {
	IDToken    string  `json:"id_token" binding:"required"`
	DeviceID   *string `json:"device_id,omitempty"`   // 可选：用于多端 refresh token 管理
	DeviceName *string `json:"device_name,omitempty"` // 可选
}

// FirebaseLoginResp Firebase 登录响应。
type FirebaseLoginResp struct {
	ServerNow   uint64                  `json:"server_now"`
	Token       TokenPair               `json:"token"`
	User        user.UserVO             `json:"user"`
	Entitlement entitlement.Entitlement `json:"entitlement"`
}

// AccountRegisterReq 账号密码注册请求。
type AccountRegisterReq struct {
	Email      string  `json:"email" binding:"required,email"`
	Password   string  `json:"password" binding:"required,min=8"`
	Name       *string `json:"name,omitempty"`
	DeviceID   *string `json:"device_id,omitempty"`   // 可选：用于多端 refresh token 管理
	DeviceName *string `json:"device_name,omitempty"` // 可选
}

// AccountRegisterResp 账号密码注册响应。
type AccountRegisterResp struct {
	ServerNow   uint64                  `json:"server_now"`
	Token       TokenPair               `json:"token"`
	User        user.UserVO             `json:"user"`
	Entitlement entitlement.Entitlement `json:"entitlement"`
}

// AccountLoginReq 账号密码登录请求。
type AccountLoginReq struct {
	Email      string  `json:"email" binding:"required,email"`
	Password   string  `json:"password" binding:"required"`
	DeviceID   *string `json:"device_id,omitempty"`   // 可选：用于多端 refresh token 管理
	DeviceName *string `json:"device_name,omitempty"` // 可选
}

// AccountLoginResp 账号密码登录响应。
type AccountLoginResp struct {
	ServerNow   uint64                  `json:"server_now"`
	Token       TokenPair               `json:"token"`
	User        user.UserVO             `json:"user"`
	Entitlement entitlement.Entitlement `json:"entitlement"`
}

// ---- Refresh Token 续期 ----

type RefreshTokenReq struct {
	RefreshToken string  `json:"refresh_token" binding:"required"`
	DeviceID     *string `json:"device_id,omitempty"`
}

type RefreshTokenResp struct {
	ServerNow   uint64                  `json:"server_now"`
	Token       TokenPair               `json:"token"`
	User        user.UserVO             `json:"user"`
	Entitlement entitlement.Entitlement `json:"entitlement"`
}

// ---- Logout（吊销 refresh token） ----

type LogoutReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
