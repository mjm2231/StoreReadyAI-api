package repo

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"storeready_ai/internal/admin/modules/feedback/dto"
	"storeready_ai/internal/contracts/feedback/model"
)

// Repository 用户反馈仓储接口。
type Repository interface {
	GetByID(ctx context.Context, tenantID uint64, id uint64) (*model.UserFeedback, error)
	List(ctx context.Context, tenantID uint64, req dto.FeedbackListReq) ([]model.UserFeedback, int64, error)
	UpdateFields(ctx context.Context, tenantID uint64, id uint64, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, tenantID uint64, id uint64, status uint8, priority uint8, handledBy uint64) error
	Reply(ctx context.Context, tenantID uint64, id uint64, replyContent string, status uint8, handledBy uint64) error
	SoftDelete(ctx context.Context, tenantID uint64, id uint64) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建用户反馈仓储。
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// Create 创建用户反馈。
func (r *repository) Create(ctx context.Context, feedback *model.UserFeedback) error {
	if feedback == nil {
		return nil
	}

	now := uint64(time.Now().Unix())
	if feedback.CreatedAt == 0 {
		feedback.CreatedAt = now
	}
	if feedback.UpdatedAt == 0 {
		feedback.UpdatedAt = now
	}

	return r.db.WithContext(ctx).Create(feedback).Error
}

// GetByID 根据 ID 获取反馈详情。
func (r *repository) GetByID(ctx context.Context, tenantID uint64, id uint64) (*model.UserFeedback, error) {
	var feedback model.UserFeedback
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		First(&feedback).Error
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

// List 分页查询用户反馈。
func (r *repository) List(ctx context.Context, tenantID uint64, req dto.FeedbackListReq) ([]model.UserFeedback, int64, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := r.db.WithContext(ctx).
		Model(&model.UserFeedback{}).
		Where("tenant_id = ? AND deleted_at = 0", tenantID)

	if req.UID > 0 {
		query = query.Where("uid = ?", req.UID)
	}
	if req.Category > 0 {
		query = query.Where("category = ?", req.Category)
	}
	if req.Status > 0 {
		query = query.Where("status = ?", req.Status)
	}
	if req.Priority > 0 {
		query = query.Where("priority = ?", req.Priority)
	}
	if req.StartAt > 0 {
		query = query.Where("created_at >= ?", req.StartAt)
	}
	if req.EndAt > 0 {
		query = query.Where("created_at <= ?", req.EndAt)
	}

	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR contact LIKE ?", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []model.UserFeedback
	offset := (page - 1) * pageSize
	err := query.
		Order("priority DESC").
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

// UpdateFields 更新指定字段。
func (r *repository) UpdateFields(ctx context.Context, tenantID uint64, id uint64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	fields["updated_at"] = uint64(time.Now().Unix())

	return r.db.WithContext(ctx).
		Model(&model.UserFeedback{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(fields).Error
}

// UpdateStatus 更新处理状态和优先级。
func (r *repository) UpdateStatus(ctx context.Context, tenantID uint64, id uint64, status uint8, priority uint8, handledBy uint64) error {
	now := uint64(time.Now().Unix())
	fields := map[string]interface{}{
		"status":     status,
		"handled_by": handledBy,
		"handled_at": now,
		"updated_at": now,
	}
	if priority > 0 {
		fields["priority"] = priority
	}

	return r.db.WithContext(ctx).
		Model(&model.UserFeedback{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(fields).Error
}

// Reply 回复反馈。
func (r *repository) Reply(ctx context.Context, tenantID uint64, id uint64, replyContent string, status uint8, handledBy uint64) error {
	now := uint64(time.Now().Unix())
	fields := map[string]interface{}{
		"reply_content": replyContent,
		"handled_by":    handledBy,
		"handled_at":    now,
		"updated_at":    now,
	}
	if status > 0 {
		fields["status"] = status
	}

	return r.db.WithContext(ctx).
		Model(&model.UserFeedback{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(fields).Error
}

// SoftDelete 软删除反馈。
func (r *repository) SoftDelete(ctx context.Context, tenantID uint64, id uint64) error {
	now := uint64(time.Now().Unix())
	return r.db.WithContext(ctx).
		Model(&model.UserFeedback{}).
		Where("tenant_id = ? AND id = ? AND deleted_at = 0", tenantID, id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		}).Error
}
