package repo

import (
	"context"
	"strings"

	"gorm.io/gorm"

	rolemodel "storeready_ai/internal/admin/modules/roles/model"
)

// ListFilter 是后台角色列表查询条件。
//
// 说明：
// 1. 当前覆盖 admin 后台角色管理最常见的筛选项；
// 2. nil 字段表示不参与筛选；
// 3. 默认只查询 deleted_at=0 的数据，避免软删除数据误入正常流程。
type ListFilter struct {
	Keyword        string
	Status         *uint8
	IsSystem       *uint8
	IDs            []uint64
	ExcludeDeleted bool
	Offset         int
	Limit          int
}

func (f ListFilter) Normalize() ListFilter {
	f.Keyword = strings.TrimSpace(f.Keyword)
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if !f.ExcludeDeleted {
		f.ExcludeDeleted = true
	}
	if len(f.IDs) > 0 {
		ids := make([]uint64, 0, len(f.IDs))
		seen := make(map[uint64]struct{}, len(f.IDs))
		for _, id := range f.IDs {
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
			f.IDs = nil
		} else {
			f.IDs = ids
		}
	}
	return f
}

// Repository 是后台角色仓储接口。
//
// 约定：
// 1. admin 模块统一采用“接口 + 实现”结构；
// 2. service 依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 查询默认过滤 deleted_at=0，避免软删除数据误入正常流程。
type Repository interface {
	// DB 返回当前仓储使用的底层数据库连接。
	DB() *gorm.DB
	// WithDB 基于指定数据库连接返回新的仓储实例，便于事务内复用。
	WithDB(db *gorm.DB) Repository

	// Create 创建角色。
	Create(ctx context.Context, role *rolemodel.AdminRole) error
	// Update 更新角色。
	Update(ctx context.Context, role *rolemodel.AdminRole) error
	// Save 保存角色。
	Save(ctx context.Context, role *rolemodel.AdminRole) error

	// GetByID 按主键获取角色。
	GetByID(ctx context.Context, tenantID uint64, id uint64) (*rolemodel.AdminRole, error)
	// GetByIDs 按主键批量获取角色。
	GetByIDs(ctx context.Context, tenantID uint64, ids []uint64) ([]*rolemodel.AdminRole, error)
	// GetByCode 按角色编码获取角色。
	GetByCode(ctx context.Context, tenantID uint64, code string) (*rolemodel.AdminRole, error)

	// ExistsByCode 检查角色编码是否已存在。
	ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error)

	// Count 统计角色数量。
	Count(ctx context.Context, tenantID uint64, filter ListFilter) (int64, error)
	// List 查询角色列表。
	List(ctx context.Context, tenantID uint64, filter ListFilter) ([]*rolemodel.AdminRole, error)

	// UpdateStatus 更新角色状态。
	UpdateStatus(ctx context.Context, tenantID uint64, id uint64, status uint8, updatedAt uint64) error
	// SoftDelete 软删除角色。
	SoftDelete(ctx context.Context, tenantID uint64, id uint64, deletedAt uint64, updatedAt uint64) error
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
	return r.db.WithContext(ctx).Model(&rolemodel.AdminRole{})
}

func (r *repository) activeModel(ctx context.Context, tenantID uint64) *gorm.DB {
	return r.model(ctx).Where("tenant_id = ? AND deleted_at = 0", tenantID)
}

func (r *repository) updateActiveByID(ctx context.Context, tenantID uint64, id uint64, values map[string]interface{}) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.activeModel(ctx, tenantID).
		Where("id = ?", id).
		Updates(values).Error
}

func (r *repository) Create(ctx context.Context, role *rolemodel.AdminRole) error {
	if r == nil || r.db == nil || role == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *repository) Update(ctx context.Context, role *rolemodel.AdminRole) error {
	if r == nil || r.db == nil || role == nil {
		return gorm.ErrInvalidDB
	}
	return r.updateActiveByID(ctx, role.TenantID, role.ID, map[string]interface{}{
		"name":       role.Name,
		"code":       role.Code,
		"status":     role.Status,
		"sort":       role.Sort,
		"is_system":  role.IsSystem,
		"remark":     role.Remark,
		"updated_at": role.UpdatedAt,
		"deleted_at": role.DeletedAt,
	})
}

func (r *repository) Save(ctx context.Context, role *rolemodel.AdminRole) error {
	if r == nil || r.db == nil || role == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID uint64, id uint64) (*rolemodel.AdminRole, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var role rolemodel.AdminRole
	err := r.activeModel(ctx, tenantID).Where("id = ?", id).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *repository) GetByIDs(ctx context.Context, tenantID uint64, ids []uint64) ([]*rolemodel.AdminRole, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	ids = cleanIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*rolemodel.AdminRole
	err := r.activeModel(ctx, tenantID).
		Where("id IN ?", ids).
		Order("sort ASC, id DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list, nil
}

func (r *repository) GetByCode(ctx context.Context, tenantID uint64, code string) (*rolemodel.AdminRole, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	code = strings.TrimSpace(code)
	var role rolemodel.AdminRole
	err := r.activeModel(ctx, tenantID).
		Where("code = ?", code).
		First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *repository) ExistsByCode(ctx context.Context, tenantID uint64, code string, excludeID uint64) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	code = strings.TrimSpace(code)
	query := r.activeModel(ctx, tenantID).
		Model(&rolemodel.AdminRole{}).
		Where("code = ?", code)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) Count(ctx context.Context, tenantID uint64, filter ListFilter) (int64, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	filter = filter.Normalize()
	query := r.applyListFilter(r.model(ctx).Where("tenant_id = ?", tenantID), filter)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) List(ctx context.Context, tenantID uint64, filter ListFilter) ([]*rolemodel.AdminRole, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	filter = filter.Normalize()
	var list []*rolemodel.AdminRole
	query := r.applyListFilter(r.model(ctx).Where("tenant_id = ?", tenantID), filter).
		Order("sort ASC, id DESC").
		Offset(filter.Offset).
		Limit(filter.Limit)
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list, nil
}

func (r *repository) UpdateStatus(ctx context.Context, tenantID uint64, id uint64, status uint8, updatedAt uint64) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.updateActiveByID(ctx, tenantID, id, map[string]interface{}{
		"status":     status,
		"updated_at": updatedAt,
	})
}

func (r *repository) SoftDelete(ctx context.Context, tenantID uint64, id uint64, deletedAt uint64, updatedAt uint64) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.updateActiveByID(ctx, tenantID, id, map[string]interface{}{
		"deleted_at": deletedAt,
		"updated_at": updatedAt,
	})
}

func (r *repository) applyListFilter(query *gorm.DB, filter ListFilter) *gorm.DB {
	if query == nil {
		return nil
	}
	if filter.ExcludeDeleted {
		query = query.Where("deleted_at = 0")
	}
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.IsSystem != nil {
		query = query.Where("is_system = ?", *filter.IsSystem)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR remark LIKE ?", like, like, like)
	}
	return query
}

func cleanIDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return nil
	}
	out := make([]uint64, 0, len(ids))
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
