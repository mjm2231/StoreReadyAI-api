package repo

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"

	userrolemodel "storeready_ai/internal/admin/modules/userroles/model"
)

var ErrNilDB = errors.New("user role repo db is nil")

// Repository 是后台管理员角色关联仓储接口。
//
// 说明：
// 1. 围绕 admin_user_roles 提供管理员绑定角色、查询角色、移除角色等能力；
// 2. service 依赖接口而非具体实现，便于测试与后续替换；
// 3. 当前先聚焦 admin_user_id / role_id 维度；
// 4. 若后续需要 join 角色详情，可在 service 层或专用查询 repo 中扩展。
type Repository interface {
	DB() *gorm.DB
	WithDB(db *gorm.DB) Repository

	// ListByAdminUserID 查询单个管理员的角色关联。
	ListByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]userrolemodel.AdminUserRole, error)

	// ListByAdminUserIDs 查询多个管理员的角色关联。
	ListByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]userrolemodel.AdminUserRole, error)

	// ListRoleIDsByAdminUserID 查询单个管理员的 role_id 列表。
	ListRoleIDsByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]uint64, error)

	// ListRoleIDsByAdminUserIDs 查询多个管理员去重后的 role_id 列表。
	ListRoleIDsByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]uint64, error)

	// CreateBatch 批量新增管理员角色关联。
	CreateBatch(ctx context.Context, items []userrolemodel.AdminUserRole) error

	// DeleteByAdminUserID 删除单个管理员的全部角色关联。
	DeleteByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) error

	// DeleteByAdminUserIDs 删除多个管理员的全部角色关联。
	DeleteByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) error

	// DeleteByAdminUserIDAndRoleIDs 删除单个管理员下指定 role_id 的关联。
	DeleteByAdminUserIDAndRoleIDs(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error

	// ReplaceAdminUserRoles 全量替换管理员角色。
	//
	// 说明：
	// 1. 先删后插；
	// 2. 若 roleIDs 为空，则等价于清空该管理员所有角色；
	// 3. 若需要与其它写操作保持原子性，请外层传 tx 后通过 WithDB(tx) 调用。
	ReplaceAdminUserRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64, now uint64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) DB() *gorm.DB {
	if r == nil {
		return nil
	}
	return r.db
}

func (r *repository) WithDB(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) model(ctx context.Context) *gorm.DB {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.WithContext(ctx).Model(&userrolemodel.AdminUserRole{})
}

func (r *repository) ListByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]userrolemodel.AdminUserRole, error) {
	if tenantID == 0 || adminUserID == 0 {
		return []userrolemodel.AdminUserRole{}, nil
	}
	return r.ListByAdminUserIDs(ctx, tenantID, []uint64{adminUserID})
}

func (r *repository) ListByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]userrolemodel.AdminUserRole, error) {
	mdb := r.model(ctx)
	if mdb == nil {
		return nil, ErrNilDB
	}
	adminUserIDs = uniqueUint64s(adminUserIDs)
	if tenantID == 0 || len(adminUserIDs) == 0 {
		return []userrolemodel.AdminUserRole{}, nil
	}

	var items []userrolemodel.AdminUserRole
	err := mdb.
		Where("tenant_id = ?", tenantID).
		Where("admin_user_id IN ?", adminUserIDs).
		Order("admin_user_id ASC, role_id ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (r *repository) ListRoleIDsByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) ([]uint64, error) {
	if tenantID == 0 || adminUserID == 0 {
		return []uint64{}, nil
	}
	return r.ListRoleIDsByAdminUserIDs(ctx, tenantID, []uint64{adminUserID})
}

func (r *repository) ListRoleIDsByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) ([]uint64, error) {
	fmt.Println("ListRoleIDsByAdminUserIDs tenantID:", tenantID, "adminUserIDs:", adminUserIDs)
	items, err := r.ListByAdminUserIDs(ctx, tenantID, adminUserIDs)
	if err != nil {
		return nil, err
	}
	roleIDs := make([]uint64, 0, len(items))
	seen := make(map[uint64]struct{}, len(items))
	for _, item := range items {
		if item.RoleID == 0 {
			continue
		}
		if _, ok := seen[item.RoleID]; ok {
			continue
		}
		seen[item.RoleID] = struct{}{}
		roleIDs = append(roleIDs, item.RoleID)
	}
	sort.Slice(roleIDs, func(i, j int) bool { return roleIDs[i] < roleIDs[j] })
	return roleIDs, nil
}

