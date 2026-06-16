package repo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"storeready_ai/internal/contracts/feedback/model"
)

// Repository 用户反馈仓储接口。
type Repository interface {
	Create(ctx context.Context, feedback *model.UserFeedback) error
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
