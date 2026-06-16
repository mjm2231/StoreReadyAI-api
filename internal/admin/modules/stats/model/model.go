package model

// UserUsageStatsDaily 对应 user_usage_stats_daily 日统计表。
type UserUsageStatsDaily struct {
	ID                   uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TenantID             uint64 `gorm:"column:tenant_id;not null;index:uk_user_date,unique" json:"tenant_id"`
	UserID               uint64 `gorm:"column:user_id;not null;index:uk_user_date,unique" json:"user_id"`
	StatDate             string `gorm:"column:stat_date;type:date;not null;index:uk_user_date,unique" json:"stat_date"`
	SubscriptionCount    int32  `gorm:"column:subscription_count;not null;default:0" json:"subscription_count"`
	ReminderCount        int32  `gorm:"column:reminder_count;not null;default:0" json:"reminder_count"`
	CreatedSubscriptions int32  `gorm:"column:created_subscriptions;not null;default:0" json:"created_subscriptions"`
	TriggeredReminders   int32  `gorm:"column:triggered_reminders;not null;default:0" json:"triggered_reminders"`
	AppOpenCount         int32  `gorm:"column:app_open_count;not null;default:0" json:"app_open_count"`
	ActiveMinutes        int32  `gorm:"column:active_minutes;not null;default:0" json:"active_minutes"`
	IsVIP                int32  `gorm:"column:is_vip;not null;default:0" json:"is_vip"`
	CreatedAt            int64  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            int64  `gorm:"column:updated_at" json:"updated_at"`
}

func (UserUsageStatsDaily) TableName() string {
	return "user_usage_stats_daily"
}
