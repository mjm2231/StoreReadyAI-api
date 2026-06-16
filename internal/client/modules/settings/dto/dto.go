package dto

// 用户全局设置 DTO（HTTP 请求/响应）。
//
// 设计原则：
// - 与数据库字段保持一一对应（便于前后端对齐）。
// - 默认值由后端统一补齐（GetOrCreate）。
// - 所有接口统一 POST（你当前路由风格）。

// GetSettingsReq 获取当前用户设置请求（一般无需 body 字段，保留扩展位）
type GetSettingsReq struct {
	// 预留：未来可支持指定 user_id/跨设备调试等；MVP 不使用。
}

// UpdateSettingsReq 更新当前用户设置请求
// 说明：
// - 字段均为指针，表示“可选更新”，未传字段不修改。
// - currency 规范：ISO 4217，3 位（如 USD）。
// - timezone 规范：IANA 时区（如 Asia/Shanghai）。
// - default_remind_before_days 范围：0~30。
// - default_remind_on_day / notification_enabled：0/1。
// - default_notify_time 格式：HH:MM:SS（如 09:00:00）。
type UpdateSettingsReq struct {
	DefaultCurrency         *string `json:"default_currency" binding:"omitempty,len=3"`                  // 默认币种
	DefaultRemindBeforeDays *uint16 `json:"default_remind_before_days" binding:"omitempty,gte=0,lte=30"` // 默认提前提醒天数(0~30)
	DefaultRemindOnDay      *uint8  `json:"default_remind_on_day" binding:"omitempty,oneof=0 1"`         // 默认到期当天提醒(0/1)
	NotificationEnabled     *uint8  `json:"notification_enabled" binding:"omitempty,oneof=0 1"`          // 通知总开关(0/1)
	DefaultNotifyTime       *string `json:"default_notify_time" binding:"omitempty,len=8"`               // 默认通知时间(HH:MM:SS)
	Timezone                *string `json:"timezone" binding:"omitempty,max=64"`                         // 时区(IANA)
}

// SettingsResp 用户设置返回
// 说明：
// - 与 user_settings 表核心字段对齐，供设置页直接展示与回填。
type SettingsResp struct {
	ID       uint64 `json:"id"`
	TenantID uint64 `json:"tenant_id"`
	UserID   uint64 `json:"user_id"`

	DefaultCurrency         string `json:"default_currency"`
	DefaultRemindBeforeDays uint16 `json:"default_remind_before_days"`
	DefaultRemindOnDay      uint8  `json:"default_remind_on_day"`
	NotificationEnabled     uint8  `json:"notification_enabled"`
	DefaultNotifyTime       string `json:"default_notify_time"`
	Timezone                string `json:"timezone"`

	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
	DeletedAt uint64 `json:"deleted_at"`
}

// 常量：0/1 开关（可选，方便前后端对齐）
const (
	SwitchOff uint8 = 0
	SwitchOn  uint8 = 1
)
