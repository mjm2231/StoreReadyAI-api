package service

import (
	"context"
	"errors"
	"time"

	userrolesdto "storeready_ai/internal/admin/modules/userroles/dto"
	userrolemodel "storeready_ai/internal/admin/modules/userroles/model"
	userrolesrepo "storeready_ai/internal/admin/modules/userroles/repo"
)

var (
	ErrNilRepository = errors.New("user role service repository is nil")
)

// Service 是后台管理员角色服务接口。
//
// 说明：
// 1. 当前围绕 admin_user_roles 提供查询、绑定、移除、替换能力；
// 2. service 对外优先返回 dto，避免直接暴露 model；
// 3. 当前先聚焦 admin_user_id / role_id 维度；
// 4. 若后续需要直接返回角色 code / name，可在 service 层聚合扩展。
type Service interface {
	// ListByAdminUserID 查询单个管理员角色关联。
	ListByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]userrolesdto.AdminUserRoleItem, error)

	// ListByAdminUserIDs 查询多个管理员角色关联。
	ListByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]userrolesdto.AdminUserRoleItem, error)

	// ListRoleIDsByAdminUserID 查询单个管理员 role_id 列表。
	ListRoleIDsByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]uint64, error)

	// ListRoleIDsByAdminUserIDs 查询多个管理员去重后的 role_id 列表。
	ListRoleIDsByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]uint64, error)

	// GrantRoles 给单个管理员追加角色。
	//
	// 说明：
	// 1. 会先查已有 role_id，并对新增项去重；
	// 2. 已存在的角色不会重复写入；
	// 3. roleIDs 为空时直接返回 nil。
	GrantRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error

	// RevokeRoles 删除单个管理员下指定 role_id 的关联。
	RevokeRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error

	// ClearAdminUserRoles 清空单个管理员的全部角色。
	ClearAdminUserRoles(ctx context.Context, tenantID, adminUserID uint64) error

	// ReplaceAdminUserRoles 全量替换单个管理员角色。
	ReplaceAdminUserRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error
}

type service struct {
	repo userrolesrepo.Repository
}

var _ Service = (*service)(nil)

func New(repo userrolesrepo.Repository) (Service, error) {
	if repo == nil {
		return nil, ErrNilRepository
	}
	return &service{repo: repo}, nil
}

func (s *service) ListByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]userrolesdto.AdminUserRoleItem, error) {
	items, err := s.repo.ListByAdminUserID(ctx, tenantID, adminUserID)
	if err != nil {
		return nil, err
	}
	return userrolesdto.FromModels(items), nil
}

func (s *service) ListByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]userrolesdto.AdminUserRoleItem, error) {
	items, err := s.repo.ListByAdminUserIDs(ctx, tenantID, adminUserIDs)
	if err != nil {
		return nil, err
	}
	return userrolesdto.FromModels(items), nil
}

func (s *service) ListRoleIDsByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]uint64, error) {
	return s.repo.ListRoleIDsByAdminUserID(ctx, tenantID, adminUserID)
}

func (s *service) ListRoleIDsByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]uint64, error) {
	return s.repo.ListRoleIDsByAdminUserIDs(ctx, tenantID, adminUserIDs)
}

func (s *service) GrantRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error {
	if tenantID == 0 || adminUserID == 0 {
		return nil
	}
	roleIDs = uniqueUint64s(roleIDs)
	if len(roleIDs) == 0 {
		return nil
	}

	existingIDs, err := s.repo.ListRoleIDsByAdminUserID(ctx, tenantID, adminUserID)
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
	items := make([]userrolesdto.AdminUserRoleItem, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID == 0 {
			continue
		}
		if _, ok := existingSet[roleID]; ok {
			continue
		}
		items = append(items, userrolesdto.AdminUserRoleItem{
			TenantID:    tenantID,
			AdminUserID: adminUserID,
			RoleID:      roleID,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	if len(items) == 0 {
		return nil
	}

	models := make([]userrolemodel.AdminUserRole, 0, len(items))
	for _, item := range items {
		models = append(models, userrolemodel.AdminUserRole{
			ID:          item.ID,
			TenantID:    item.TenantID,
			AdminUserID: item.AdminUserID,
			RoleID:      item.RoleID,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return s.repo.CreateBatch(ctx, models)
}

func (s *service) RevokeRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error {
	return s.repo.DeleteByAdminUserIDAndRoleIDs(ctx, tenantID, adminUserID, roleIDs)
}

func (s *service) ClearAdminUserRoles(ctx context.Context, tenantID, adminUserID uint64) error {
	return s.repo.DeleteByAdminUserID(ctx, tenantID, adminUserID)
}

func (s *service) ReplaceAdminUserRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error {
	now := uint64(time.Now().Unix())
	return s.repo.ReplaceAdminUserRoles(ctx, tenantID, adminUserID, roleIDs, now)
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
