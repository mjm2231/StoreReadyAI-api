package repo

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	reportmodel "storeready_ai/internal/client/modules/client_event/model"
)

// ListClientEventsOption 客户端埋点查询条件。
//
// 说明：
// 1. 当前先提供基础列表查询能力；
// 2. 支持按 tenant/uid、事件分组、事件名、平台、时间范围过滤；
// 3. offset/limit 用于后台排查或后续日志页面分页。
type ListClientEventsOption struct {
	TenantID   uint64
	UID        uint64
	EventGroup string
	EventName  string
	Platform   string
	StartAt    uint64
	EndAt      uint64
	Offset     int
	Limit      int
}

// CreateBatchResult 客户端埋点写入结果。
//
// 说明：
// 1. RequestCount 表示本次传入的原始事件数量；
// 2. ValidCount 表示过滤 nil 并标准化后实际参与入库的事件数量；
// 3. InsertedCount 表示本次实际成功插入的数量；
// 4. DuplicateCount 表示因 event_id 冲突被幂等忽略的数量。
type CreateBatchResult struct {
	RequestCount   int
	ValidCount     int
	InsertedCount  int64
	DuplicateCount int64
}

// ClientEventRepo 客户端埋点仓储。
//
// 职责：
// 1. 提供 client_events 表的基础写入能力；
// 2. 当前支持客户端单条/批量事件上报；
// 3. 后续如需更多聚合查询能力，可继续在这里扩展。
type ClientEventRepo interface {
	// Create 创建一条客户端埋点事件。
	Create(ctx context.Context, event *reportmodel.ClientEvent) (*CreateBatchResult, error)

	// CreateBatch 批量创建客户端埋点事件。
	CreateBatch(ctx context.Context, events []*reportmodel.ClientEvent) (*CreateBatchResult, error)

	// List 查询客户端埋点事件列表。
	List(ctx context.Context, opt ListClientEventsOption) ([]*reportmodel.ClientEvent, error)
}

type clientEventRepo struct {
	db *gorm.DB
}

// NewClientEventRepo 创建客户端埋点仓储实现。
func NewClientEventRepo(db *gorm.DB) ClientEventRepo {
	return &clientEventRepo{db: db}
}

// Create 创建一条客户端埋点事件。
func (r *clientEventRepo) Create(ctx context.Context, event *reportmodel.ClientEvent) (*CreateBatchResult, error) {
	if event == nil {
		return &CreateBatchResult{}, nil
	}

	r.normalizeEvent(event)
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(event)
	if res.Error != nil {
		return nil, res.Error
	}

	inserted := res.RowsAffected
	if inserted < 0 {
		inserted = 0
	}
	duplicate := int64(1) - inserted
	if duplicate < 0 {
		duplicate = 0
	}

	return &CreateBatchResult{
		RequestCount:   1,
		ValidCount:     1,
		InsertedCount:  inserted,
		DuplicateCount: duplicate,
	}, nil
}

// CreateBatch 批量创建客户端埋点事件。
func (r *clientEventRepo) CreateBatch(ctx context.Context, events []*reportmodel.ClientEvent) (*CreateBatchResult, error) {
	result := &CreateBatchResult{RequestCount: len(events)}
	if len(events) == 0 {
		return result, nil
	}

	filtered := make([]*reportmodel.ClientEvent, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		r.normalizeEvent(event)
		filtered = append(filtered, event)
	}
	result.ValidCount = len(filtered)
	if len(filtered) == 0 {
		return result, nil
	}

	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(&filtered)
	if res.Error != nil {
		return nil, res.Error
	}

	inserted := res.RowsAffected
	if inserted < 0 {
		inserted = 0
	}
	duplicate := int64(len(filtered)) - inserted
	if duplicate < 0 {
		duplicate = 0
	}

	result.InsertedCount = inserted
	result.DuplicateCount = duplicate
	return result, nil
}

func (r *clientEventRepo) normalizeEvent(event *reportmodel.ClientEvent) {
	if event == nil {
		return
	}

	event.EventID = strings.TrimSpace(event.EventID)
	if event.EventID == "" {
		event.EventID = uuid.NewString()
	}

	if event.ReceivedAt == 0 {
		event.ReceivedAt = uint64(time.Now().Unix())
	}
}

// List 查询客户端埋点事件列表。
func (r *clientEventRepo) List(ctx context.Context, opt ListClientEventsOption) ([]*reportmodel.ClientEvent, error) {
	q := r.db.WithContext(ctx).Model(&reportmodel.ClientEvent{})

	if opt.TenantID > 0 {
		q = q.Where("tenant_id = ?", opt.TenantID)
	}
	if opt.UID > 0 {
		q = q.Where("uid = ?", opt.UID)
	}
	if opt.EventGroup != "" {
		q = q.Where("event_group = ?", opt.EventGroup)
	}
	if opt.EventName != "" {
		q = q.Where("event_name = ?", opt.EventName)
	}
	if opt.Platform != "" {
		q = q.Where("platform = ?", opt.Platform)
	}
	if opt.StartAt > 0 {
		q = q.Where("created_at >= ?", opt.StartAt)
	}
	if opt.EndAt > 0 {
		q = q.Where("created_at <= ?", opt.EndAt)
	}

	if opt.Offset > 0 {
		q = q.Offset(opt.Offset)
	}
	if opt.Limit > 0 {
		q = q.Limit(opt.Limit)
	}

	q = q.Order("id DESC")

	var items []*reportmodel.ClientEvent
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
