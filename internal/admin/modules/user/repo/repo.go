package repo

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	usermodel "storeready_ai/internal/admin/modules/user/model"
)

// ListFilter 是后台管理员列表查询条件。
//
// 说明：
// 1. 当前先覆盖 admin 后台最常见的筛选项；
// 2. nil 字段表示不参与筛选；
// 3. 默认只查询 deleted_at=0 的数据，除非显式关闭 ExcludeDeleted。
type ListFilter struct {
	Keyword        string
	Status         *uint8
	IsSuperAdmin   *uint8
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
		// Admin 用户默认应过滤软删除数据。
		f.ExcludeDeleted = true
	}
	if len(f.IDs) > 0 {
		ids := make([]uint64, 0, len(f.IDs))
		for _, id := range f.IDs {
			if id == 0 {
				continue
			}
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

// Repository 是后台管理员用户仓储接口。
//
// 约定：
// 1. 后续 admin 模块统一采用“接口 + 实现”结构；
// 2. service 依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 查询默认过滤 deleted_at=0，避免软删除数据误入正常流程。
type Repository interface {
	DB() *gorm.DB
	WithDB(db *gorm.DB) Repository

	Create(ctx context.Context, user *usermodel.AdminUser) error
	Update(ctx context.Context, user *usermodel.AdminUser) error
	Save(ctx context.Context, user *usermodel.AdminUser) error

	GetByID(ctx context.Context, tenantID uint64, id uint64) (*usermodel.AdminUser, error)
	GetByIDs(ctx context.Context, tenantID uint64, ids []uint64) ([]*usermodel.AdminUser, error)
	GetByUsername(ctx context.Context, tenantID uint64, username string) (*usermodel.AdminUser, error)
	GetByEmail(ctx context.Context, tenantID uint64, email string) (*usermodel.AdminUser, error)

	ExistsByUsername(ctx context.Context, tenantID uint64, username string, excludeID uint64) (bool, error)
	ExistsByEmail(ctx context.Context, tenantID uint64, email string, excludeID uint64) (bool, error)

	Count(ctx context.Context, tenantID uint64, filter ListFilter) (int64, error)
	List(ctx context.Context, tenantID uint64, filter ListFilter) ([]*usermodel.AdminUser, error)

	UpdateLoginInfo(ctx context.Context, tenantID uint64, id uint64, lastLoginAt uint64, lastLoginIP string, updatedAt uint64) error
	UpdatePassword(ctx context.Context, tenantID uint64, id uint64, passwordHash string, updatedAt uint64) error
	UpdateStatus(ctx context.Context, tenantID uint64, id uint64, status uint8, updatedAt uint64) error
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

func (r *repository) Create(ctx context.Context, user *usermodel.AdminUser) error {
	if r == nil || r.db == nil || user == nil {
		return gorm.ErrInvalidDB
	}
	fmt.Printf("Create: tenantID=%d, username=%s\n", user.TenantID, user.Username)
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *repository) Update(ctx context.Context, user *usermodel.AdminUser) error {
	if r == nil || r.db == nil || user == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).
		Model(&usermodel.AdminUser{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", user.TenantID, user.ID).
		Updates(map[string]interface{}{
			"username":       user.Username,
			"password_hash":  user.PasswordHash,
			"nickname":       user.Nickname,
			"email":          user.Email,
			"mobile":         user.Mobile,
			"avatar":         user.Avatar,
			"status":         user.Status,
			"is_super_admin": user.IsSuperAdmin,
			"last_login_at":  user.LastLoginAt,
			"last_login_ip":  user.LastLoginIP,
			"remark":         user.Remark,
			"updated_at":     user.UpdatedAt,
			"deleted_at":     user.DeletedAt,
		}).Error
}

func (r *repository) Save(ctx context.Context, user *usermodel.AdminUser) error {
	if r == nil || r.db == nil || user == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *repository) GetByID(ctx context.Context, tenantID uint64, id uint64) (*usermodel.AdminUser, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var user usermodel.AdminUser
	err := r.baseQuery(ctx, tenantID).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByIDs(ctx context.Context, tenantID uint64, ids []uint64) ([]*usermodel.AdminUser, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	ids = cleanIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*usermodel.AdminUser
	err := r.baseQuery(ctx, tenantID).
		Where("id IN ?", ids).
		Order("id DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list, nil
}

func (r *repository) GetByUsername(ctx context.Context, tenantID uint64, username string) (*usermodel.AdminUser, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	if tenantID == 0 {
		return nil, gorm.ErrInvalidData
	}
	username = strings.TrimSpace(username)
	var user usermodel.AdminUser
	err := r.baseQuery(ctx, tenantID).
		Where("username = ?", username).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByEmail(ctx context.Context, tenantID uint64, email string) (*usermodel.AdminUser, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	email = strings.TrimSpace(email)
	var user usermodel.AdminUser
	err := r.baseQuery(ctx, tenantID).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) ExistsByUsername(ctx context.Context, tenantID uint64, username string, excludeID uint64) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	username = strings.TrimSpace(username)
	query := r.baseQuery(ctx, tenantID).
		Model(&usermodel.AdminUser{}).
		Where("username = ?", username)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *repository) ExistsByEmail(ctx context.Context, tenantID uint64, email string, excludeID uint64) (bool, error) {
	if r == nil || r.db == nil {
		return false, gorm.ErrInvalidDB
	}
	email = strings.TrimSpace(email)
	query := r.baseQuery(ctx, tenantID).
		Model(&usermodel.AdminUser{}).
		Where("email = ?", email)
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
	query := r.applyListFilter(r.db.WithContext(ctx).Model(&usermodel.AdminUser{}).Where("tenant_id = ?", tenantID), filter)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) List(ctx context.Context, tenantID uint64, filter ListFilter) ([]*usermodel.AdminUser, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	filter = filter.Normalize()
	var list []*usermodel.AdminUser
	query := r.applyListFilter(r.db.WithContext(ctx).Model(&usermodel.AdminUser{}).Where("tenant_id = ?", tenantID), filter).
		Order("id DESC").
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

func (r *repository) UpdateLoginInfo(ctx context.Context, tenantID uint64, id uint64, lastLoginAt uint64, lastLoginIP string, updatedAt uint64) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).
		Model(&usermodel.AdminUser{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(map[string]interface{}{
			"last_login_at": lastLoginAt,
			"last_login_ip": strings.TrimSpace(lastLoginIP),
			"updated_at":    updatedAt,
		}).Error
}

func (r *repository) UpdatePassword(ctx context.Context, tenantID uint64, id uint64, passwordHash string, updatedAt uint64) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).
		Model(&usermodel.AdminUser{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(map[string]interface{}{
			"password_hash": strings.TrimSpace(passwordHash),
			"updated_at":    updatedAt,
		}).Error
}

func (r *repository) UpdateStatus(ctx context.Context, tenantID uint64, id uint64, status uint8, updatedAt uint64) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).
		Model(&usermodel.AdminUser{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": updatedAt,
		}).Error
}

func (r *repository) SoftDelete(ctx context.Context, tenantID uint64, id uint64, deletedAt uint64, updatedAt uint64) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).
		Model(&usermodel.AdminUser{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(map[string]interface{}{
			"status":     usermodel.AdminUserStatusDeleted,
			"deleted_at": deletedAt,
			"updated_at": updatedAt,
		}).Error
}

func (r *repository) baseQuery(ctx context.Context, tenantID uint64) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&usermodel.AdminUser{}).
		Where("tenant_id = ? AND deleted_at = 0", tenantID)
}

func (r *repository) applyListFilter(query *gorm.DB, filter ListFilter) *gorm.DB {
	if query == nil {
		return nil
	}
	if filter.ExcludeDeleted {
		query = query.Where("deleted_at = 0")
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ? OR mobile LIKE ?", like, like, like, like)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.IsSuperAdmin != nil {
		query = query.Where("is_super_admin = ?", *filter.IsSuperAdmin)
	}
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
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
