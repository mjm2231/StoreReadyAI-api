package model

// ClientEvent 客户端埋点事件。
//
// 设计说明：
// 1. 该模型对应 client_events 表；
// 2. 用于接收客户端上报的 billing / sync / login / app 等事件；
// 3. tenant_id / uid / created_at 等服务端可补齐字段，统一在 service 层处理。
type ClientEvent struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	TenantID uint64 `gorm:"column:tenant_id;not null;default:0" json:"tenant_id"`
	UID      uint64 `gorm:"column:uid;not null;default:0" json:"uid"`

	EventID    string `gorm:"column:event_id;type:varchar(64);not null;default:''" json:"event_id"`
	ReceivedAt uint64 `gorm:"column:received_at;not null;default:0" json:"received_at"`

	EventGroup  string `gorm:"column:event_group;type:varchar(32);not null;default:''" json:"event_group"`
	EventName   string `gorm:"column:event_name;type:varchar(64);not null;default:''" json:"event_name"`
	EventSource string `gorm:"column:event_source;type:varchar(32);not null;default:''" json:"event_source"`
	Platform    string `gorm:"column:platform;type:varchar(16);not null;default:''" json:"platform"`

	AppVersion  string `gorm:"column:app_version;type:varchar(32);not null;default:''" json:"app_version"`
	BuildNumber string `gorm:"column:build_number;type:varchar(32);not null;default:''" json:"build_number"`
	PackageName string `gorm:"column:package_name;type:varchar(128);not null;default:''" json:"package_name"`

	DeviceID    string `gorm:"column:device_id;type:varchar(128);not null;default:''" json:"device_id"`
	DeviceModel string `gorm:"column:device_model;type:varchar(128);not null;default:''" json:"device_model"`
	OSVersion   string `gorm:"column:os_version;type:varchar(64);not null;default:''" json:"os_version"`

	NetworkType    string `gorm:"column:network_type;type:varchar(32);not null;default:''" json:"network_type"`
	StoreAvailable bool   `gorm:"column:store_available;not null;default:0" json:"store_available"`

	EventCode    string `gorm:"column:event_code;type:varchar(64);not null;default:''" json:"event_code"`
	EventMessage string `gorm:"column:event_message;type:varchar(255);not null;default:''" json:"event_message"`
	Payload      string `gorm:"column:payload;type:json" json:"payload"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0" json:"created_at"`
}

// TableName 指定表名。
func (ClientEvent) TableName() string {
	return "client_events"
}
