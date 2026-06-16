package dto

import (
	"storeready_ai/internal/common/page"
	"strings"
)

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

// UserCoreStats 是用户核心统计。
//
// 面向后台运营首页/用户增长页，聚合用户规模、新增与活跃口径。
// 当前先保留最常用字段，后续可继续扩展留存、回流、活跃率等指标。
type UserCoreStats struct {
	TotalUsers      int64   `json:"total_users"`
	NewUsers        int64   `json:"new_users"`
	DAU             int64   `json:"dau"`
	WAU             int64   `json:"wau"`
	MAU             int64   `json:"mau"`
	ActiveUserRate  float64 `json:"active_user_rate"`
	ReturnUserCount int64   `json:"return_user_count"`
}

// LoginStats 是登录核心统计。
//
// 说明：
// 1. 用于观察登录链路是否顺畅；
// 2. 既覆盖总登录，也预留第三方登录聚合结果；
// 3. success_rate 建议由 service 按开始/成功口径统一计算。
type LoginStats struct {
	LoginStartCount          int64   `json:"login_start_count"`
	LoginSuccessCount        int64   `json:"login_success_count"`
	LoginFailedCount         int64   `json:"login_failed_count"`
	LoginSuccessRate         float64 `json:"login_success_rate"`
	ThirdPartyStartCount     int64   `json:"third_party_start_count"`
	ThirdPartySuccessCount   int64   `json:"third_party_success_count"`
	ThirdPartyFailedCount    int64   `json:"third_party_failed_count"`
	ThirdPartyExceptionCount int64   `json:"third_party_exception_count"`
}

// LoginMethodStats 是登录方式分布统计。
type LoginMethodStats struct {
	Method       string `json:"method"`
	StartCount   int64  `json:"start_count"`
	SuccessCount int64  `json:"success_count"`
	FailedCount  int64  `json:"failed_count"`
}

// SubscriptionCoreStats 是订阅核心运营统计。
//
// 覆盖订阅列表访问、详情访问、创建编辑、归档恢复等最常用行为指标。
type SubscriptionCoreStats struct {
	ListViewUV         int64 `json:"list_view_uv"`
	CreateViewCount    int64 `json:"create_view_count"`
	EditViewCount      int64 `json:"edit_view_count"`
	AddClickCount      int64 `json:"add_click_count"`
	CreateSuccessCount int64 `json:"create_success_count"`
	EditSuccessCount   int64 `json:"edit_success_count"`
	SubmitFailedCount  int64 `json:"submit_failed_count"`
	ArchiveCount       int64 `json:"archive_count"`
	RestoreCount       int64 `json:"restore_count"`
	DetailViewCount    int64 `json:"detail_view_count"`
	QuickUpdateCount   int64 `json:"quick_update_count"`
	RolloverCount      int64 `json:"rollover_count"`
	CancelArchiveCount int64 `json:"cancel_archive_count"`
}

// SubscriptionFormStats 是订阅表单转化统计。
type SubscriptionFormStats struct {
	CreateViewCount      int64   `json:"create_view_count"`
	EditViewCount        int64   `json:"edit_view_count"`
	SaveClickCount       int64   `json:"save_click_count"`
	SubmitStartedCount   int64   `json:"submit_started_count"`
	SubmitSuccessCount   int64   `json:"submit_success_count"`
	SubmitFailedCount    int64   `json:"submit_failed_count"`
	SubmitRejectedCount  int64   `json:"submit_rejected_count"`
	CreateConversionRate float64 `json:"create_conversion_rate"`
	EditConversionRate   float64 `json:"edit_conversion_rate"`
}

// SubscriptionSearchStats 是订阅搜索使用统计。
type SubscriptionSearchStats struct {
	SearchCount      int64   `json:"search_count"`
	ClearCount       int64   `json:"clear_count"`
	SearchUserCount  int64   `json:"search_user_count"`
	AvgKeywordLength float64 `json:"avg_keyword_length"`
}

