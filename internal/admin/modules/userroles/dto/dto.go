package dto

import userrolemodel "storeready_ai/internal/admin/modules/userroles/model"

// AdminUserRoleItem 是后台管理员角色关联项。
//
// 说明：
// 1. 当前用于管理员角色绑定关系展示与写操作响应；
// 2. 先聚焦 admin_user_id / role_id 维度；
// 3. 若后续需要返回角色 code / name，可在 service 层聚合后扩展字段。
type AdminUserRoleItem struct {
	ID          uint64 `json:"id"`
	TenantID    uint64 `json:"tenant_id"`
	AdminUserID uint64 `json:"admin_user_id"`
	RoleID      uint64 `json:"role_id"`
	CreatedAt   uint64 `json:"created_at"`
	UpdatedAt   uint64 `json:"updated_at"`
}

// ListAdminUserRolesRequest 查询单个管理员角色关联请求。
type ListAdminUserRolesRequest struct {
	TenantID    uint64 `json:"tenant_id"`
	AdminUserID uint64 `json:"admin_user_id"`
}

// ListAdminUserRolesResponse 查询单个管理员角色关联响应。
type ListAdminUserRolesResponse struct {
	Items []AdminUserRoleItem `json:"items"`
}

// ListAdminUserRoleIDsRequest 查询单个管理员 role_id 列表请求。
type ListAdminUserRoleIDsRequest struct {
	TenantID    uint64 `json:"tenant_id"`
	AdminUserID uint64 `json:"admin_user_id"`
}

// ListAdminUserRoleIDsResponse 查询单个管理员 role_id 列表响应。
type ListAdminUserRoleIDsResponse struct {
	AdminUserID uint64   `json:"admin_user_id"`
	RoleIDs     []uint64 `json:"role_ids"`
	RoleCount   int      `json:"role_count"`
}

// GrantAdminUserRolesRequest 给管理员追加角色请求。
type GrantAdminUserRolesRequest struct {
	TenantID    uint64   `json:"tenant_id"`
	AdminUserID uint64   `json:"admin_user_id"`
	RoleIDs     []uint64 `json:"role_ids"`
}

// RevokeAdminUserRolesRequest 删除管理员下指定角色请求。
type RevokeAdminUserRolesRequest struct {
	TenantID    uint64   `json:"tenant_id"`
	AdminUserID uint64   `json:"admin_user_id"`
	RoleIDs     []uint64 `json:"role_ids"`
}

// ReplaceAdminUserRolesRequest 全量替换管理员角色请求。
type ReplaceAdminUserRolesRequest struct {
	TenantID    uint64   `json:"tenant_id"`
	AdminUserID uint64   `json:"admin_user_id"`
	RoleIDs     []uint64 `json:"role_ids"`
}

// ClearAdminUserRolesRequest 清空管理员全部角色请求。
type ClearAdminUserRolesRequest struct {
	TenantID    uint64 `json:"tenant_id"`
	AdminUserID uint64 `json:"admin_user_id"`
}

// AdminUserRoleMutationResponse 管理员角色写操作统一响应。
type AdminUserRoleMutationResponse struct {
	AdminUserID uint64   `json:"admin_user_id"`
	RoleIDs     []uint64 `json:"role_ids"`
	RoleCount   int      `json:"role_count"`
}

// ListAdminUserRolesByAdminUserIDsRequest 查询多管理员角色关联请求。
type ListAdminUserRolesByAdminUserIDsRequest struct {
	TenantID     uint64   `json:"tenant_id"`
	AdminUserIDs []uint64 `json:"admin_user_ids"`
}

// ListAdminUserRolesByAdminUserIDsResponse 查询多管理员角色关联响应。
type ListAdminUserRolesByAdminUserIDsResponse struct {
	Items []AdminUserRoleItem `json:"items"`
}

// ListRoleIDsByAdminUserIDsRequest 查询多管理员去重 role_id 列表请求。
type ListRoleIDsByAdminUserIDsRequest struct {
	TenantID     uint64   `json:"tenant_id"`
	AdminUserIDs []uint64 `json:"admin_user_ids"`
}

// ListRoleIDsByAdminUserIDsResponse 查询多管理员去重 role_id 列表响应。
type ListRoleIDsByAdminUserIDsResponse struct {
	AdminUserIDs []uint64 `json:"admin_user_ids"`
	RoleIDs      []uint64 `json:"role_ids"`
	RoleCount    int      `json:"role_count"`
}

// FromModel 把 model.AdminUserRole 转成 dto.AdminUserRoleItem。
func FromModel(in userrolemodel.AdminUserRole) AdminUserRoleItem {
	return AdminUserRoleItem{
		ID:          in.ID,
		TenantID:    in.TenantID,
		AdminUserID: in.AdminUserID,
		RoleID:      in.RoleID,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
	}
}

// FromModels 批量转换管理员角色关联项。
func FromModels(in []userrolemodel.AdminUserRole) []AdminUserRoleItem {
	if len(in) == 0 {
		return []AdminUserRoleItem{}
	}
	out := make([]AdminUserRoleItem, 0, len(in))
	for _, item := range in {
		out = append(out, FromModel(item))
	}
	return out
}
