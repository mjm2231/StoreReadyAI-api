package service

import (
	"context"
	"fmt"
	"storeready_ai/internal/admin/modules/stats/dto"
	"storeready_ai/internal/contracts/stats"
	"time"
)

type SubscriptionService interface {
	CountSubscriptions(ctx context.Context, filter stats.SubscriptionCountFilter) (int64, error)
	GetSubscriptionCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]stats.TrendPoint, error)
	GetSubscriptionCycleStats(ctx context.Context, tenantID uint64) ([]stats.SubscriptionCycleStats, error)
	GetSubscriptionCurrencyStats(ctx context.Context, tenantID uint64, limit int) ([]stats.SubscriptionCurrencyStats, error)
	GetTopServiceNames(ctx context.Context, tenantID uint64, limit int) ([]stats.ServiceNameStats, error)

	// 下面这些统计先按“事件聚合仓储”方式约定，后续由 repo 基于 client_events/user_usage_stats_daily 实现。
	CountSubscriptionListViewUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
	CountSubscriptionEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
	CountSubscriptionSyncEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
}

type ReminderService interface {
	CountReminders(ctx context.Context, filter stats.ReminderCountFilter) (int64, error)
	GetReminderSentTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]stats.TrendPoint, error)
	GetReminderStatusStats(ctx context.Context, tenantID uint64) ([]stats.ReminderStatusStats, error)

	CountReminderEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
	CountReminderSumField(ctx context.Context, tenantID uint64, eventName, fieldName, startDate, endDate string) (int64, error)
}

type UserService interface {
	CountUsers(ctx context.Context, filter stats.UserFilter) (int64, error)
	CountNewUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
	GetUserCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]stats.TrendPoint, error)

	CountActiveUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
	CountReturnUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
	CountLoginEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
	CountLoginMethodEvent(ctx context.Context, tenantID uint64, method, eventName, startDate, endDate string) (int64, error)
	CountVipEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
}

// Service 聚合后台统计所需的用户、订阅、提醒三个维度服务，统一对 handler 输出 DTO。
type Service struct {
	Subscription SubscriptionService
	Reminder     ReminderService
	User         UserService
}

// New 创建后台统计服务，按模块注入依赖，方便后续扩展更多运营统计口径。
func New(
	subscriptionRepo SubscriptionService,
	reminderRepo ReminderService,
	userRepo UserService,
) *Service {
	return &Service{
		Subscription: subscriptionRepo,
		Reminder:     reminderRepo,
		User:         userRepo,
	}
}

