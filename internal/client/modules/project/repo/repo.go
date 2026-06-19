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
	// GetProjectByID 查询当前租户、当前用户下的项目。
	GetProjectByID(ctx context.Context, tenantID, userID, projectID uint64) (*model.Project, error)
	// GetStoreInfoByProjectID 查询项目上架资料。
	GetStoreInfoByProjectID(ctx context.Context, tenantID, userID, projectID uint64) (*model.ProjectStoreInfo, error)
	// SaveStoreInfo 保存项目上架资料；存在则更新，不存在则创建。
	SaveStoreInfo(ctx context.Context, info *model.ProjectStoreInfo) error
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
	project.Description = strings.TrimSpace(project.Description)
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

// GetProjectByID 查询当前租户、当前用户下的项目。
func (r *gormRepo) GetProjectByID(ctx context.Context, tenantID, userID, projectID uint64) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ? AND deleted_at = 0", tenantID, userID, projectID).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// GetStoreInfoByProjectID 查询项目上架资料。
func (r *gormRepo) GetStoreInfoByProjectID(ctx context.Context, tenantID, userID, projectID uint64) (*model.ProjectStoreInfo, error) {
	var info model.ProjectStoreInfo
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND project_id = ? AND deleted_at = 0", tenantID, userID, projectID).
		First(&info).Error
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// SaveStoreInfo 保存项目上架资料；存在则更新，不存在则创建。
func (r *gormRepo) SaveStoreInfo(ctx context.Context, info *model.ProjectStoreInfo) error {
	if info == nil {
		return gorm.ErrInvalidData
	}

	now := nowSec()
	trimStoreInfo(info)
	if info.Status == "" {
		info.Status = model.ProjectStoreInfoStatusDraft
	}
	info.DeletedAt = 0

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.ProjectStoreInfo
		err := tx.Where("tenant_id = ? AND user_id = ? AND project_id = ? AND deleted_at = 0", info.TenantID, info.UserID, info.ProjectID).
			First(&existing).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		updates := map[string]any{
			"app_name":           info.AppName,
			"subtitle":           info.Subtitle,
			"keywords":           info.Keywords,
			"short_description":  info.ShortDescription,
			"full_description":   info.FullDescription,
			"category":           info.Category,
			"content_rating":     info.ContentRating,
			"privacy_policy_url": info.PrivacyPolicyURL,
			"support_url":        info.SupportURL,
			"marketing_url":      info.MarketingURL,
			"copyright":          info.Copyright,
			"contact_email":      info.ContactEmail,
			"status":             info.Status,
			"updated_at":         now,
		}

		if err == nil {
			info.ID = existing.ID
			info.CreatedAt = existing.CreatedAt
			info.UpdatedAt = now
			return tx.Model(&model.ProjectStoreInfo{}).
				Where("id = ?", existing.ID).
				Updates(updates).Error
		}

		info.CreatedAt = now
		info.UpdatedAt = now
		return tx.Create(info).Error
	})
}

func trimStoreInfo(info *model.ProjectStoreInfo) {
	info.AppName = strings.TrimSpace(info.AppName)
	info.Subtitle = strings.TrimSpace(info.Subtitle)
	info.Keywords = strings.TrimSpace(info.Keywords)
	info.ShortDescription = strings.TrimSpace(info.ShortDescription)
	info.FullDescription = strings.TrimSpace(info.FullDescription)
	info.Category = strings.TrimSpace(info.Category)
	info.ContentRating = strings.TrimSpace(info.ContentRating)
	info.PrivacyPolicyURL = strings.TrimSpace(info.PrivacyPolicyURL)
	info.SupportURL = strings.TrimSpace(info.SupportURL)
	info.MarketingURL = strings.TrimSpace(info.MarketingURL)
	info.Copyright = strings.TrimSpace(info.Copyright)
	info.ContactEmail = strings.TrimSpace(info.ContactEmail)
	info.Status = strings.TrimSpace(info.Status)
}
