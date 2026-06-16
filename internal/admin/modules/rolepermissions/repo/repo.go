package repo

import (
	"context"
	"errors"
	"sort"

	"gorm.io/gorm"

	permissionmodel "storeready_ai/internal/admin/modules/rolepermissions/model"
)

var ErrNilDB = errors.New("role permission repo db is nil")

// Repository 是后台角色权限关联仓储接口。
//
// 约定：
// 1. admin 模块统一采用“接口 + 实现”结构；
// 2. service 依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 当前围绕 admin_role_permissions 提供角色授权、查询与替换能力；
// 4. 若后续补充权限树/菜单能力，可继续在此接口上扩展查询方法。
//
// 当前预留的高频能力：
// - 查询单角色/多角色的 permission_id；
// - 删除单角色/多角色的权限关联；
// - 批量新增角色权限关联；
// - 角色权限全量替换（事务由 service 或上层 db 控制）。
type Repository interface {
	DB() *gorm.DB
	WithDB(db *gorm.DB) Repository

	// ListByRoleID 查询单个角色的所有权限关联。
	ListByRoleID(ctx context.Context, tenantID, roleID uint64) ([]permissionmodel.AdminRolePermission, error)

	// ListByRoleIDs 查询多个角色的所有权限关联。
	ListByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]permissionmodel.AdminRolePermission, error)

	// ListPermissionIDsByRoleID 查询单个角色的 permission_id 列表。
	ListPermissionIDsByRoleID(ctx context.Context, tenantID, roleID uint64) ([]uint64, error)

	// ListPermissionIDsByRoleIDs 查询多个角色去重后的 permission_id 列表。
	ListPermissionIDsByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]uint64, error)

	// CreateBatch 批量新增角色权限关联。
	CreateBatch(ctx context.Context, items []permissionmodel.AdminRolePermission) error

	// DeleteByRoleID 删除单个角色的全部权限关联。
	DeleteByRoleID(ctx context.Context, tenantID, roleID uint64) error

	// DeleteByRoleIDs 删除多个角色的全部权限关联。
	DeleteByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) error

	// DeleteByRoleIDAndPermissionIDs 删除单个角色下指定 permission_id 的关联。
	DeleteByRoleIDAndPermissionIDs(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error

	// ReplaceRolePermissions 全量替换角色权限。
	//
	// 说明：
	// 1. 先删后插；
	// 2. 若 permissionIDs 为空，则等价于清空该角色所有权限；
	// 3. 若需要与其它写操作保持原子性，请在外层传入 tx 后再调用 WithDB(tx)。
	ReplaceRolePermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64, now uint64) error
}

// repository 是 Repository 的 GORM 实现。
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
	return r.db.WithContext(ctx).Model(&permissionmodel.AdminRolePermission{})
}

func (r *repository) ListByRoleID(ctx context.Context, tenantID, roleID uint64) ([]permissionmodel.AdminRolePermission, error) {
	if tenantID == 0 || roleID == 0 {
		return []permissionmodel.AdminRolePermission{}, nil
	}
	return r.ListByRoleIDs(ctx, tenantID, []uint64{roleID})
}

func (r *repository) ListByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]permissionmodel.AdminRolePermission, error) {
	mdb := r.model(ctx)
	if mdb == nil {
		return nil, ErrNilDB
	}
	roleIDs = uniqueUint64s(roleIDs)
	if tenantID == 0 || len(roleIDs) == 0 {
		return []permissionmodel.AdminRolePermission{}, nil
	}

	var items []permissionmodel.AdminRolePermission
	err := mdb.
		Where("tenant_id = ?", tenantID).
		Where("role_id IN ?", roleIDs).
		Order("role_id ASC, permission_id ASC, id ASC").
		Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) ListPermissionIDsByRoleID(ctx context.Context, tenantID, roleID uint64) ([]uint64, error) {
	if tenantID == 0 || roleID == 0 {
		return []uint64{}, nil
	}
	return r.ListPermissionIDsByRoleIDs(ctx, tenantID, []uint64{roleID})
}