func (r *repository) CreateBatch(ctx context.Context, items []userrolemodel.AdminUserRole) error {
	mdb := r.model(ctx)
	if mdb == nil {
		return ErrNilDB
	}
	items = normalizeAdminUserRoleItems(items)
	if len(items) == 0 {
		return nil
	}
	return mdb.Create(&items).Error
}

func (r *repository) DeleteByAdminUserID(ctx context.Context, tenantID, adminUserID uint64) error {
	if tenantID == 0 || adminUserID == 0 {
		return nil
	}
	return r.DeleteByAdminUserIDs(ctx, tenantID, []uint64{adminUserID})
}

func (r *repository) DeleteByAdminUserIDs(ctx context.Context, tenantID uint64, adminUserIDs []uint64) error {
	mdb := r.model(ctx)
	if mdb == nil {
		return ErrNilDB
	}
	adminUserIDs = uniqueUint64s(adminUserIDs)
	if tenantID == 0 || len(adminUserIDs) == 0 {
		return nil
	}
	return mdb.
		Where("tenant_id = ?", tenantID).
		Where("admin_user_id IN ?", adminUserIDs).
		Delete(&userrolemodel.AdminUserRole{}).Error
}

func (r *repository) DeleteByAdminUserIDAndRoleIDs(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64) error {
	mdb := r.model(ctx)
	if mdb == nil {
		return ErrNilDB
	}
	roleIDs = uniqueUint64s(roleIDs)
	if tenantID == 0 || adminUserID == 0 || len(roleIDs) == 0 {
		return nil
	}
	return mdb.
		Where("tenant_id = ?", tenantID).
		Where("admin_user_id = ?", adminUserID).
		Where("role_id IN ?", roleIDs).
		Delete(&userrolemodel.AdminUserRole{}).Error
}

func (r *repository) ReplaceAdminUserRoles(ctx context.Context, tenantID, adminUserID uint64, roleIDs []uint64, now uint64) error {
	if tenantID == 0 || adminUserID == 0 {
		return nil
	}
	if err := r.DeleteByAdminUserID(ctx, tenantID, adminUserID); err != nil {
		return err
	}
	roleIDs = uniqueUint64s(roleIDs)
	if len(roleIDs) == 0 {
		return nil
	}

	items := make([]userrolemodel.AdminUserRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		if roleID == 0 {
			continue
		}
		items = append(items, userrolemodel.AdminUserRole{
			TenantID:    tenantID,
			AdminUserID: adminUserID,
			RoleID:      roleID,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return r.CreateBatch(ctx, items)
}

func normalizeAdminUserRoleItems(items []userrolemodel.AdminUserRole) []userrolemodel.AdminUserRole {
	if len(items) == 0 {
		return nil
	}
	out := make([]userrolemodel.AdminUserRole, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.TenantID == 0 || item.AdminUserID == 0 || item.RoleID == 0 {
			continue
		}
		key := adminUserRoleKey(item.TenantID, item.AdminUserID, item.RoleID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
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
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func adminUserRoleKey(tenantID, adminUserID, roleID uint64) string {
	return fmtUint64(tenantID) + ":" + fmtUint64(adminUserID) + ":" + fmtUint64(roleID)
}

func fmtUint64(v uint64) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append(buf, byte('0'+v%10))
		v /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
