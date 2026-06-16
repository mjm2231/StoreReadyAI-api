package model

type User struct {
	ID       uint64 `gorm:"column:id;primaryKey;autoIncrement;comment:自增ID" json:"id"`
	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_users_tenant_id,priority:1;comment:租户ID（MVP固定0，后期多租户扩展）" json:"tenant_id"`
	UID      uint64 `gorm:"column:uid;not null;default:0;index:idx_users_uid,priority:1;comment:用户唯一ID（对外暴露，MVP先用自增ID，后期可改为雪花ID或UUID）" json:"uid"`

	Status uint8 `gorm:"column:status;not null;default:1;comment:状态:0=unknown,1=active,2=banned,3=deleted" json:"status"`

	LoginType string `gorm:"column:login_type;type:varchar(32);not null;default:'password';index:idx_users_login_type,priority:1;comment:主登录方式:password/google/apple/github" json:"login_type"`

	Email        *string `gorm:"column:email;type:varchar(255);index:idx_users_email,priority:1;comment:邮箱（可为空；登录唯一性由 user_auth_identities 控制）" json:"email,omitempty"`
	PasswordHash *string `gorm:"column:password_hash;type:varchar(255);comment:密码哈希（仅账号密码登录使用；第三方登录为空）" json:"-"`
	Name         *string `gorm:"column:name;type:varchar(128);comment:昵称" json:"name,omitempty"`
	Avatar       *string `gorm:"column:avatar;type:varchar(512);comment:头像URL" json:"avatar,omitempty"`

	Locale   *string `gorm:"column:locale;type:varchar(32);comment:语言/地区（可选）" json:"locale,omitempty"`
	Timezone *string `gorm:"column:timezone;type:varchar(64);comment:时区（可选）" json:"timezone,omitempty"`

	IsVIP        uint8  `gorm:"column:is_vip;not null;default:0;comment:VIP标记（后期付费）" json:"is_vip"`
	VIPStartedAt uint64 `gorm:"column:vip_started_at;not null;default:0;comment:VIP开始时间戳秒" json:"vip_started_at"`
	VIPExpiredAt uint64 `gorm:"column:vip_expired_at;not null;default:0;comment:VIP到期时间戳秒" json:"vip_expired_at"`

	LastLoginAt uint64 `gorm:"column:last_login_at;not null;default:0;index:idx_users_last_login_at,priority:1;comment:最近登录时间戳秒" json:"last_login_at"`
	CreatedAt   uint64 `gorm:"column:created_at;not null;default:0;comment:创建时间戳秒" json:"created_at"`
	UpdatedAt   uint64 `gorm:"column:updated_at;not null;default:0;comment:更新时间戳秒" json:"updated_at"`
}

func (User) TableName() string { return "users" }

const (
	UserStatusUnknown uint8 = 0
	UserStatusActive  uint8 = 1
	UserStatusBanned  uint8 = 2
	UserStatusDeleted uint8 = 3
)

const (
	UserLoginTypePassword = "password"
	UserLoginTypeGoogle   = "google"
	UserLoginTypeApple    = "apple"
	UserLoginTypeGitHub   = "github"
)
