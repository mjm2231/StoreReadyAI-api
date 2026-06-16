package model

// UserEntitlement 对应表 `user_entitlements`（用户权益/VIP）。
//
// 约定：
// - 时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
// - tenant_id 在 MVP 固定为 0，但仍保留字段用于未来多租户扩展。
// - entitlement 目前仅使用 "vip"，但保留为字符串，便于未来扩展更多权益。
//
// 字段对齐 SQL：
// - entitlement: VARCHAR(32)
// - product_code: VARCHAR(64)
// - product_id: VARCHAR(128)
// - source: TINYINT (0=manual,1=ios_iap,2=google_play,3=promo)
// - status: TINYINT (1=active,2=expired,3=revoked)
// - started_at/expired_at: BIGINT UNSIGNED (Unix 秒)
// - ref_id: VARCHAR(128) NULL
// - auto_renew: TINYINT (0/1)
//
// 索引/约束：
// - idx_ent_user(tenant_id,user_id,entitlement,status)
// - uk_ent_ref(tenant_id,entitlement,ref_id)（ref_id 非空时幂等；MySQL 允许多个 NULL）
// - idx_ent_expired(tenant_id,user_id,entitlement,expired_at)
//
// 注意：
// - UNIQUE/KEY 建议通过 SQL migration 创建；gorm tag 仅声明索引名。
type UserEntitlement struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement;column:id"` // 主键
	TenantID uint64 `gorm:"not null;default:0;column:tenant_id;index:idx_ent_user,priority:1;index:idx_ent_expired,priority:1;uniqueIndex:uk_ent_ref,priority:1"`
	UserID   uint64 `gorm:"not null;column:user_id;index:idx_ent_user,priority:2;index:idx_ent_expired,priority:2"`

	Entitlement string `gorm:"type:varchar(32);not null;column:entitlement;index:idx_ent_user,priority:3;index:idx_ent_expired,priority:3;uniqueIndex:uk_ent_ref,priority:2"` // 权益标识: vip
	ProductCode string `gorm:"type:varchar(64);not null;default:'';column:product_code"`                                                                                      // 内部商品编码，如 vip_monthly/vip_yearly
	ProductID   string `gorm:"type:varchar(128);not null;default:'';column:product_id"`                                                                                       // 商店商品ID，如 Google Play/App Store productId
	Source      uint8  `gorm:"type:tinyint;not null;default:0;column:source"`                                                                                                 // 来源
	Status      uint8  `gorm:"type:tinyint;not null;default:1;column:status;index:idx_ent_user,priority:4"`                                                                   // 状态

	StartedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:started_at"`                                  // 开始时间
	ExpiredAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:expired_at;index:idx_ent_expired,priority:4"` // 到期时间

	RefID     *string `gorm:"type:varchar(128);column:ref_id;uniqueIndex:uk_ent_ref,priority:3"` // 外部订单/交易ID(可选)
	AutoRenew uint8   `gorm:"type:tinyint;not null;default:0;column:auto_renew"`                 // 是否自动续期(0/1)

	CreatedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:created_at"`
	UpdatedAt uint64 `gorm:"type:bigint unsigned;not null;default:0;column:updated_at"`
}

func (UserEntitlement) TableName() string { return "user_entitlements" }

// 默认权益标识
const (
	EntitlementVIP = "vip"
)

// Source 枚举（来源）
const (
	EntSourceManual     uint8 = 0
	EntSourceIOSIAP     uint8 = 1
	EntSourceGooglePlay uint8 = 2
	EntSourcePromo      uint8 = 3
)

// Status 枚举（状态）
const (
	EntStatusActive  uint8 = 1
	EntStatusExpired uint8 = 2
	EntStatusRevoked uint8 = 3
)

// AutoRenew 枚举（是否自动续期）
const (
	AutoRenewOff uint8 = 0
	AutoRenewOn  uint8 = 1
)

func EntitlementStatusText(status uint8) string {
	switch status {
	case EntStatusActive:
		return "active"
	case EntStatusExpired:
		return "expired"
	case EntStatusRevoked:
		return "revoked"
	default:
		return "unknown"
	}
}

func EntitlementSourceText(status uint8) string {
	switch status {
	case EntSourceManual:
		return "manual"
	case EntSourceIOSIAP:
		return "ios_iap"
	case EntSourceGooglePlay:
		return "google_play"
	case EntSourcePromo:
		return "promo"
	default:
		return "unknown"
	}
}

func EntitlementAutoRenew(status uint8) bool {
	switch status {
	case AutoRenewOff:
		return false
	case AutoRenewOn:
		return true
	default:
		return false
	}
}
