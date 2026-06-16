package dto

import (
	"strings"

	permissionmodel "storeready_ai/internal/admin/modules/permissions/model"
)

// CreatePermissionRequest 创建后台权限请求。
type CreatePermissionRequest struct {
	TenantID uint64 `json:"tenant_id"`
	Name     string `json:"name" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Module   string `json:"module"`
	Type     uint8  `json:"type" binding:"required"`
	ParentID uint64 `json:"parent_id"`
	Path     string `json:"path"`
	Icon     string `json:"icon"`
	Sort     int32  `json:"sort"`
	Status   uint8  `json:"status"`
	IsSystem uint8  `json:"is_system"`
	Remark   string `json:"remark"`
}

func (r CreatePermissionRequest) Normalize() CreatePermissionRequest {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	r.Module = strings.TrimSpace(r.Module)
	r.Path = strings.TrimSpace(r.Path)
	r.Icon = strings.TrimSpace(r.Icon)
	r.Remark = strings.TrimSpace(r.Remark)
	if r.Status == 0 {
		r.Status = permissionmodel.AdminPermissionStatusActive
	}
	if r.IsSystem > 0 {
		r.IsSystem = 1
	}
	return r
}

// UpdatePermissionRequest 更新后台权限请求。
type UpdatePermissionRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Module   string `json:"module"`
	Type     uint8  `json:"type" binding:"required"`
	ParentID uint64 `json:"parent_id"`
	Path     string `json:"path"`
	Icon     string `json:"icon"`
	Sort     int32  `json:"sort"`
	Status   uint8  `json:"status" binding:"required"`
	IsSystem uint8  `json:"is_system"`
	Remark   string `json:"remark"`
}

func (r UpdatePermissionRequest) Normalize() UpdatePermissionRequest {
	r.Name = strings.TrimSpace(r.Name)
	r.Code = strings.TrimSpace(r.Code)
	r.Module = strings.TrimSpace(r.Module)
	r.Path = strings.TrimSpace(r.Path)
	r.Icon = strings.TrimSpace(r.Icon)
	r.Remark = strings.TrimSpace(r.Remark)
	if r.IsSystem > 0 {
		r.IsSystem = 1
	}
	return r
}

// UpdatePermissionStatusRequest 更新后台权限状态请求。
type UpdatePermissionStatusRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
	Status   uint8  `json:"status" binding:"required"`
}

// DeletePermissionRequest 删除后台权限请求。
type DeletePermissionRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
}

// GetPermissionDetailRequest 获取后台权限详情请求。
type GetPermissionDetailRequest struct {
	TenantID uint64 `json:"tenant_id"`
	ID       uint64 `json:"id" binding:"required"`
}

// PermissionListRequest 后台权限列表请求。
type PermissionListRequest struct {
	TenantID uint64   `json:"tenant_id"`
	Keyword  string   `json:"keyword"`
	Module   string   `json:"module"`
	Type     *uint8   `json:"type"`
	Status   *uint8   `json:"status"`
	ParentID *uint64  `json:"parent_id"`
	IsSystem *uint8   `json:"is_system"`
	IDs      []uint64 `json:"ids"`
	Offset   int      `json:"offset"`
	Limit    int      `json:"limit"`
}

func (r PermissionListRequest) Normalize() PermissionListRequest {
	r.Keyword = strings.TrimSpace(r.Keyword)
	r.Module = strings.TrimSpace(r.Module)
	if r.Offset < 0 {
		r.Offset = 0
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 200 {
		r.Limit = 200
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

// PermissionItem 后台权限列表项。
type PermissionItem struct {
	TenantID  uint64 `json:"tenant_id"`
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Module    string `json:"module"`
	Type      uint8  `json:"type"`
	ParentID  uint64 `json:"parent_id"`
	Path      string `json:"path"`
	Icon      string `json:"icon"`
	Sort      int32  `json:"sort"`
	Status    uint8  `json:"status"`
	IsSystem  uint8  `json:"is_system"`
	Remark    string `json:"remark"`
	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
	DeletedAt uint64 `json:"deleted_at"`
}

// PermissionDetail 后台权限详情。
type PermissionDetail struct {
	TenantID  uint64 `json:"tenant_id"`
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Module    string `json:"module"`
	Type      uint8  `json:"type"`
	ParentID  uint64 `json:"parent_id"`
	Path      string `json:"path"`
	Icon      string `json:"icon"`
	Sort      int32  `json:"sort"`
	Status    uint8  `json:"status"`
	IsSystem  uint8  `json:"is_system"`
	Remark    string `json:"remark"`
	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
	DeletedAt uint64 `json:"deleted_at"`
}

// PermissionListResponse 后台权限列表响应。
type PermissionListResponse struct {
	Total int64            `json:"total"`
	Items []PermissionItem `json:"items"`
}

func ToPermissionItem(m *permissionmodel.AdminPermission) PermissionItem {
	if m == nil {
		return PermissionItem{}
	}
	return PermissionItem{
		TenantID:  m.TenantID,
		ID:        m.ID,
		Name:      m.Name,
		Code:      m.Code,
		Module:    m.Module,
		Type:      m.Type,
		ParentID:  m.ParentID,
		Path:      m.Path,
		Icon:      m.Icon,
		Sort:      m.Sort,
		Status:    m.Status,
		IsSystem:  m.IsSystem,
		Remark:    m.Remark,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func ToPermissionDetail(m *permissionmodel.AdminPermission) PermissionDetail {
	if m == nil {
		return PermissionDetail{}
	}
	return PermissionDetail{
		TenantID:  m.TenantID,
		ID:        m.ID,
		Name:      m.Name,
		Code:      m.Code,
		Module:    m.Module,
		Type:      m.Type,
		ParentID:  m.ParentID,
		Path:      m.Path,
		Icon:      m.Icon,
		Sort:      m.Sort,
		Status:    m.Status,
		IsSystem:  m.IsSystem,
		Remark:    m.Remark,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func ToPermissionItems(list []*permissionmodel.AdminPermission) []PermissionItem {
	if len(list) == 0 {
		return nil
	}
	items := make([]PermissionItem, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		items = append(items, ToPermissionItem(item))
	}
	if len(items) == 0 {
		return nil
	}
	return items
}
