package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"storeready_ai/internal/client/modules/project/model"
)

// ProjectRepo 项目领域的持久化接口（Repo层）。
// 约定：时间字段均为 Unix 秒。
type ProjectRepo interface {
	// Create 创建项目。
	Create(ctx context.Context, project *model.Project) error
	// ListByUser 查询当前租户、当前用户的项目列表。
	ListByUser(ctx context.Context, tenantID, userID uint64, page, pageSize int) ([]*model.Project, int64, error)
}

// gormRepo ProjectRepo 的 GORM 实现。
type gormRepo struct {
	db *gorm.DB
}

// New 创建 Repo。
func New(db *gorm.DB) ProjectRepo {
	return &gormRepo{db: db}
}

// nowSec 当前时间（Unix 秒）。
func nowSec() uint64 { return uint64(time.Now().Unix()) }

// Create 创建项目。
func (r *gormRepo) Create(ctx context.Context, project *model.Project) error {
	if project == nil {
		return gorm.ErrInvalidData
	}

	now := nowSec()
	project.Name = strings.TrimSpace(project.Name)
	project.Platform = strings.TrimSpace(project.Platform)
	project.Status = strings.TrimSpace(project.Status)
	if project.Status == "" {
		project.Status = model.ProjectStatusDraft
	}
	if project.CreatedAt == 0 {
		project.CreatedAt = now
	}
	project.UpdatedAt = now
	project.DeletedAt = 0

	return r.db.WithContext(ctx).Create(project).Error
}

// ListByUser 查询当前租户、当前用户的项目列表。
func (r *gormRepo) ListByUser(ctx context.Context, tenantID, userID uint64, page, pageSize int) ([]*model.Project, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("tenant_id = ? AND user_id = ? AND deleted_at = 0", tenantID, userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	projects := make([]*model.Project, 0)
	offset := (page - 1) * pageSize
	if err := query.
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}
