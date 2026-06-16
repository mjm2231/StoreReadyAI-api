package dto

// 设备（user_devices）相关 DTO（HTTP 请求/响应）。
//
// 说明（MVP）：
// - 用于多设备同步前置：设备登记、心跳、同步时间点更新、设备列表。
// - 所有接口统一使用 POST。
// - user_id 从服务端上下文（common.GetUID）获取，客户端不传。

// RegisterDeviceReq 设备登记请求
// Route: POST /v1/devices/api/register
//
// 约定：device_id 由客户端生成并持久化（同一用户+device_id 唯一）。
// platform: 1=iOS,2=Android,3=Web,9=Unknown
// push_token: 推送 token（APNs/FCM），可选
// device_name/app_version 可选
//
// 注意：last_ip/user_agent 由服务端从请求获取，不建议客户端传。
type RegisterDeviceReq struct {
	DeviceID   string  `json:"device_id" binding:"required,max=128"`
	Platform   uint8   `json:"platform" binding:"required,oneof=1 2 3 9"`
	DeviceName *string `json:"device_name" binding:"omitempty,max=128"`
	AppVersion *string `json:"app_version" binding:"omitempty,max=32"`
	PushToken  *string `json:"push_token" binding:"omitempty,max=256"`
}

// HeartbeatReq 设备心跳/活跃上报
// Route: POST /v1/devices/api/heartbeat
//
// 说明：
// - 用于更新 last_seen_at。
// - 可选 patch push_token/app_version（例如 token 变化）。
// - platform/device_name 可选，允许客户端补全。
type HeartbeatReq struct {
	DeviceID   string  `json:"device_id" binding:"required,max=128"`
	Platform   *uint8  `json:"platform" binding:"omitempty,oneof=1 2 3 9"`
	DeviceName *string `json:"device_name" binding:"omitempty,max=128"`
	AppVersion *string `json:"app_version" binding:"omitempty,max=32"`
	PushToken  *string `json:"push_token" binding:"omitempty,max=256"`
}

// TouchSyncReq 同步时间点上报
// Route: POST /v1/devices/api/touch_sync
//
// 说明：
// - 用于更新 last_sync_at（客户端完成一次 pull/push 后上报）。
// - last_sync_at 不传/为 0：服务端可用当前时间兜底。
type TouchSyncReq struct {
	DeviceID   string `json:"device_id" binding:"required,max=128"`
	LastSyncAt uint64 `json:"last_sync_at"` // Unix 秒
}

// ListDevicesReq 设备列表请求
// Route: POST /v1/devices/api/list
//
// 说明：仅返回 active 设备。
type ListDevicesReq struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// RevokeDeviceReq 撤销设备（踢设备）
// Route: POST /v1/devices/api/revoke
//
// 说明：将该设备 status 标记为 revoked。
type RevokeDeviceReq struct {
	DeviceID string `json:"device_id" binding:"required,max=128"`
}

// DeviceItem 设备信息返回
type DeviceItem struct {
	ID       uint64 `json:"id"`
	TenantID uint64 `json:"tenant_id"`
	UserID   uint64 `json:"user_id"`

	DeviceID   string  `json:"device_id"`
	Platform   uint8   `json:"platform"`
	DeviceName *string `json:"device_name"`
	AppVersion *string `json:"app_version"`
	PushToken  *string `json:"push_token"`

	Status uint8 `json:"status"` // 1=active,2=revoked

	LastSeenAt uint64 `json:"last_seen_at"`
	LastSyncAt uint64 `json:"last_sync_at"`

	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
}

// ListDevicesResp 设备列表返回
type ListDevicesResp struct {
	Total int64         `json:"total"`
	Items []*DeviceItem `json:"items"`
}

// 常量：平台
const (
	PlatformIOS     uint8 = 1
	PlatformAndroid uint8 = 2
	PlatformWeb     uint8 = 3
	PlatformUnknown uint8 = 9
)

// 常量：设备状态
const (
	DeviceStatusActive  uint8 = 1
	DeviceStatusRevoked uint8 = 2
)