// GetOverview 返回后台首页总览卡片统计，优先提供最核心的规模型指标。
func (s *Service) GetOverview(ctx context.Context, tenantID uint64, now time.Time) (*dto.OverviewStats, error) {
	// 兜底当前时间，避免外部传入零值导致 24h 统计口径异常。
	if now.IsZero() {
		now = time.Now()
	}

	// 运营后台统一采用“最近 24 小时”作为增量观察窗口。
	last24h := uint64(now.Add(-24 * time.Hour).Unix())
	overviewStats := &dto.OverviewStats{}

	// 1) 用户总数
	count, err := s.User.CountUsers(
		ctx,
		stats.UserFilter{
			TenantID: tenantID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count total users failed: %w", err)
	}
	overviewStats.TotalUsers = count

	// 2) 最近 24 小时新增用户数
	count, err = s.User.CountUsers(
		ctx,
		stats.UserFilter{
			TenantID:     tenantID,
			CreatedAfter: &last24h,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count users created in 24h failed: %w", err)
	}
	overviewStats.UsersCreated24h = count

	// 3) 订阅总数（包含 active / archived，便于运营看整体规模）
	count, err = s.Subscription.CountSubscriptions(
		ctx,
		stats.SubscriptionCountFilter{
			TenantID: tenantID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count total subscriptions failed: %w", err)
	}
	overviewStats.TotalSubscriptions = count

	// 4) 活跃订阅数
	count, err = s.Subscription.CountSubscriptions(
		ctx,
		stats.SubscriptionCountFilter{
			TenantID: tenantID,
			Status:   ptr(uint8(1)), // 1 = active
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count active subscriptions failed: %w", err)
	}
	overviewStats.ActiveSubscriptions = count

	// 5) 已归档订阅数
	count, err = s.Subscription.CountSubscriptions(
		ctx,
		stats.SubscriptionCountFilter{
			TenantID: tenantID,
			Status:   ptr(uint8(2)), // 2 = archived
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count archived subscriptions failed: %w", err)
	}
	overviewStats.ArchivedSubscriptions = count

	// 6) 最近 24 小时有变更的订阅数（便于看近期活跃度）
	count, err = s.Subscription.CountSubscriptions(
		ctx,
		stats.SubscriptionCountFilter{
			TenantID:     tenantID,
			UpdatedAfter: &last24h,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count subscriptions updated in 24h failed: %w", err)
	}
	overviewStats.SubscriptionsUpdated24h = count

	// 7) 提醒总数（所有状态）
	count, err = s.Reminder.CountReminders(
		ctx,
		stats.ReminderCountFilter{
			TenantID: tenantID,
			Status:   nil,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count total reminders failed: %w", err)
	}
	overviewStats.TotalReminders = count

	// 8) 待发送提醒数
	count, err = s.Reminder.CountReminders(
		ctx,
		stats.ReminderCountFilter{
			TenantID: tenantID,
			Status:   ptr(uint8(1)), // 1 = pending
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count pending reminders failed: %w", err)
	}
	overviewStats.PendingReminders = count

	// 9) 已发送提醒数
	count, err = s.Reminder.CountReminders(
		ctx,
		stats.ReminderCountFilter{
			TenantID: tenantID,
			Status:   ptr(uint8(2)), // 2 = sent
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count sent reminders failed: %w", err)
	}
	overviewStats.SentReminders = count

	// 10) 已取消提醒数
	count, err = s.Reminder.CountReminders(
		ctx,
		stats.ReminderCountFilter{
			TenantID: tenantID,
			Status:   ptr(uint8(3)), // 3 = canceled
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count canceled reminders failed: %w", err)
	}
	overviewStats.CanceledReminders = count

	// 11) 已跳过提醒数
	count, err = s.Reminder.CountReminders(
		ctx,
		stats.ReminderCountFilter{
			TenantID: tenantID,
			Status:   ptr(uint8(4)), // 4 = skipped
		},
	)
	if err != nil {
		return nil, fmt.Errorf("count skipped reminders failed: %w", err)
	}
	overviewStats.SkippedReminders = count

	return overviewStats, nil
}

// GetUserCoreStats 返回用户规模、新增、活跃等核心指标。
func (s *Service) GetUserCoreStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.UserCoreStats, error) {
	result := &dto.UserCoreStats{}

	// 1) 用户总量：看全量规模。
	totalUsers, err := s.User.CountUsers(ctx, stats.UserFilter{TenantID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("count total users failed: %w", err)
	}
	result.TotalUsers = totalUsers

	// 2) 新增用户：严格按当前查询窗口统计，不再误用总用户数。
	newUsers, err := s.User.CountNewUsers(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count new users failed: %w", err)
	}
	result.NewUsers = newUsers

	// 3) 活跃用户：DAU / WAU / MAU 统一以 endDate 为观察截止日回推。
	dauStart, dauEnd, err := resolveRecentWindow(endDate, 1)
	if err != nil {
		return nil, fmt.Errorf("resolve dau window failed: %w", err)
	}
	wauStart, wauEnd, err := resolveRecentWindow(endDate, 7)
	if err != nil {
		return nil, fmt.Errorf("resolve wau window failed: %w", err)
	}
	mauStart, mauEnd, err := resolveRecentWindow(endDate, 30)
	if err != nil {
		return nil, fmt.Errorf("resolve mau window failed: %w", err)
	}

	dau, err := s.User.CountActiveUsers(ctx, tenantID, dauStart, dauEnd)
	if err != nil {
		return nil, fmt.Errorf("count dau failed: %w", err)
	}
	result.DAU = dau

	wau, err := s.User.CountActiveUsers(ctx, tenantID, wauStart, wauEnd)
	if err != nil {
		return nil, fmt.Errorf("count wau failed: %w", err)
	}
	result.WAU = wau

	mau, err := s.User.CountActiveUsers(ctx, tenantID, mauStart, mauEnd)
	if err != nil {
		return nil, fmt.Errorf("count mau failed: %w", err)
	}
	result.MAU = mau

	// 4) 回流用户：仍按当前查询窗口统计，后续可在 repo 层细化“回流”定义。
	returnUsers, err := s.User.CountReturnUsers(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count return users failed: %w", err)
	}
	result.ReturnUserCount = returnUsers

	// 5) 活跃率：当前先按 DAU / TotalUsers 口径给后台首页使用。
	if totalUsers > 0 && result.DAU > 0 {
		result.ActiveUserRate = float64(result.DAU) / float64(totalUsers)
	}

	return result, nil
}

// GetLoginStats 返回登录链路核心统计，覆盖开始、成功、失败与第三方登录情况。
func (s *Service) GetLoginStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.LoginStats, error) {
	result := &dto.LoginStats{}

	count, err := s.User.CountLoginEvent(ctx, tenantID, "login_page_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count login page view failed: %w", err)
	}
	result.LoginStartCount = count

	count, err = s.User.CountLoginEvent(ctx, tenantID, "login_submit_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count login success failed: %w", err)
	}
	result.LoginSuccessCount = count

	count, err = s.User.CountLoginEvent(ctx, tenantID, "login_submit_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count login failed failed: %w", err)
	}
	result.LoginFailedCount = count

	count, err = s.User.CountLoginEvent(ctx, tenantID, "third_party_login_started", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count third party login started failed: %w", err)
	}
	result.ThirdPartyStartCount = count

	count, err = s.User.CountLoginEvent(ctx, tenantID, "third_party_login_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count third party login success failed: %w", err)
	}
	result.ThirdPartySuccessCount = count

	count, err = s.User.CountLoginEvent(ctx, tenantID, "third_party_login_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count third party login fail failed: %w", err)
	}
	result.ThirdPartyFailedCount = count

	count, err = s.User.CountLoginEvent(ctx, tenantID, "third_party_login_exception", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count third party login exception failed: %w", err)
	}
	result.ThirdPartyExceptionCount = count

	if result.LoginStartCount > 0 && result.LoginSuccessCount > 0 {
		result.LoginSuccessRate = float64(result.LoginSuccessCount) / float64(result.LoginStartCount)
	}

	return result, nil
}

// GetSubscriptionCoreStats 返回订阅模块的核心运营统计。
func (s *Service) GetSubscriptionCoreStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.SubscriptionCoreStats, error) {
	result := &dto.SubscriptionCoreStats{}

	count, err := s.Subscription.CountSubscriptionListViewUsers(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription list view users failed: %w", err)
	}
	result.ListViewUV = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_create_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription create view failed: %w", err)
	}
	result.CreateViewCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_edit_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription edit view failed: %w", err)
	}
	result.EditViewCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_add_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription add click failed: %w", err)
	}
	result.AddClickCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_create_submit_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription create success failed: %w", err)
	}
	result.CreateSuccessCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_edit_submit_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription edit success failed: %w", err)
	}
	result.EditSuccessCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_submit_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription submit fail failed: %w", err)
	}
	result.SubmitFailedCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_archive_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription archive click failed: %w", err)
	}
	result.ArchiveCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_restore_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription restore click failed: %w", err)
	}
	result.RestoreCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_detail_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription detail view failed: %w", err)
	}
	result.DetailViewCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_quick_update_next_billing_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription quick update click failed: %w", err)
	}
	result.QuickUpdateCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_rollover_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription rollover click failed: %w", err)
	}
	result.RolloverCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_cancel_archive_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription cancel archive click failed: %w", err)
	}
	result.CancelArchiveCount = count

	return result, nil
}

