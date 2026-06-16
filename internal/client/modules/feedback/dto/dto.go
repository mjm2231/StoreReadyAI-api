package dto

// CreateFeedbackReq 创建用户反馈请求。
type CreateFeedbackReq struct {
	Category uint8  `json:"category" binding:"omitempty"`              // 反馈分类：1普通反馈 2问题报错 3功能建议 4支付订阅 5账号登录 6其他
	Title    string `json:"title" binding:"omitempty,max=128"`         // 反馈标题，客户端可选填
	Content  string `json:"content" binding:"required,min=1,max=5000"` // 反馈内容
	Contact  string `json:"contact" binding:"omitempty,max=128"`       // 用户联系方式：邮箱/手机号/其他
	Extra    string `json:"extra" binding:"omitempty"`                 // 扩展信息JSON字符串，如页面路径、截图URL、错误日志ID等
}

// FeedbackVO 用户反馈展示对象。
type FeedbackVO struct {
	ID       uint64 `json:"id"`
	TenantID uint64 `json:"tenant_id"`
	UID      uint64 `json:"uid"`

	Category         uint8  `json:"category"`
	CategoryLabel    string `json:"category_label"`
	CategoryLabelKey string `json:"category_label_key"`

	Title   string `json:"title"`
	Content string `json:"content"`
	Contact string `json:"contact"`

	Status         uint8  `json:"status"`
	StatusLabel    string `json:"status_label"`
	StatusLabelKey string `json:"status_label_key"`

	Priority         uint8  `json:"priority"`
	PriorityLabel    string `json:"priority_label"`
	PriorityLabelKey string `json:"priority_label_key"`

	ReplyContent string `json:"reply_content"`
	HandledBy    uint64 `json:"handled_by"`
	HandledAt    uint64 `json:"handled_at"`

	AppVersion  string `json:"app_version"`
	BuildNumber string `json:"build_number"`
	Platform    string `json:"platform"`
	DeviceModel string `json:"device_model"`
	OSVersion   string `json:"os_version"`
	Locale      string `json:"locale"`

	Extra string `json:"extra"`

	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
}
