package model

// UserFeedback 用户反馈表。
type UserFeedback struct {
	ID       uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键ID" json:"id"`
	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_feedback_tenant_uid,priority:1;index:idx_feedback_status_priority,priority:1;index:idx_feedback_category_created,priority:1;index:idx_feedback_created,priority:1;index:idx_feedback_deleted,priority:1;comment:租户ID" json:"tenant_id"`
	UID      uint64 `gorm:"column:uid;not null;default:0;index:idx_feedback_tenant_uid,priority:2;comment:业务用户UID，未登录可为0" json:"uid"`

	Category uint8  `gorm:"column:category;type:tinyint unsigned;not null;default:1;index:idx_feedback_category_created,priority:2;comment:反馈分类：1普通反馈 2问题报错 3功能建议 4支付订阅 5账号登录 6其他" json:"category"`
	Title    string `gorm:"column:title;type:varchar(128);not null;default:'';comment:反馈标题，客户端可选填" json:"title"`
	Content  string `gorm:"column:content;type:text;not null;comment:反馈内容" json:"content"`
	Contact  string `gorm:"column:contact;type:varchar(128);not null;default:'';comment:用户联系方式：邮箱/手机号/其他" json:"contact"`

	Status   uint8 `gorm:"column:status;type:tinyint unsigned;not null;default:1;index:idx_feedback_status_priority,priority:2;comment:处理状态：1待处理 2处理中 3已处理 4已关闭" json:"status"`
	Priority uint8 `gorm:"column:priority;type:tinyint unsigned;not null;default:2;index:idx_feedback_status_priority,priority:3;comment:优先级：1低 2普通 3高 4紧急" json:"priority"`

	ReplyContent string `gorm:"column:reply_content;type:text;comment:后台回复内容" json:"reply_content"`
	HandledBy    uint64 `gorm:"column:handled_by;not null;default:0;comment:处理人管理员ID" json:"handled_by"`
	HandledAt    uint64 `gorm:"column:handled_at;not null;default:0;comment:处理时间，Unix秒" json:"handled_at"`

	AppVersion  string `gorm:"column:app_version;type:varchar(32);not null;default:'';comment:App版本号" json:"app_version"`
	BuildNumber string `gorm:"column:build_number;type:varchar(32);not null;default:'';comment:构建号" json:"build_number"`
	Platform    string `gorm:"column:platform;type:varchar(16);not null;default:'';comment:平台：ios/android/web" json:"platform"`
	DeviceModel string `gorm:"column:device_model;type:varchar(128);not null;default:'';comment:设备型号" json:"device_model"`
	OSVersion   string `gorm:"column:os_version;type:varchar(64);not null;default:'';comment:系统版本" json:"os_version"`
	Locale      string `gorm:"column:locale;type:varchar(32);not null;default:'';comment:客户端语言，如 zh-CN/en-US" json:"locale"`

	Extra string `gorm:"column:extra;type:json;comment:扩展信息JSON，如页面路径、截图URL、错误日志ID等" json:"extra"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0;index:idx_feedback_category_created,priority:3;index:idx_feedback_created,priority:2;comment:创建时间，Unix秒" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0;comment:更新时间，Unix秒" json:"updated_at"`
	DeletedAt uint64 `gorm:"column:deleted_at;not null;default:0;index:idx_feedback_deleted,priority:2;comment:软删除时间，0表示未删除" json:"deleted_at"`
}

// TableName 指定表名。
func (UserFeedback) TableName() string {
	return "user_feedbacks"
}

const (
	FeedbackCategoryGeneral    uint8 = 1 // 普通反馈
	FeedbackCategoryBug        uint8 = 2 // 问题报错
	FeedbackCategorySuggestion uint8 = 3 // 功能建议
	FeedbackCategoryBilling    uint8 = 4 // 支付订阅
	FeedbackCategoryAccount    uint8 = 5 // 账号登录
	FeedbackCategoryOther      uint8 = 6 // 其他
)

const (
	FeedbackStatusPending    uint8 = 1 // 待处理
	FeedbackStatusProcessing uint8 = 2 // 处理中
	FeedbackStatusResolved   uint8 = 3 // 已处理
	FeedbackStatusClosed     uint8 = 4 // 已关闭
)

const (
	FeedbackPriorityLow    uint8 = 1 // 低
	FeedbackPriorityNormal uint8 = 2 // 普通
	FeedbackPriorityHigh   uint8 = 3 // 高
	FeedbackPriorityUrgent uint8 = 4 // 紧急
)

// IsValidFeedbackCategory 判断反馈分类是否合法。
func IsValidFeedbackCategory(category uint8) bool {
	switch category {
	case FeedbackCategoryGeneral,
		FeedbackCategoryBug,
		FeedbackCategorySuggestion,
		FeedbackCategoryBilling,
		FeedbackCategoryAccount,
		FeedbackCategoryOther:
		return true
	default:
		return false
	}
}

// NormalizeFeedbackCategory 归一化反馈分类，非法值统一归为“其他”。
func NormalizeFeedbackCategory(category uint8) uint8 {
	if IsValidFeedbackCategory(category) {
		return category
	}
	return FeedbackCategoryOther
}