// GetSubscriptionFormStats 返回订阅创建/编辑表单转化统计。
func (s *Service) GetSubscriptionFormStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.SubscriptionFormStats, error) {
	result := &dto.SubscriptionFormStats{}

	count, err := s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_create_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription create view failed: %w", err)
	}
	result.CreateViewCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_edit_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription edit view failed: %w", err)
	}
	result.EditViewCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_save_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription save click failed: %w", err)
	}
	result.SaveClickCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_submit_started", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription submit started failed: %w", err)
	}
	result.SubmitStartedCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_submit_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription submit success failed: %w", err)
	}
	result.SubmitSuccessCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_submit_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription submit fail failed: %w", err)
	}
	result.SubmitFailedCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "subscription_submit_rejected", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription submit rejected failed: %w", err)
	}
	result.SubmitRejectedCount = count

	if result.CreateViewCount > 0 && result.SubmitSuccessCount > 0 {
		result.CreateConversionRate = float64(result.SubmitSuccessCount) / float64(result.CreateViewCount)
	}
	if result.EditViewCount > 0 && result.SubmitSuccessCount > 0 {
		result.EditConversionRate = float64(result.SubmitSuccessCount) / float64(result.EditViewCount)
	}

	return result, nil
}

// GetSubscriptionSyncStats 返回订阅同步链路健康情况。
func (s *Service) GetSubscriptionSyncStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.SubscriptionSyncStats, error) {
	result := &dto.SubscriptionSyncStats{}

	count, err := s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_fetch_started", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription fetch started failed: %w", err)
	}
	result.FetchStartedCount = count

	count, err = s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_fetch_remote_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription fetch remote success failed: %w", err)
	}
	result.FetchRemoteSuccessCount = count

	count, err = s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_fetch_remote_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription fetch remote fail failed: %w", err)
	}
	result.FetchRemoteFailedCount = count

	count, err = s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_fetch_remote_skipped", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription fetch remote skipped failed: %w", err)
	}
	result.FetchRemoteSkippedCount = count

	count, err = s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_archive_remote_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription archive remote success failed: %w", err)
	}
	result.ArchiveRemoteSuccessCount = count

	count, err = s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_archive_remote_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription archive remote fail failed: %w", err)
	}
	result.ArchiveRemoteFailedCount = count

	count, err = s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_restore_remote_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription restore remote success failed: %w", err)
	}
	result.RestoreRemoteSuccessCount = count

	count, err = s.Subscription.CountSubscriptionSyncEvent(ctx, tenantID, "subscription_restore_remote_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription restore remote fail failed: %w", err)
	}
	result.RestoreRemoteFailedCount = count

	return result, nil
}

