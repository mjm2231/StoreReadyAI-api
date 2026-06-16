package dto

import (
	"strings"

	rolemodel "storeready_ai/internal/admin/modules/roles/model"
)

// CreateRoleRequest 创建后台角色请求。
type CreateRoleRequest struct {
	TenantID uint64 `json:"tenant_id"`
	Name     string `json:"name" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Status   uint8  `json:"status"`
	Sort     int32  `json:"sort"`
	IsSystem uint8  `json:"is_system"`
	Remark   string `json:"remark"`
}

func (r CreateRoleRequest) Normalize() CreateRoleRequest {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	r.Remark = strings.TrimSpace(r.Remark)
	if r.Status == 0 {
		r.Status = rolemodel.AdminRoleStatusActive
	}
	if r.IsSystem > 0 {
		r.IsSystem = 1
	}
	return r
}

// UpdateRoleRequest 更新后台角色请求。
type UpdateRoleRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Status   uint8  `json:"status" binding:"required"`
	Sort     int32  `json:"sort"`
	IsSystem uint8  `json:"is_system"`
	Remark   string `json:"remark"`
}

func (r UpdateRoleRequest) Normalize() UpdateRoleRequest {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	r.Remark = strings.TrimSpace(r.Remark)
	if r.IsSystem > 0 {
		r.IsSystem = 1
	}
	return r
}

// UpdateRoleStatusRequest 更新后台角色状态请求。
type UpdateRoleStatusRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
	Status   uint8  `json:"status" binding:"required"`
}

// DeleteRoleRequest 删除后台角色请求。
type DeleteRoleRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
}

// GetRoleDetailRequest 获取后台角色详情请求。
type GetRoleDetailRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
}

// RoleListRequest 后台角色列表请求。
type RoleListRequest struct {
	TenantID uint64   `json:"tenant_id"`
	Keyword  string   `json:"keyword"`
	Status   *uint8   `json:"status"`
	IsSystem *uint8   `json:"is_system"`
	IDs      []uint64 `json:"ids"`
	Offset   int      `json:"offset"`
	Limit    int      `json:"limit"`
}

func (r RoleListRequest) Normalize() RoleListRequest {
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 100 {
		r.Limit = 100
	}
	if len(r.IDs) > 0 {
		ids := make([]uint64, 0, len(r.IDs))
		seen := make(map[uint64]struct{}, len(r.IDs))
		for _, id := range r.IDs {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			r.IDs = nil
		} else {
			r.IDs = ids
		}
	}
	if r.IsSystem != nil && *r.IsSystem > 0 {
		v := uint8(1)
		r.IsSystem = &v
	}
	return r
}

// RoleItem 后台角色列表项。
type RoleItem struct {
	TenantID  uint64 `json:"tenant_id"`
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Status    uint8  `json:"status"`
	Sort      int32  `json:"sort"`
	IsSystem  uint8  `json:"is_system"`
	Remark    string `json:"remark"`
	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
	DeletedAt uint64 `json:"deleted_at"`
}

// RoleDetail 后台角色详情。
type RoleDetail struct {
	TenantID  uint64 `json:"tenant_id"`
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Status    uint8  `json:"status"`
	Sort      int32  `json:"sort"`
	IsSystem  uint8  `json:"is_system"`
	Remark    string `json:"remark"`
	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
	DeletedAt uint64 `json:"deleted_at"`
}

// RoleListResponse 后台角色列表响应。
type RoleListResponse struct {
	Total int64      `json:"total"`
	Items []RoleItem `json:"items"`
}

func ToRoleItem(m *rolemodel.AdminRole) RoleItem {
	if m == nil {
		return RoleItem{}
	}
	return RoleItem{
		TenantID:  m.TenantID,
		ID:        m.ID,
		Name:      m.Name,
		Code:      m.Code,
		Status:    m.Status,
		Sort:      m.Sort,
		IsSystem:  m.IsSystem,
		Remark:    m.Remark,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func ToRoleDetail(m *rolemodel.AdminRole) RoleDetail {
	if m == nil {
		return RoleDetail{}
	}
	return RoleDetail{
		TenantID:  m.TenantID,
		ID:        m.ID,
		Name:      m.Name,
		Code:      m.Code,
		Status:    m.Status,
		Sort:      m.Sort,
		IsSystem:  m.IsSystem,
		Remark:    m.Remark,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func ToRoleItems(list []*rolemodel.AdminRole) []RoleItem {
	if len(list) == 0 {
		return nil
	}
	items := make([]RoleItem, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		items = append(items, ToRoleItem(item))
	}
	if len(items) == 0 {
		return nil
	}
	return items
}
