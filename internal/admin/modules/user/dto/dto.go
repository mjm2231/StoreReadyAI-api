package dto

import (
	"strings"

	usermodel "storeready_ai/internal/admin/modules/user/model"
)

// CreateAdminUserRequest 创建后台管理员请求。
type CreateAdminUserRequest struct {
	TenantID     uint64 `json:"tenant_id"`
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	Nickname     string `json:"nickname"`
	Email        string `json:"email"`
	Mobile       string `json:"mobile"`
	Avatar       string `json:"avatar"`
	Status       uint8  `json:"status"`
	IsSuperAdmin uint8  `json:"is_super_admin"`
	Remark       string `json:"remark"`
}

func (r CreateAdminUserRequest) Normalize() CreateAdminUserRequest {
	r.Username = strings.TrimSpace(r.Username)
	r.Password = strings.TrimSpace(r.Password)
	r.Nickname = strings.TrimSpace(r.Nickname)
	r.Email = strings.TrimSpace(r.Email)
	r.Mobile = strings.TrimSpace(r.Mobile)
	r.Avatar = strings.TrimSpace(r.Avatar)
	r.Remark = strings.TrimSpace(r.Remark)
	if r.Status == 0 {
		r.Status = usermodel.AdminUserStatusActive
	}
	return r
}

// UpdateAdminUserRequest 更新后台管理员请求。
type UpdateAdminUserRequest struct {
	TenantID     uint64 `json:"tenant_id"`
	ID           uint64 `json:"id" binding:"required"`
	Nickname     string `json:"nickname"`
	Email        string `json:"email"`
	Mobile       string `json:"mobile"`
	Avatar       string `json:"avatar"`
	Status       uint8  `json:"status"`
	IsSuperAdmin uint8  `json:"is_super_admin"`
	Remark       string `json:"remark"`
}

func (r UpdateAdminUserRequest) Normalize() UpdateAdminUserRequest {
	r.Nickname = strings.TrimSpace(r.Nickname)
	r.Email = strings.TrimSpace(r.Email)
	r.Mobile = strings.TrimSpace(r.Mobile)
	r.Avatar = strings.TrimSpace(r.Avatar)
	r.Remark = strings.TrimSpace(r.Remark)
	return r
}

// UpdateAdminUserPasswordRequest 更新后台管理员密码请求。
type UpdateAdminUserPasswordRequest struct {
	TenantID        uint64 `json:"tenant_id"`
	ID              uint64 `json:"id" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

func (r UpdateAdminUserPasswordRequest) Normalize() UpdateAdminUserPasswordRequest {
	r.Password = strings.TrimSpace(r.Password)
	r.ConfirmPassword = strings.TrimSpace(r.ConfirmPassword)
	return r
}

// UpdateAdminUserStatusRequest 更新后台管理员状态请求。
type UpdateAdminUserStatusRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
	Status   uint8  `json:"status" binding:"required"`
}

// DeleteAdminUserRequest 删除后台管理员请求。
type DeleteAdminUserRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
}

// GetAdminUserDetailRequest 获取后台管理员详情请求。
type GetAdminUserDetailRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
}

type PageReq struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// AdminUserListRequest 后台管理员列表请求。
type AdminUserListRequest struct {
	TenantID     string  `json:"tenant_id"`
	Keyword      string  `json:"keyword"`
	Status       uint8   `json:"status"`
	IsSuperAdmin *uint8  `json:"is_super_admin"`
	Page         PageReq `json:"page_req"`
}

func (r AdminUserListRequest) Normalize() AdminUserListRequest {
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Page.Page < 1 {
		r.Page.Page = 1
	}
	if r.Page.PageSize <= 0 {
		r.Page.PageSize = 20
	}
	if r.Page.PageSize > 100 {
		r.Page.PageSize = 100
	}
	return r
}

// AdminUserItem 后台管理员列表项。
type AdminUserItem struct {
	TenantID     uint64   `json:"tenant_id"`
	ID           uint64   `json:"id"`
	Username     string   `json:"username"`
	Nickname     string   `json:"nickname"`
	Email        string   `json:"email"`
	Mobile       string   `json:"mobile"`
	Avatar       string   `json:"avatar"`
	Status       uint8    `json:"status"`
	IsSuperAdmin uint8    `json:"is_super_admin"`
	LastLoginAt  uint64   `json:"last_login_at"`
	LastLoginIP  string   `json:"last_login_ip"`
	Remark       string   `json:"remark"`
	CreatedAt    uint64   `json:"created_at"`
	UpdatedAt    uint64   `json:"updated_at"`
	Roles        []string // 角色
	Perms        []string // 权限点（超管可能为 ["*"]）
}

// AdminUserDetail 后台管理员详情。
type AdminUserDetail struct {
	TenantID     uint64 `json:"tenant_id"`
	ID           uint64 `json:"id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname"`
	Email        string `json:"email"`
	Mobile       string `json:"mobile"`
	Avatar       string `json:"avatar"`
	Status       uint8  `json:"status"`
	IsSuperAdmin uint8  `json:"is_super_admin"`
	LastLoginAt  uint64 `json:"last_login_at"`
	LastLoginIP  string `json:"last_login_ip"`
	Remark       string `json:"remark"`
	CreatedAt    uint64 `json:"created_at"`
	UpdatedAt    uint64 `json:"updated_at"`
	DeletedAt    uint64 `json:"deleted_at"`
}

// AdminUserListResponse 后台管理员列表响应。
type AdminUserListResponse struct {
	Total int64           `json:"total"`
	Items []AdminUserItem `json:"items"`
}

func ToAdminUserItem(m *usermodel.AdminUser) AdminUserItem {
	if m == nil {
		return AdminUserItem{}
	}
	return AdminUserItem{
		TenantID:     m.TenantID,
		ID:           m.ID,
		Username:     m.Username,
		Nickname:     m.Nickname,
		Email:        m.Email,
		Mobile:       m.Mobile,
		Avatar:       m.Avatar,
		Status:       m.Status,
		IsSuperAdmin: m.IsSuperAdmin,
		LastLoginAt:  m.LastLoginAt,
		LastLoginIP:  m.LastLoginIP,
		Remark:       m.Remark,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func ToAdminUserDetail(m *usermodel.AdminUser) AdminUserDetail {
	if m == nil {
		return AdminUserDetail{}
	}
	return AdminUserDetail{
		TenantID:     m.TenantID,
		ID:           m.ID,
		Username:     m.Username,
		Nickname:     m.Nickname,
		Email:        m.Email,
		Mobile:       m.Mobile,
		Avatar:       m.Avatar,
		Status:       m.Status,
		IsSuperAdmin: m.IsSuperAdmin,
		LastLoginAt:  m.LastLoginAt,
		LastLoginIP:  m.LastLoginIP,
		Remark:       m.Remark,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}
}

func ToAdminUserItems(list []*usermodel.AdminUser) []AdminUserItem {
	if len(list) == 0 {
		return nil
	}
	items := make([]AdminUserItem, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		items = append(items, ToAdminUserItem(item))
	}
	if len(items) == 0 {
		return nil
	}
	return items
}