// FilterUsageStats 是筛选项使用统计。
type FilterUsageStats struct {
	FilterKey   string `json:"filter_key"`
	FilterValue string `json:"filter_value"`
	Count       int64  `json:"count"`
}

// SubscriptionSyncStats 是订阅同步链路统计。
type SubscriptionSyncStats struct {
	FetchStartedCount         int64 `json:"fetch_started_count"`
	FetchRemoteSuccessCount   int64 `json:"fetch_remote_success_count"`
	FetchRemoteFailedCount    int64 `json:"fetch_remote_failed_count"`
	FetchRemoteSkippedCount   int64 `json:"fetch_remote_skipped_count"`
	ArchiveRemoteSuccessCount int64 `json:"archive_remote_success_count"`
	ArchiveRemoteFailedCount  int64 `json:"archive_remote_failed_count"`
	RestoreRemoteSuccessCount int64 `json:"restore_remote_success_count"`
	RestoreRemoteFailedCount  int64 `json:"restore_remote_failed_count"`
}

// ReminderCoreStats 是提醒模块核心统计。
type ReminderCoreStats struct {
	ReminderPageViewCount  int64 `json:"reminder_page_view_count"`
	TodayCountSum          int64 `json:"today_count_sum"`
	Next7CountSum          int64 `json:"next_7_count_sum"`
	Next30CountSum         int64 `json:"next_30_count_sum"`
	SettingsViewCount      int64 `json:"settings_view_count"`
	RefreshCount           int64 `json:"refresh_count"`
	RetryCount             int64 `json:"retry_count"`
	InitSuccessCount       int64 `json:"init_success_count"`
	InitFailedCount        int64 `json:"init_failed_count"`
	LoadRemoteSuccessCount int64 `json:"load_remote_success_count"`
	LoadRemoteFailedCount  int64 `json:"load_remote_failed_count"`
	LoadRemoteSkippedCount int64 `json:"load_remote_skipped_count"`
}

// ReminderSettingsStats 是提醒设置统计。
type ReminderSettingsStats struct {
	SettingsViewCount        int64 `json:"settings_view_count"`
	NotificationEnableCount  int64 `json:"notification_enable_count"`
	NotificationDisableCount int64 `json:"notification_disable_count"`
	BeforeDaysChangedCount   int64 `json:"before_days_changed_count"`
	OnDayEnableCount         int64 `json:"on_day_enable_count"`
	OnDayDisableCount        int64 `json:"on_day_disable_count"`
	UpdateStartedCount       int64 `json:"update_started_count"`
	LocalAppliedCount        int64 `json:"local_applied_count"`
	RemoteSavedCount         int64 `json:"remote_saved_count"`
	RemoteFailedCount        int64 `json:"remote_failed_count"`
}

// ReminderPermissionStats 是提醒权限相关统计。
type ReminderPermissionStats struct {
	PermissionGrantedCount         int64 `json:"permission_granted_count"`
	PermissionDeniedCount          int64 `json:"permission_denied_count"`
	PermissionBannerShowCount      int64 `json:"permission_banner_show_count"`
	PermissionBannerClickCount     int64 `json:"permission_banner_click_count"`
	OpenSystemSettingsSuccessCount int64 `json:"open_system_settings_success_count"`
	OpenSystemSettingsFailedCount  int64 `json:"open_system_settings_failed_count"`
}

// OverviewPageStats 是总览页使用统计。
type OverviewPageStats struct {
	OverviewViewCount         int64 `json:"overview_view_count"`
	OverviewEmptyViewCount    int64 `json:"overview_empty_view_count"`
	RefreshCount              int64 `json:"refresh_count"`
	RetryCount                int64 `json:"retry_count"`
	LocalBuildSuccessCount    int64 `json:"local_build_success_count"`
	LocalBuildFailedCount     int64 `json:"local_build_failed_count"`
	RemoteSuccessCount        int64 `json:"remote_success_count"`
	RemoteFailedCount         int64 `json:"remote_failed_count"`
	RemoteSkippedCount        int64 `json:"remote_skipped_count"`
	HasTopSubscriptionCount   int64 `json:"has_top_subscription_count"`
	TopSubscriptionClickCount int64 `json:"top_subscription_click_count"`
}

