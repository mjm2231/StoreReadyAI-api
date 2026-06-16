package repo

import (
	"context"
	"errors"
	"storeready_ai/internal/admin/modules/stats/dto"
	"storeready_ai/internal/admin/modules/stats/model"

	"gorm.io/gorm"
)

// Repository 是后台统计仓储接口。
//
// 约定：
// 1. admin 模块统一采用“接口 + 实现”结构；
// 2. service 依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 当前 repo 只负责聚合查询，不承担业务状态转换逻辑。
type Repository interface {
	// DB 返回当前仓储使用的底层数据库连接。
	DB() *gorm.DB
	// WithDB 基于指定数据库连接返回新的仓储实例，便于事务内复用。
	WithDB(db *gorm.DB) Repository
	//创建stats统计数据
	Create(ctx context.Context, req model.UserUsageStatsDaily) error
	//查询统计数据
	List(ctx context.Context, tenantID uint64, filter dto.ListFilter) ([]*model.UserUsageStatsDaily, error)
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

func (r *repository) Create(ctx context.Context, req model.UserUsageStatsDaily) error {
	return r.db.Create(req).Error
}

func (r *repository) List(ctx context.Context, tenantID uint64, filter dto.ListFilter) ([]*model.UserUsageStatsDaily, error) {
	if tenantID == 0 {
		return nil, errors.New("tenantID 非法")
	}
	filter = filter.Normalize()
	var list []*model.UserUsageStatsDaily
	err := r.db.Model(ctx).Where("tenant_id =?", tenantID).Find(list).Error
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}

	return list, nil
}
