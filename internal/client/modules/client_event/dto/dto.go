package dto

// ClientEventItemReq 客户端埋点单项请求。
//
// 设计说明：
// 1. tenant_id / uid / created_at 由服务端从上下文或服务层补齐；
// 2. 客户端只上报事件本身与设备/版本/环境信息；
// 3. payload 先使用字符串承接 JSON，避免过早绑定具体结构。
type ClientEventItemReq struct {
	EventID    string `json:"event_id" binding:"required"`
	EventGroup string `json:"event_group" binding:"required"`
	EventName  string `json:"event_name" binding:"required"`

	EventSource string `json:"event_source"`
	Platform    string `json:"platform" binding:"required"`

	AppVersion  string `json:"app_version"`
	BuildNumber string `json:"build_number"`
	PackageName string `json:"package_name"`

	DeviceID    string `json:"device_id"`
	DeviceModel string `json:"device_model"`
	OSVersion   string `json:"os_version"`

	NetworkType    string `json:"network_type"`
	StoreAvailable bool   `json:"store_available"`

	EventCode    string `json:"event_code"`
	EventMessage string `json:"event_message"`
	Payload      string `json:"payload"`
}

// ReportClientEventReq 客户端单条埋点上报请求。
type ReportClientEventReq struct {
	ClientEventItemReq
}

// ReportClientEventsBatchReq 客户端批量埋点上报请求。
type ReportClientEventsBatchReq struct {
	Items []*ClientEventItemReq `json:"items" binding:"required"`
}

// ListClientEventsReq 客户端埋点列表查询请求。
//
// 说明：
// 1. 当前用于后台排查或后续日志页面；
// 2. tenant_id / uid 可由服务端结合登录态补充或覆盖；
// 3. 先提供基础过滤能力，不额外设计复杂排序字段。
type ListClientEventsReq struct {
	UID        uint64 `json:"uid"`
	EventGroup string `json:"event_group"`
	EventName  string `json:"event_name"`
	Platform   string `json:"platform"`
	StartAt    uint64 `json:"start_at"`
	EndAt      uint64 `json:"end_at"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
}

// ClientEventResp 客户端埋点响应。
type ClientEventResp struct {
	ID uint64 `json:"id"`

	TenantID   uint64 `json:"tenant_id"`
	UID        uint64 `json:"uid"`
	EventID    string `json:"event_id"`
	ReceivedAt uint64 `json:"received_at"`

	EventGroup  string `json:"event_group"`
	EventName   string `json:"event_name"`
	EventSource string `json:"event_source"`
	Platform    string `json:"platform"`

	AppVersion  string `json:"app_version"`
	BuildNumber string `json:"build_number"`
	PackageName string `json:"package_name"`

	DeviceID    string `json:"device_id"`
	DeviceModel string `json:"device_model"`
	OSVersion   string `json:"os_version"`

	NetworkType    string `json:"network_type"`
	StoreAvailable bool   `json:"store_available"`

	EventCode    string `json:"event_code"`
	EventMessage string `json:"event_message"`
	Payload      string `json:"payload"`

	CreatedAt uint64 `json:"created_at"`
}

// ListClientEventsResp 客户端埋点列表响应。
type ListClientEventsResp struct {
	Items []*ClientEventResp `json:"items"`
}