func (r *repository) ListPermissionIDsByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]uint64, error) {
	items, err := r.ListByRoleIDs(ctx, tenantID, roleIDs)
	if err != nil {
		return nil, err
	}
	permissionIDs := make([]uint64, 0, len(items))
	seen := make(map[uint64]struct{}, len(items))
	for _, item := range items {
		if item.PermissionID == 0 {
			continue
		}
		if _, ok := seen[item.PermissionID]; ok {
			continue
		}
		seen[item.PermissionID] = struct{}{}
		permissionIDs = append(permissionIDs, item.PermissionID)
	}
	sort.Slice(permissionIDs, func(i, j int) bool { return permissionIDs[i] < permissionIDs[j] })
	return permissionIDs, nil
}

func (r *repository) CreateBatch(ctx context.Context, items []permissionmodel.AdminRolePermission) error {
	mdb := r.model(ctx)
	if mdb == nil {
		return ErrNilDB
	}
	items = normalizeRolePermissionItems(items)
	if len(items) == 0 {
		return nil
	}
	return mdb.Create(&items).Error
}

func (r *repository) DeleteByRoleID(ctx context.Context, tenantID, roleID uint64) error {
	if tenantID == 0 || roleID == 0 {
		return nil
	}
	return r.DeleteByRoleIDs(ctx, tenantID, []uint64{roleID})
}

func (r *repository) DeleteByRoleIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) error {
	mdb := r.model(ctx)
	if mdb == nil {
		return ErrNilDB
	}
	roleIDs = uniqueUint64s(roleIDs)
	if tenantID == 0 || len(roleIDs) == 0 {
		return nil
	}
	return mdb.
		Where("tenant_id = ?", tenantID).
		Where("role_id IN ?", roleIDs).
		Delete(&permissionmodel.AdminRolePermission{}).Error
}

func (r *repository) DeleteByRoleIDAndPermissionIDs(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64) error {
	mdb := r.model(ctx)
	if mdb == nil {
		return ErrNilDB
	}
	permissionIDs = uniqueUint64s(permissionIDs)
	if tenantID == 0 || roleID == 0 || len(permissionIDs) == 0 {
		return nil
	}
	return mdb.
		Where("tenant_id = ?", tenantID).
		Where("role_id = ?", roleID).
		Where("permission_id IN ?", permissionIDs).
		Delete(&permissionmodel.AdminRolePermission{}).Error
}

func (r *repository) ReplaceRolePermissions(ctx context.Context, tenantID, roleID uint64, permissionIDs []uint64, now uint64) error {
	if tenantID == 0 || roleID == 0 {
		return nil
	}
	if err := r.DeleteByRoleID(ctx, tenantID, roleID); err != nil {
		return err
	}
	permissionIDs = uniqueUint64s(permissionIDs)
	if len(permissionIDs) == 0 {
		return nil
	}

	items := make([]permissionmodel.AdminRolePermission, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		if permissionID == 0 {
			continue
		}
		items = append(items, permissionmodel.AdminRolePermission{
			TenantID:     tenantID,
			RoleID:       roleID,
			PermissionID: permissionID,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	return r.CreateBatch(ctx, items)
}

func normalizeRolePermissionItems(items []permissionmodel.AdminRolePermission) []permissionmodel.AdminRolePermission {
	if len(items) == 0 {
		return nil
	}
	out := make([]permissionmodel.AdminRolePermission, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.TenantID == 0 || item.RoleID == 0 || item.PermissionID == 0 {
			continue
		}
		key := rolePermissionKey(item.TenantID, item.RoleID, item.PermissionID)
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

func rolePermissionKey(tenantID, roleID, permissionID uint64) string {
	return fmtUint64(tenantID) + ":" + fmtUint64(roleID) + ":" + fmtUint64(permissionID)
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