// GetReminderCoreStats 返回提醒模块核心统计。
func (s *Service) GetReminderCoreStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.ReminderCoreStats, error) {
	result := &dto.ReminderCoreStats{}

	count, err := s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_page_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder page view failed: %w", err)
	}
	result.ReminderPageViewCount = count

	count, err = s.Reminder.CountReminderSumField(ctx, tenantID, "reminder_page_view", "today_count", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder today_count sum failed: %w", err)
	}
	result.TodayCountSum = count

	count, err = s.Reminder.CountReminderSumField(ctx, tenantID, "reminder_page_view", "next_7_count", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder next_7_count sum failed: %w", err)
	}
	result.Next7CountSum = count

	count, err = s.Reminder.CountReminderSumField(ctx, tenantID, "reminder_page_view", "next_30_count", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder next_30_count sum failed: %w", err)
	}
	result.Next30CountSum = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_settings_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder settings view failed: %w", err)
	}
	result.SettingsViewCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_refresh_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder refresh click failed: %w", err)
	}
	result.RefreshCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_retry_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder retry click failed: %w", err)
	}
	result.RetryCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_initialize_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder initialize success failed: %w", err)
	}
	result.InitSuccessCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_initialize_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder initialize fail failed: %w", err)
	}
	result.InitFailedCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_load_remote_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder load remote success failed: %w", err)
	}
	result.LoadRemoteSuccessCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_load_remote_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder load remote fail failed: %w", err)
	}
	result.LoadRemoteFailedCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_load_remote_skipped", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder load remote skipped failed: %w", err)
	}
	result.LoadRemoteSkippedCount = count

	return result, nil
}