// TopSubscriptionClickStats 是最贵订阅点击统计。
type TopSubscriptionClickStats struct {
	ServiceName string `json:"service_name"`
	Count       int64  `json:"count"`
}

// VipCoreStats 是 VIP 核心转化统计。
type VipCoreStats struct {
	PaywallViewCount        int64   `json:"paywall_view_count"`
	UpgradeClickCount       int64   `json:"upgrade_click_count"`
	UpgradeClickRate        float64 `json:"upgrade_click_rate"`
	FreeLimitHitCount       int64   `json:"free_limit_hit_count"`
	CanCreateMoreFalseCount int64   `json:"can_create_more_false_count"`
}

// VipPaywallStats 是不同来源的 VIP 弹层统计。
type VipPaywallStats struct {
	Source     string  `json:"source"`
	ViewCount  int64   `json:"view_count"`
	ClickCount int64   `json:"click_count"`
	CTR        float64 `json:"ctr"`
}

// FunnelPoint 是转化漏斗节点。
type FunnelPoint struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// EventHealthStats 是事件健康度统计。
type EventHealthStats struct {
	TotalEventCount     int64 `json:"total_event_count"`
	SuccessEventCount   int64 `json:"success_event_count"`
	FailedEventCount    int64 `json:"failed_event_count"`
	ExceptionEventCount int64 `json:"exception_event_count"`
	RemoteFailedCount   int64 `json:"remote_failed_count"`
	LocalFailedCount    int64 `json:"local_failed_count"`
}

// FailureStats 是异常事件排行。
type FailureStats struct {
	EventName string `json:"event_name"`
	EventCode string `json:"event_code"`
	Count     int64  `json:"count"`
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

// ScopeRequest 仅作用域请求。
type ScopeRequest struct {
	TenantID string `json:"tenant_id"`
}

// LimitRequest 带 limit 的统计请求。
type LimitRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Limit    int    `json:"limit"`
}

// OverviewRequest 后台概览请求。
type OverviewRequest struct {
	TenantID string `json:"tenant_id"`
}

// DateRangeRequest 是时间范围统计请求。
//
// 适用于登录、订阅、提醒、VIP 等按时间窗口聚合的统计查询。
type DateRangeRequest struct {
	TenantID  string `json:"tenant_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// TopRequest 是带 limit 的排行统计请求。
type TopRequest struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Limit    int    `json:"limit"`
}

// TrendRequest 趋势统计请求。
type TrendRequest struct {
	TenantID  string `json:"tenant_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (r TrendRequest) Normalize() TrendRequest {
	r.StartDate = strings.TrimSpace(r.StartDate)
	r.EndDate = strings.TrimSpace(r.EndDate)
	return r
}

func (r DateRangeRequest) Normalize() DateRangeRequest {
	r.StartDate = strings.TrimSpace(r.StartDate)
	r.EndDate = strings.TrimSpace(r.EndDate)
	return r
}

func (r TopRequest) Normalize() TopRequest {
	if r.Limit <= 0 {
		r.Limit = 10
	}
	if r.Limit > 100 {
		r.Limit = 100
	}
	return r
}

func (r LimitRequest) Normalize() LimitRequest {
	if r.Limit <= 0 {
		r.Limit = 10
	}
	if r.Limit > 100 {
		r.Limit = 100
	}
	return r
}

// ListFilter 是后台分页列表过滤条件。
type ListFilter struct {
	Page page.PageReq
}

func (l ListFilter) Normalize() ListFilter {
	if l.Page.Page <= 0 {
		l.Page.Page = 1
	}
	if l.Page.PageSize <= 0 {
		l.Page.PageSize = 20
	}
	if l.Page.PageSize > 100 {
		l.Page.PageSize = 100
	}
	return l
}
