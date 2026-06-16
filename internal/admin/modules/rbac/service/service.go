package service

import (
	"context"
	"errors"
	"fmt"
	"storeready_ai/internal/admin/modules/rbac/dto"
	adminRoleDto "storeready_ai/internal/admin/modules/roles/dto"
	adminUserDto "storeready_ai/internal/admin/modules/user/dto"
)

type AdminUserService interface {
	GetByID(ctx context.Context, tenantID uint64, id uint64) (*adminUserDto.AdminUserItem, error)
}

type AdminUserRolesService interface {
	// ListRoleIDsByAdminUserID 查询单个管理员 role_id 列表。
	ListRoleIDsByAdminUserID(ctx context.Context, tenantID, ID uint64) ([]uint64, error)
}

type AdminRolesService interface {
	// ListRolesByIds 查询管理员角色列表。
	ListRolesByIds(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]adminRoleDto.RoleItem, error)
}

type AdminRolePermissionService interface {
	// ListPermissionIDsByRoleIDs 查询多个角色去重后的 permission_id 列表。
	ListPermissionIDsByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]uint64, error)
}

type AdminPermissionService interface {
	// ListPermissionCodesByIds 查询管理员权限列表。
	ListPermissionCodesByIds(ctx context.Context, tenantID uint64, perIds []uint64) ([]string, error)
	// ListRolesByUser(ctx context.Context, tenantID uint64, ID uint64) ([]dto.RoleInfo, error)
}
type Service interface {
	GetByID(ctx context.Context, tenantID uint64, id uint64) (*dto.AdminUserItem, error)
	GetSnapshot(ctx context.Context, tenantID, ID uint64) (*dto.Snapshot, error)
	GetRoleCodesByAdminUserID(ctx context.Context, tenantID uint64, ID uint64) ([]string, error)
}

type service struct {
	adminUsersv           AdminUserService
	adminUserRolessv      AdminUserRolesService
	adminRolessv          AdminRolesService
	adminRolePermissionsv AdminRolePermissionService
	adminPermissionsv     AdminPermissionService
}

func New(adminUsersv AdminUserService, adminUserRolessv AdminUserRolesService, adminRolessv AdminRolesService, adminRolePermissionsv AdminRolePermissionService, adminPermissionsv AdminPermissionService) Service {
	return &service{
		adminUsersv:           adminUsersv,
		adminUserRolessv:      adminUserRolessv,
		adminRolessv:          adminRolessv,
		adminRolePermissionsv: adminRolePermissionsv,
		adminPermissionsv:     adminPermissionsv,
	}
}

func (s *service) GetByID(ctx context.Context, tenantID uint64, id uint64) (*dto.AdminUserItem, error) {
	u, err := s.adminUsersv.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	return &dto.AdminUserItem{
		ID:           u.ID,
		Username:     u.Username,
		Nickname:     u.Nickname,
		Avatar:       u.Avatar,
		TenantID:     u.TenantID,
		Status:       u.Status,
		Email:        u.Email,
		Mobile:       u.Mobile,
		IsSuperAdmin: u.IsSuperAdmin,
		LastLoginAt:  u.LastLoginAt,
		LastLoginIP:  u.LastLoginIP,
		Remark:       u.Remark,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}, err

}

func (s *service) GetSnapshot(ctx context.Context, tenantID, ID uint64) (*dto.Snapshot, error) {
	roleIDs, err := s.adminUserRolessv.ListRoleIDsByAdminUserID(ctx, tenantID, ID)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		fmt.Printf("GetSnapshot role ids len==")
		return nil, errors.New("role ids len==0")
	}
	roleItem, err := s.adminRolessv.ListRolesByIds(ctx, tenantID, roleIDs)
	if err != nil {
		return nil, err
	}
	if len(roleItem) == 0 {
		fmt.Printf("GetSnapshot roles len==")
		return nil, errors.New("roles len==0")
	}
	infos := make([]dto.RoleInfo, 0, len(roleItem))
	for _, item := range roleItem {
		infos = append(infos, dto.RoleInfo{
			TenantID:  item.TenantID,
			ID:        item.ID,
			Name:      item.Name,
			Code:      item.Code,
			Status:    item.Status,
			Sort:      item.Sort,
			IsSystem:  item.IsSystem,
			Remark:    item.Remark,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			DeletedAt: item.DeletedAt,
		})
	}
	permsids, err := s.adminRolePermissionsv.ListPermissionIDsByRoleIDs(ctx, tenantID, roleIDs)
	if err != nil {
		return nil, err
	}
	if len(permsids) == 0 {
		fmt.Printf("GetSnapshot permission ids len==")
		return nil, errors.New("permission ids len==0")
	}
	perms, err := s.adminPermissionsv.ListPermissionCodesByIds(ctx, tenantID, permsids)
	if err != nil {
		return nil, err
	}
	if len(perms) == 0 {
		fmt.Printf("GetSnapshot permissions len==")
		return nil, errors.New("permissions len==0")
	}
	snapshot := &dto.Snapshot{
		Roles:           infos,
		PermissionCodes: perms,
	}
	fmt.Printf("GetSnapshot Snapshot: %+v", snapshot)
	return snapshot, nil
}

func (s *service) GetRoleCodesByAdminUserID(ctx context.Context, tenantID, ID uint64) ([]string, error) {
	roleIDs, err := s.adminUserRolessv.ListRoleIDsByAdminUserID(ctx, tenantID, ID)
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return nil, errors.New("role ids len==0")
	}
	roleItem, err := s.adminRolessv.ListRolesByIds(ctx, tenantID, roleIDs)
	if err != nil {
		return nil, err
	}
	roles := make([]string, 0, len(roleItem))
	for _, item := range roleItem {
		roles = append(roles, item.Code)
	}
	return roles, nil
}