// GetReminderSettingsStats 返回提醒设置修改与保存统计。
func (s *Service) GetReminderSettingsStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.ReminderSettingsStats, error) {
	result := &dto.ReminderSettingsStats{}

	count, err := s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_settings_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder settings view failed: %w", err)
	}
	result.SettingsViewCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_notification_enable", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder notification enable failed: %w", err)
	}
	result.NotificationEnableCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_notification_disable", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder notification disable failed: %w", err)
	}
	result.NotificationDisableCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_before_days_changed", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder before days changed failed: %w", err)
	}
	result.BeforeDaysChangedCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_on_day_enable", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder on day enable failed: %w", err)
	}
	result.OnDayEnableCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_on_day_disable", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder on day disable failed: %w", err)
	}
	result.OnDayDisableCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_settings_update_started", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder settings update started failed: %w", err)
	}
	result.UpdateStartedCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_settings_local_applied", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder settings local applied failed: %w", err)
	}
	result.LocalAppliedCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_settings_remote_saved", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder settings remote saved failed: %w", err)
	}
	result.RemoteSavedCount = count

	count, err = s.Reminder.CountReminderEvent(ctx, tenantID, "reminder_settings_remote_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count reminder settings remote fail failed: %w", err)
	}
	result.RemoteFailedCount = count

	return result, nil
}

// GetOverviewPageStats 返回前台总览页使用情况，便于后台评估该价值模块使用率。
func (s *Service) GetOverviewPageStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.OverviewPageStats, error) {
	result := &dto.OverviewPageStats{}

	count, err := s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_page_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview page view failed: %w", err)
	}
	result.OverviewViewCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_empty_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview empty view failed: %w", err)
	}
	result.OverviewEmptyViewCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_refresh_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview refresh click failed: %w", err)
	}
	result.RefreshCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_retry_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview retry click failed: %w", err)
	}
	result.RetryCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_local_build_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview local build success failed: %w", err)
	}
	result.LocalBuildSuccessCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_local_build_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview local build fail failed: %w", err)
	}
	result.LocalBuildFailedCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_remote_success", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview remote success failed: %w", err)
	}
	result.RemoteSuccessCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_remote_fail", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview remote fail failed: %w", err)
	}
	result.RemoteFailedCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_remote_skipped", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview remote skipped failed: %w", err)
	}
	result.RemoteSkippedCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_has_top_subscription", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview has top subscription failed: %w", err)
	}
	result.HasTopSubscriptionCount = count

	count, err = s.Subscription.CountSubscriptionEvent(ctx, tenantID, "overview_top_subscription_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count overview top subscription click failed: %w", err)
	}
	result.TopSubscriptionClickCount = count

	return result, nil
}

// GetVipCoreStats 返回免费上限触发与 VIP 引导点击的核心统计。
func (s *Service) GetVipCoreStats(ctx context.Context, tenantID uint64, startDate, endDate string) (*dto.VipCoreStats, error) {
	result := &dto.VipCoreStats{}

	count, err := s.User.CountVipEvent(ctx, tenantID, "vip_paywall_view", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count vip paywall view failed: %w", err)
	}
	result.PaywallViewCount = count

	count, err = s.User.CountVipEvent(ctx, tenantID, "vip_upgrade_click", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count vip upgrade click failed: %w", err)
	}
	result.UpgradeClickCount = count

	count, err = s.User.CountVipEvent(ctx, tenantID, "subscription_free_limit_hit", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription free limit hit failed: %w", err)
	}
	result.FreeLimitHitCount = count

	count, err = s.User.CountVipEvent(ctx, tenantID, "subscription_can_create_more_false", startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("count subscription can_create_more false failed: %w", err)
	}
	result.CanCreateMoreFalseCount = count

	if result.PaywallViewCount > 0 && result.UpgradeClickCount > 0 {
		result.UpgradeClickRate = float64(result.UpgradeClickCount) / float64(result.PaywallViewCount)
	}

	return result, nil
}

