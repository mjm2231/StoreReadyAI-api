package stats

// OverviewStats 是后台首页核心概览统计。
//
// 说明：
// 1. 当前聚焦 admin dashboard 最小必要指标；
// 2. 后续可继续扩展 VIP、收入、活跃率等衍生指标；
// 3. 这里仅表达统计结果，不绑定具体图表表现层。
type OverviewStats struct {
	TotalUsers              int64 `json:"total_users"`
	TotalSubscriptions      int64 `json:"total_subscriptions"`
	ActiveSubscriptions     int64 `json:"active_subscriptions"`
	ArchivedSubscriptions   int64 `json:"archived_subscriptions"`
	TotalReminders          int64 `json:"total_reminders"`
	PendingReminders        int64 `json:"pending_reminders"`
	SentReminders           int64 `json:"sent_reminders"`
	CanceledReminders       int64 `json:"canceled_reminders"`
	SkippedReminders        int64 `json:"skipped_reminders"`
	SubscriptionsUpdated24h int64 `json:"subscriptions_updated_24h"`
	UsersCreated24h         int64 `json:"users_created_24h"`
}

// TrendPoint 是统计趋势点。
type TrendPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// SubscriptionCycleStats 是订阅周期分布。
type SubscriptionCycleStats struct {
	Cycle uint8 `json:"cycle"`
	Count int64 `json:"count"`
}

// SubscriptionCurrencyStats 是订阅币种分布。
type SubscriptionCurrencyStats struct {
	Currency string `json:"currency"`
	Count    int64  `json:"count"`
}

// ReminderStatusStats 是提醒状态分布。
type ReminderStatusStats struct {
	Status uint8 `json:"status"`
	Count  int64 `json:"count"`
}

// ServiceNameStats 是服务名称排行。
type ServiceNameStats struct {
	ServiceName string `json:"service_name"`
	Count       int64  `json:"count"`
}

type TrendRow struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}

type UserFilter struct {
	TenantID     uint64
	CreatedAfter *uint64
}

type SubscriptionCountFilter struct {
	TenantID       uint64
	Status         *uint8
	UpdatedAfter   *uint64
	IncludeDeleted bool
}

type ReminderCountFilter struct {
	TenantID uint64
	Status   *uint8
}
