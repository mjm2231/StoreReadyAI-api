package dto

import permissionmodel "storeready_ai/internal/admin/modules/rolepermissions/model"

// RolePermissionItem 是角色权限关联项。
//
// 说明：
// 1. 用于列表返回与批量写入结果展示；
// 2. 当前先聚焦 role_id / permission_id 维度；
// 3. 若后续需要返回 permission code / name，可在 service 层聚合后扩展字段。
type RolePermissionItem struct {
	ID           uint64 `json:"id"`
	TenantID     uint64 `json:"tenant_id"`
	RoleID       uint64 `json:"role_id"`
	PermissionID uint64 `json:"permission_id"`
	CreatedAt    uint64 `json:"created_at"`
	UpdatedAt    uint64 `json:"updated_at"`
}

// ListRolePermissionsRequest 查询单角色权限关联请求。
type ListRolePermissionsRequest struct {
	TenantID uint64 `json:"tenant_id"`
	RoleID   uint64 `json:"role_id"`
}

// ListRolePermissionsResponse 查询单角色权限关联响应。
type ListRolePermissionsResponse struct {
	Items []RolePermissionItem `json:"items"`
}

// ListRolePermissionIDsRequest 查询单角色 permission_id 列表请求。
type ListRolePermissionIDsRequest struct {
	TenantID uint64 `json:"tenant_id"`
	RoleID   uint64 `json:"role_id"`
}

// ListRolePermissionIDsResponse 查询单角色 permission_id 列表响应。
type ListRolePermissionIDsResponse struct {
	RoleID          uint64   `json:"role_id"`
	PermissionIDs   []uint64 `json:"permission_ids"`
	PermissionCount int      `json:"permission_count"`
}

// GrantRolePermissionsRequest 给角色追加权限请求。
type GrantRolePermissionsRequest struct {
	TenantID      uint64   `json:"tenant_id"`
	RoleID        uint64   `json:"role_id"`
	PermissionIDs []uint64 `json:"permission_ids"`
}

// RevokeRolePermissionsRequest 删除角色下指定权限请求。
type RevokeRolePermissionsRequest struct {
	TenantID      uint64   `json:"tenant_id"`
	RoleID        uint64   `json:"role_id"`
	PermissionIDs []uint64 `json:"permission_ids"`
}

// ReplaceRolePermissionsRequest 全量替换角色权限请求。
type ReplaceRolePermissionsRequest struct {
	TenantID      uint64   `json:"tenant_id"`
	RoleID        uint64   `json:"role_id"`
	PermissionIDs []uint64 `json:"permission_ids"`
}

// ClearRolePermissionsRequest 清空角色全部权限请求。
type ClearRolePermissionsRequest struct {
	TenantID uint64 `json:"tenant_id"`
	RoleID   uint64 `json:"role_id"`
}

// RolePermissionMutationResponse 角色权限写操作统一响应。
type RolePermissionMutationResponse struct {
	RoleID          uint64   `json:"role_id"`
	PermissionIDs   []uint64 `json:"permission_ids"`
	PermissionCount int      `json:"permission_count"`
}

// ListRolePermissionsByRoleIDsRequest 查询多角色权限关联请求。
type ListRolePermissionsByRoleIDsRequest struct {
	TenantID uint64   `json:"tenant_id"`
	RoleIDs  []uint64 `json:"role_ids"`
}

// ListRolePermissionsByRoleIDsResponse 查询多角色权限关联响应。
type ListRolePermissionsByRoleIDsResponse struct {
	Items []RolePermissionItem `json:"items"`
}

// ListPermissionIDsByRoleIDsRequest 查询多角色去重 permission_id 列表请求。
type ListPermissionIDsByRoleIDsRequest struct {
	TenantID uint64   `json:"tenant_id"`
	RoleIDs  []uint64 `json:"role_ids"`
}

// ListPermissionIDsByRoleIDsResponse 查询多角色去重 permission_id 列表响应。
type ListPermissionIDsByRoleIDsResponse struct {
	RoleIDs         []uint64 `json:"role_ids"`
	PermissionIDs   []uint64 `json:"permission_ids"`
	PermissionCount int      `json:"permission_count"`
}

// FromModel 把 model.AdminRolePermission 转成 dto.RolePermissionItem。
func FromModel(in permissionmodel.AdminRolePermission) RolePermissionItem {
	return RolePermissionItem{
		ID:           in.ID,
		TenantID:     in.TenantID,
		RoleID:       in.RoleID,
		PermissionID: in.PermissionID,
		CreatedAt:    in.CreatedAt,
		UpdatedAt:    in.UpdatedAt,
	}
}

// FromModels 批量转换角色权限关联项。
func FromModels(in []permissionmodel.AdminRolePermission) []RolePermissionItem {
	if len(in) == 0 {
		return []RolePermissionItem{}
	}
	out := make([]RolePermissionItem, 0, len(in))
	for _, item := range in {
		out = append(out, FromModel(item))
	}
	return out
}