// GetSubscriptionCreatedTrend 返回订阅新增趋势，适合后台折线图直接消费。
func (s *Service) GetSubscriptionCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]dto.TrendPoint, error) {
	trendPoint, err := s.Subscription.GetSubscriptionCreatedTrend(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	dtoPoints := make([]dto.TrendPoint, 0, len(trendPoint))
	for _, point := range trendPoint {
		dtoPoints = append(dtoPoints, dto.TrendPoint{
			Date:  point.Date,
			Count: point.Count,
		})
	}
	return dtoPoints, nil
}

// GetReminderSentTrend 返回提醒发送趋势，便于观察提醒系统是否稳定生效。
func (s *Service) GetReminderSentTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]dto.TrendPoint, error) {
	trendPoints, err := s.Reminder.GetReminderSentTrend(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	dtoPoints := make([]dto.TrendPoint, 0, len(trendPoints))
	for _, point := range trendPoints {
		dtoPoints = append(dtoPoints, dto.TrendPoint{
			Date:  point.Date,
			Count: point.Count,
		})
	}
	return dtoPoints, nil
}

// GetReminderStatusStats 返回提醒状态统计。
func (s *Service) GetReminderStatusStats(ctx context.Context, tenantID uint64) ([]dto.ReminderStatusStats, error) {
	reminderStatusStats, err := s.Reminder.GetReminderStatusStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	dtoStats := make([]dto.ReminderStatusStats, 0, len(reminderStatusStats))
	for _, stat := range reminderStatusStats {
		dtoStats = append(dtoStats, dto.ReminderStatusStats{
			Status: stat.Status,
			Count:  stat.Count,
		})
	}
	return dtoStats, nil
}

// GetUserCreatedTrend 返回用户新增趋势，便于观察用户增长情况。
func (s *Service) GetUserCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]dto.TrendPoint, error) {
	userCreatedTrend, err := s.User.GetUserCreatedTrend(ctx, tenantID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	dtoPoints := make([]dto.TrendPoint, 0, len(userCreatedTrend))
	for _, point := range userCreatedTrend {
		dtoPoints = append(dtoPoints, dto.TrendPoint{
			Date:  point.Date,
			Count: point.Count,
		})
	}
	return dtoPoints, nil
}

// GetSubscriptionCycleStats 返回订阅周期分布统计。
func (s *Service) GetSubscriptionCycleStats(ctx context.Context, tenantID uint64) ([]dto.SubscriptionCycleStats, error) {
	cycleStats, err := s.Subscription.GetSubscriptionCycleStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(cycleStats) == 0 {
		return make([]dto.SubscriptionCycleStats, 0), nil
	}

	dtoStats := make([]dto.SubscriptionCycleStats, 0, len(cycleStats))
	for _, stat := range cycleStats {
		dtoStats = append(dtoStats, dto.SubscriptionCycleStats{
			Cycle: stat.Cycle,
			Count: stat.Count,
		})
	}
	return dtoStats, nil
}

// GetSubscriptionCurrencyStats 返回订阅币种分布统计。
func (s *Service) GetSubscriptionCurrencyStats(ctx context.Context, tenantID uint64, limit int) ([]dto.SubscriptionCurrencyStats, error) {
	currencyStats, err := s.Subscription.GetSubscriptionCurrencyStats(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	if len(currencyStats) == 0 {
		return make([]dto.SubscriptionCurrencyStats, 0), nil
	}

	dtoStats := make([]dto.SubscriptionCurrencyStats, 0, len(currencyStats))
	for _, stat := range currencyStats {
		dtoStats = append(dtoStats, dto.SubscriptionCurrencyStats{
			Currency: stat.Currency,
			Count:    stat.Count,
		})
	}
	return dtoStats, nil
}

// GetTopServiceNames 返回热门服务名排行。
func (s *Service) GetTopServiceNames(ctx context.Context, tenantID uint64, limit int) ([]dto.ServiceNameStats, error) {
	serviceNameStats, err := s.Subscription.GetTopServiceNames(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	if len(serviceNameStats) == 0 {
		return make([]dto.ServiceNameStats, 0), nil
	}

	dtoStats := make([]dto.ServiceNameStats, 0, len(serviceNameStats))
	for _, stat := range serviceNameStats {
		dtoStats = append(dtoStats, dto.ServiceNameStats{
			ServiceName: stat.ServiceName,
			Count:       stat.Count,
		})
	}
	return dtoStats, nil
}

func resolveRecentWindow(endDate string, days int) (string, string, error) {
	if days <= 0 {
		days = 1
	}

	end, err := resolveWindowEndDate(endDate)
	if err != nil {
		return "", "", err
	}
	start := end.AddDate(0, 0, -(days - 1))
	return formatDate(start), formatDate(end), nil
}

func resolveWindowEndDate(endDate string) (time.Time, error) {
	if endDate == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	parsed, err := time.ParseInLocation(time.DateOnly, endDate, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid end_date %q: %w", endDate, err)
	}
	return parsed, nil
}

func formatDate(t time.Time) string {
	return t.Format(time.DateOnly)
}

// ptr 用于快速构造可选筛选字段指针，减少临时变量样板代码。
func ptr[T any](v T) *T {
	return &v
}
