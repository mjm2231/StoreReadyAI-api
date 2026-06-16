package model

// UserRefreshToken corresponds to table `user_refresh_tokens`.
// Security note: token_hash stores SHA256(refresh_token) hex string (64 chars). Never store refresh token plaintext.
// Time fields are unix seconds (BIGINT UNSIGNED) to match the SQL schema.

type UserRefreshToken struct {
	ID       uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:主键" json:"id"`
	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_refresh_user,priority:1;index:idx_refresh_user_status,priority:1;comment:租户ID" json:"tenant_id"`
	UserID   uint64 `gorm:"column:user_id;not null;index:idx_refresh_user,priority:2;index:idx_refresh_user_status,priority:2;comment:用户ID" json:"user_id"`

	TokenHash string `gorm:"column:token_hash;type:char(64);not null;uniqueIndex:uk_refresh_token_hash,priority:1;comment:refresh token SHA256哈希（不存明文）" json:"-"`

	DeviceID   *string `gorm:"column:device_id;type:varchar(128);comment:设备ID（可选）" json:"device_id,omitempty"`
	DeviceName *string `gorm:"column:device_name;type:varchar(128);comment:设备名（可选）" json:"device_name,omitempty"`
	IP         *string `gorm:"column:ip;type:varchar(64);comment:IP（可选）" json:"ip,omitempty"`
	UserAgent  *string `gorm:"column:user_agent;type:varchar(255);comment:UA（可选）" json:"user_agent,omitempty"`

	Status     uint8  `gorm:"column:status;not null;default:1;index:idx_refresh_user_status,priority:3;comment:状态:1=active,2=revoked" json:"status"`
	ExpiredAt  uint64 `gorm:"column:expired_at;not null;default:0;index:idx_refresh_expired_at,priority:1;comment:过期时间戳秒" json:"expired_at"`
	LastUsedAt uint64 `gorm:"column:last_used_at;not null;default:0;comment:最近使用时间戳秒" json:"last_used_at"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0;comment:创建时间戳秒" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0;comment:更新时间戳秒" json:"updated_at"`
}

func (UserRefreshToken) TableName() string { return "user_refresh_tokens" }

const (
	RefreshTokenStatusActive  uint8 = 1
	RefreshTokenStatusRevoked uint8 = 2
)
