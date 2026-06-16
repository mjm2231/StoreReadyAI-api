package dto

// FeedbackListReq 后台反馈列表请求。
type FeedbackListReq struct {
	UID      uint64 `json:"uid" binding:"omitempty"`      // 业务用户UID
	Category uint8  `json:"category" binding:"omitempty"` // 反馈分类
	Status   uint8  `json:"status" binding:"omitempty"`   // 处理状态：1待处理 2处理中 3已处理 4已关闭
	Priority uint8  `json:"priority" binding:"omitempty"` // 优先级：1低 2普通 3高 4紧急
	Keyword  string `json:"keyword" binding:"omitempty"`  // 搜索标题/内容/联系方式

	StartAt uint64 `json:"start_at" binding:"omitempty"` // 创建时间起，Unix秒
	EndAt   uint64 `json:"end_at" binding:"omitempty"`   // 创建时间止，Unix秒

	Page     int `json:"page" binding:"omitempty"`      // 页码，从1开始
	PageSize int `json:"page_size" binding:"omitempty"` // 每页数量
}

// UpdateFeedbackStatusReq 更新反馈状态请求。
type UpdateFeedbackStatusReq struct {
	ID       uint64 `json:"id" binding:"required"`               // 反馈ID
	Status   uint8  `json:"status" binding:"required"`           // 处理状态：1待处理 2处理中 3已处理 4已关闭
	Priority uint8  `json:"priority" binding:"omitempty"`        // 优先级：1低 2普通 3高 4紧急，不传则不修改
	Remark   string `json:"remark" binding:"omitempty,max=1000"` // 处理备注，当前可作为扩展字段预留
}

// ReplyFeedbackReq 回复反馈请求。
type ReplyFeedbackReq struct {
	ID           uint64 `json:"id" binding:"required"`                     // 反馈ID
	ReplyContent string `json:"reply_content" binding:"required,max=5000"` // 后台回复内容
	Status       uint8  `json:"status" binding:"omitempty"`                // 回复后状态，不传可由 service 默认置为已处理
}

// FeedbackDetailReq 反馈详情请求。
type FeedbackDetailReq struct {
	ID uint64 `json:"id" binding:"required"` // 反馈ID
}

// DeleteFeedbackReq 删除反馈请求。
type DeleteFeedbackReq struct {
	ID uint64 `json:"id" binding:"required"` // 反馈ID
}

// Feedback 用户反馈展示对象。
type Feedback struct {
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

// FeedbackListResp 后台反馈列表响应。
type FeedbackListResp struct {
	List  []Feedback `json:"list"`
	Total int64      `json:"total"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
}

// FeedbackOption 枚举展示项，用于返回给前端下拉/筛选展示。
type FeedbackOption struct {
	Value    uint8  `json:"value"`
	Label    string `json:"label"`
	LabelKey string `json:"label_key"`
}

// FeedbackOptionsResp 反馈相关枚举选项响应。
type FeedbackOptionsResp struct {
	Categories []FeedbackOption `json:"categories"`
	Statuses   []FeedbackOption `json:"statuses"`
	Priorities []FeedbackOption `json:"priorities"`
}
