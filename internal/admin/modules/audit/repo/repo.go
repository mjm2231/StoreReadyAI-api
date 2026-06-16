package repo

import (
	"context"
	"strings"

	"gorm.io/gorm"

	auditmodel "storeready_ai/internal/admin/modules/audit/model"
)

// ListFilter 是后台审计日志列表查询条件。
//
// 说明：
// 1. 当前覆盖 admin 后台审计查询最常见的筛选项；
// 2. nil 字段表示不参与筛选；
// 3. 审计日志为追加型数据，不做软删除过滤。
type ListFilter struct {
	RID          string
	TraceID      string
	UID          string
	TenantID     string
	Role         string
	Action       string
	ResourceType string
	ResourceID   string
	Success      *uint8
	HTTPStatus   *int32
	ErrCode      string
	Method       string
	Path         string
	Keyword      string
	CreatedFrom  *int64
	CreatedTo    *int64
	IDs          []uint64
	Offset       int
	Limit        int
}

func (f ListFilter) Normalize() ListFilter {
	f.RID = strings.TrimSpace(f.RID)
	f.TraceID = strings.TrimSpace(f.TraceID)
	f.UID = strings.TrimSpace(f.UID)
	f.TenantID = strings.TrimSpace(f.TenantID)
	f.Role = strings.TrimSpace(f.Role)
	f.Action = strings.TrimSpace(f.Action)
	f.ResourceType = strings.TrimSpace(f.ResourceType)
	f.ResourceID = strings.TrimSpace(f.ResourceID)
	f.ErrCode = strings.TrimSpace(f.ErrCode)
	f.Method = strings.TrimSpace(strings.ToUpper(f.Method))
	f.Path = strings.TrimSpace(f.Path)
	f.Keyword = strings.TrimSpace(f.Keyword)
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 200 {
		f.Limit = 200
	}
	f.IDs = cleanIDs(f.IDs)
	return f
}

// Repository 是后台审计日志仓储接口。
//
// 约定：
// 1. admin 模块统一采用“接口 + 实现”结构；
// 2. service 依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 审计日志默认只追加，不提供更新/删除能力。
type Repository interface {
	// DB 返回当前仓储使用的底层数据库连接。
	DB() *gorm.DB
	// WithDB 基于指定数据库连接返回新的仓储实例，便于事务内复用。
	WithDB(db *gorm.DB) Repository

	// Create 创建审计日志。
	Create(ctx context.Context, log *auditmodel.AuditLog) error
	// BatchCreate 批量创建审计日志。
	BatchCreate(ctx context.Context, logs []*auditmodel.AuditLog) error

	// GetByID 按主键获取审计日志。
	GetByID(ctx context.Context, id uint64) (*auditmodel.AuditLog, error)
	// GetByRID 按请求ID获取审计日志列表。
	GetByRID(ctx context.Context, rid string) ([]*auditmodel.AuditLog, error)

	// Count 统计审计日志数量。
	Count(ctx context.Context, filter ListFilter) (int64, error)
	// List 查询审计日志列表。
	List(ctx context.Context, filter ListFilter) ([]*auditmodel.AuditLog, error)
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
	return r.db.WithContext(ctx).Model(&auditmodel.AuditLog{})
}

func (r *repository) Create(ctx context.Context, log *auditmodel.AuditLog) error {
	if r == nil || r.db == nil || log == nil {
		return gorm.ErrInvalidDB
	}
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *repository) BatchCreate(ctx context.Context, logs []*auditmodel.AuditLog) error {
	if r == nil || r.db == nil {
		return gorm.ErrInvalidDB
	}
	logs = cleanLogs(logs)
	if len(logs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&logs).Error
}

func (r *repository) GetByID(ctx context.Context, id uint64) (*auditmodel.AuditLog, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	var log auditmodel.AuditLog
	err := r.model(ctx).Where("id = ?", id).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *repository) GetByRID(ctx context.Context, rid string) ([]*auditmodel.AuditLog, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	rid = strings.TrimSpace(rid)
	if rid == "" {
		return nil, nil
	}
	var list []*auditmodel.AuditLog
	err := r.model(ctx).
		Where("rid = ?", rid).
		Order("id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list, nil
}

func (r *repository) Count(ctx context.Context, filter ListFilter) (int64, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	filter = filter.Normalize()
	query := r.applyListFilter(r.model(ctx), filter)
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) List(ctx context.Context, filter ListFilter) ([]*auditmodel.AuditLog, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	filter = filter.Normalize()
	var list []*auditmodel.AuditLog
	query := r.applyListFilter(r.model(ctx), filter).
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

func (r *repository) applyListFilter(query *gorm.DB, filter ListFilter) *gorm.DB {
	if query == nil {
		return nil
	}
	if len(filter.IDs) > 0 {
		query = query.Where("id IN ?", filter.IDs)
	}
	if filter.RID != "" {
		query = query.Where("rid = ?", filter.RID)
	}
	if filter.TraceID != "" {
		query = query.Where("trace_id = ?", filter.TraceID)
	}
	if filter.UID != "" {
		query = query.Where("uid = ?", filter.UID)
	}
	if filter.TenantID != "" {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		query = query.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		query = query.Where("resource_id = ?", filter.ResourceID)
	}
	if filter.Success != nil {
		query = query.Where("success = ?", *filter.Success)
	}
	if filter.HTTPStatus != nil {
		query = query.Where("http_status = ?", *filter.HTTPStatus)
	}
	if filter.ErrCode != "" {
		query = query.Where("err_code = ?", filter.ErrCode)
	}
	if filter.Method != "" {
		query = query.Where("method = ?", filter.Method)
	}
	if filter.Path != "" {
		query = query.Where("path = ?", filter.Path)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where(
			"rid LIKE ? OR trace_id LIKE ? OR uid LIKE ? OR tenant_id LIKE ? OR role LIKE ? OR action LIKE ? OR resource_type LIKE ? OR resource_id LIKE ? OR ip LIKE ? OR ua LIKE ? OR device LIKE ? OR refer LIKE ? OR err_code LIKE ? OR method LIKE ? OR path LIKE ? OR query_summary LIKE ? OR body_summary LIKE ? OR resp_summary LIKE ? OR risk_action LIKE ? OR risk_reasons LIKE ?",
			like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like, like,
		)
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

func cleanLogs(logs []*auditmodel.AuditLog) []*auditmodel.AuditLog {
	if len(logs) == 0 {
		return nil
	}
	out := make([]*auditmodel.AuditLog, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			continue
		}
		out = append(out, log)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
