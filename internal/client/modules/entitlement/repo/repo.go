package repo

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"storeready_ai/internal/client/modules/entitlement/model"
)

// EntitlementRepo 用户权益（VIP）持久化接口（Repo 层）。
//
// 说明：
// - 与传输层无关（HTTP / 后期 gRPC 可复用同一套接口）。
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
// - 本表允许多条记录共存（用于历史/审计）。
//
// MVP：tenant_id 固定为 0，但接口仍保留字段便于未来扩展。
type EntitlementRepo interface {
	// GetLatest 获取某用户某权益的最新一条记录（按 updated_at/created_at/id 倒序）。
	GetLatest(ctx context.Context, tenantID, userID uint64, entitlement string) (*model.UserEntitlement, error)

	// GetActive 获取当前生效的权益（status=active 且 expired_at > now）。
	// 若没有则返回 nil, nil。
	GetActive(ctx context.Context, tenantID, userID uint64, entitlement string, nowUnix uint64) (*model.UserEntitlement, error)

	// UpsertByRef 幂等写入（ref_id 非空时）。
	// 基于 uk_ent_ref(tenant_id,entitlement,ref_id) 做 UPSERT。
	// 注意：ref_id 为空时不应使用该方法（MySQL 允许多个 NULL，会失去幂等意义）。
	UpsertByRef(ctx context.Context, row *model.UserEntitlement) (*model.UserEntitlement, error)

	// Create 直接创建一条记录（适用于 manual/promo 且无 ref_id 的场景）。
	Create(ctx context.Context, row *model.UserEntitlement) (*model.UserEntitlement, error)

	// UpdateStatus 按 id 更新 status（可用于 revoke/expire）。
	UpdateStatus(ctx context.Context, tenantID, userID, id uint64, status uint8, nowUnix uint64) error

	// ExpireActiveIfNeeded 将已过期的 active 记录批量置为 expired（可选：用于定时/查询前修正）。
	ExpireActiveIfNeeded(ctx context.Context, tenantID, userID uint64, entitlement string, nowUnix uint64) (int64, error)
}

// gormRepo EntitlementRepo 的 GORM 实现（对外隐藏具体类型，只暴露接口）。
type gormRepo struct {
	db *gorm.DB
}

// New 创建 Repo（返回接口，便于 mock / 替换实现）。
func New(db *gorm.DB) EntitlementRepo {
	return &gormRepo{db: db}
}

// nowSec 当前时间（Unix 秒）。
func nowSec() uint64 { return uint64(time.Now().Unix()) }

func normalizeEntitlement(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return model.EntitlementVIP
	}
	return v
}

func (r *gormRepo) GetLatest(ctx context.Context, tenantID, userID uint64, entitlement string) (*model.UserEntitlement, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	entitlement = normalizeEntitlement(entitlement)

	var row model.UserEntitlement
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND entitlement = ?", tenantID, userID, entitlement).
		Order("updated_at desc, created_at desc, id desc").
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *gormRepo) GetActive(ctx context.Context, tenantID, userID uint64, entitlement string, nowUnix uint64) (*model.UserEntitlement, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	entitlement = normalizeEntitlement(entitlement)
	if nowUnix == 0 {
		nowUnix = nowSec()
	}

	// 可选：查询前顺手把过期的 active 置 expired（不强依赖，失败不影响主流程）
	_, _ = r.ExpireActiveIfNeeded(ctx, tenantID, userID, entitlement, nowUnix)

	var row model.UserEntitlement
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND entitlement = ?", tenantID, userID, entitlement).
		Where("status = ?", model.EntStatusActive).
		Where("expired_at = 0 OR expired_at > ?", nowUnix).
		Order("expired_at desc, started_at desc, updated_at desc, id desc").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *gormRepo) UpsertByRef(ctx context.Context, row *model.UserEntitlement) (*model.UserEntitlement, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if row == nil {
		return nil, errors.New("row 不能为空")
	}
	if row.TenantID == 0 || row.UserID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	row.Entitlement = normalizeEntitlement(row.Entitlement)
	if row.RefID == nil || strings.TrimSpace(*row.RefID) == "" {
		return nil, errors.New("ref_id 不能为空（幂等写入必须有 ref_id）")
	}
	ref := strings.TrimSpace(*row.RefID)
	row.RefID = &ref

	now := nowSec()
	if row.CreatedAt == 0 {
		row.CreatedAt = now
	}
	row.UpdatedAt = now

	// 以 uk_ent_ref(tenant_id,entitlement,ref_id) 为准做 UPSERT
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "entitlement"}, {Name: "ref_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"user_id",
				"source",
				"status",
				"started_at",
				"expired_at",
				"auto_renew",
				"updated_at",
			}),
		}).
		Create(row).Error
	if err != nil {
		return nil, err
	}

	// 尽量返回带 ID 的记录
	if row.ID != 0 {
		return row, nil
	}

	var out model.UserEntitlement
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND entitlement = ? AND ref_id = ?", row.TenantID, row.Entitlement, *row.RefID).
		Order("id desc").
		First(&out).Error; err != nil {
		return row, nil
	}
	return &out, nil
}

func (r *gormRepo) Create(ctx context.Context, row *model.UserEntitlement) (*model.UserEntitlement, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repo 未初始化")
	}
	if row == nil {
		return nil, errors.New("row 不能为空")
	}
	if row.TenantID == 0 || row.UserID == 0 {
		return nil, errors.New("tenant_id/user_id 非法")
	}
	row.Entitlement = normalizeEntitlement(row.Entitlement)

	now := nowSec()
	if row.CreatedAt == 0 {
		row.CreatedAt = now
	}
	row.UpdatedAt = now
	if row.Status == 0 {
		row.Status = model.EntStatusActive
	}

	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (r *gormRepo) UpdateStatus(ctx context.Context, tenantID, userID, id uint64, status uint8, nowUnix uint64) error {
	if r == nil || r.db == nil {
		return errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 || id == 0 {
		return errors.New("tenant_id/user_id/id 非法")
	}
	if status == 0 {
		return errors.New("status 非法")
	}
	if nowUnix == 0 {
		nowUnix = nowSec()
	}

	return r.db.WithContext(ctx).
		Model(&model.UserEntitlement{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		Updates(map[string]any{"status": status, "updated_at": nowUnix}).Error
}

func (r *gormRepo) ExpireActiveIfNeeded(ctx context.Context, tenantID, userID uint64, entitlement string, nowUnix uint64) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repo 未初始化")
	}
	if tenantID == 0 || userID == 0 {
		return 0, errors.New("tenant_id/user_id 非法")
	}
	entitlement = normalizeEntitlement(entitlement)
	if nowUnix == 0 {
		nowUnix = nowSec()
	}

	res := r.db.WithContext(ctx).
		Model(&model.UserEntitlement{}).
		Where("tenant_id = ? AND user_id = ? AND entitlement = ?", tenantID, userID, entitlement).
		Where("status = ?", model.EntStatusActive).
		Where("expired_at > 0 AND expired_at <= ?", nowUnix).
		Updates(map[string]any{"status": model.EntStatusExpired, "updated_at": nowUnix})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
