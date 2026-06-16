package service

import (
	"context"
	"errors"
	"time"

	rolepermissiondto "storeready_ai/internal/admin/modules/rolepermissions/dto"
	permissionmodel "storeready_ai/internal/admin/modules/rolepermissions/model"
	rolepermissionrepo "storeready_ai/internal/admin/modules/rolepermissions/repo"
)

var (
	ErrNilRepository = errors.New("role permission service repository is nil")
)

// Service 是后台角色权限服务接口。
//
// 说明：
// 1. 当前围绕 admin_role_permissions 提供查询、授权、撤销、替换能力；
// 2. service 只依赖 repo 接口，便于 app 层装配和单测替换；
// 3. 当前先聚焦 permission_id 维度，后续若需要返回 permission code/tree，可继续扩展；
// 4. 时间统一采用秒级时间戳，与 admin 模块其它表保持一致。
type Service interface {
	// ListByRoleID 查询单个角色的角色权限关联。
	ListByRoleID(ctx context.Context, tenantID, roleID uint64) ([]rolepermissiondto.RolePermissionItem, error)

	// ListByRoleIDs 查询多个角色的角色权限关联。
	ListByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]rolepermissiondto.RolePermissionItem, error)

	// ListPermissionIDsByRoleID 查询单个角色的 permission_id 列表。
	ListPermissionIDsByRoleID(ctx context.Context, tenantID, roleID uint64) ([]uint64, error)

	// ListPermissionIDsByRoleIDs 查询多个角色去重后的 permission_id 列表。
	ListPermissionIDsByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]uint64, error)

	// GrantPermissions 给单个角色追加权限。
	//
	// 说明：
	// 1. 会先查已有 permission_id，并对新增项去重；
	// 2. 已存在的权限不会重复写入；
	// 3. permissionIDs 为空时直接返回 nil。
	GrantPermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error

	// RevokePermissions 删除单个角色下指定 permission_id 的关联。
	RevokePermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error

	// ClearRolePermissions 清空单个角色的全部权限。
	ClearRolePermissions(ctx context.Context, tenantID, roleID uint64) error

	// ReplaceRolePermissions 全量替换单个角色权限。
	ReplaceRolePermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error
}

type service struct {
	repo rolepermissionrepo.Repository
}

var _ Service = (*service)(nil)

func New(repo rolepermissionrepo.Repository) (Service, error) {
	if repo == nil {
		return nil, ErrNilRepository
	}
	return &service{repo: repo}, nil
}

func (s *service) ListByRoleID(ctx context.Context, tenantID, roleID uint64) ([]rolepermissiondto.RolePermissionItem, error) {
	items, err := s.repo.ListByRoleID(ctx, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	return rolepermissiondto.FromModels(items), nil
}

func (s *service) ListByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]rolepermissiondto.RolePermissionItem, error) {
	items, err := s.repo.ListByRoleIDs(ctx, tenantID, roleIDs)
	if err != nil {
		return nil, err
	}
	return rolepermissiondto.FromModels(items), nil
}

func (s *service) ListPermissionIDsByRoleID(ctx context.Context, tenantID, roleID uint64) ([]uint64, error) {
	return s.repo.ListPermissionIDsByRoleID(ctx, tenantID, roleID)
}

func (s *service) ListPermissionIDsByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]uint64, error) {
	return s.repo.ListPermissionIDsByRoleIDs(ctx, tenantID, roleIDs)
}

func (s *service) GrantPermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error {
	if tenantID == 0 || roleID == 0 {
		return nil
	}
	permissionIDs = uniqueUint64s(permissionIDs)
	if len(permissionIDs) == 0 {
		return nil
	}

	existingIDs, err := s.repo.ListPermissionIDsByRoleID(ctx, tenantID, roleID)
	if err != nil {
		return err
	}
	existingSet := make(map[uint64]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		if id == 0 {
			continue
		}
		existingSet[id] = struct{}{}
	}

	now := uint64(time.Now().Unix())
	items := make([]rolepermissiondto.RolePermissionItem, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		if permissionID == 0 {
			continue
		}
		if _, ok := existingSet[permissionID]; ok {
			continue
		}
		items = append(items, rolepermissiondto.RolePermissionItem{
			TenantID:     tenantID,
			RoleID:       roleID,
			PermissionID: permissionID,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	if len(items) == 0 {
		return nil
	}
	models := make([]permissionmodel.AdminRolePermission, 0, len(items))
	for _, item := range items {
		models = append(models, permissionmodel.AdminRolePermission{
			ID:           item.ID,
			TenantID:     item.TenantID,
			RoleID:       item.RoleID,
			PermissionID: item.PermissionID,
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}
	return s.repo.CreateBatch(ctx, models)
}

func (s *service) RevokePermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error {
	return s.repo.DeleteByRoleIDAndPermissionIDs(ctx, tenantID, roleID, permissionIDs)
}

func (s *service) ClearRolePermissions(ctx context.Context, tenantID, roleID uint64) error {
	return s.repo.DeleteByRoleID(ctx, tenantID, roleID)
}

func (s *service) ReplaceRolePermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error {
	now := uint64(time.Now().Unix())
	return s.repo.ReplaceRolePermissions(ctx, tenantID, roleID, permissionIDs, now)
}

func uniqueUint64s(in []uint64) []uint64 {
	if len(in) == 0 {
		return nil
	}
	out := make([]uint64, 0, len(in))
	seen := make(map[uint64]struct{}, len(in))
	for _, item := range in {
		if item == 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
