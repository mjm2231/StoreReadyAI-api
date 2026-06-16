package model

const (
	AdminUserStatusActive   uint8 = 1
	AdminUserStatusDisabled uint8 = 2
	AdminUserStatusDeleted  uint8 = 3
)

// AdminUser 对应表：admin_users
//
// 说明：
// 1. 该模型用于后台管理员账号本身，不复用 C 端 users；
// 2. 时间字段统一使用秒级时间戳，与当前项目其它表保持一致；
// 3. deleted_at 采用软删除时间戳语义，默认 0 表示未删除。
type AdminUser struct {
	TenantID     uint64 `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;uniqueIndex:uk_admin_tenant_username;uniqueIndex:uk_admin_tenant_email;index:idx_admin_tenant_id" json:"tenant_id"`
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string `gorm:"column:username;type:varchar(64);not null;uniqueIndex:uk_admin_tenant_username" json:"username"`
	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null" json:"-"`
	Nickname     string `gorm:"column:nickname;type:varchar(64);not null;default:''" json:"nickname"`
	Email        string `gorm:"column:email;type:varchar(128);not null;default:'';uniqueIndex:uk_admin_tenant_email" json:"email"`
	Mobile       string `gorm:"column:mobile;type:varchar(32);not null;default:''" json:"mobile"`
	Avatar       string `gorm:"column:avatar;type:varchar(255);not null;default:''" json:"avatar"`
	Status       uint8  `gorm:"column:status;type:tinyint unsigned;not null;default:1;index:idx_admin_status" json:"status"`

	IsSuperAdmin uint8  `gorm:"column:is_super_admin;type:tinyint unsigned;not null;default:0" json:"is_super_admin"`
	LastLoginAt  uint64 `gorm:"column:last_login_at;type:bigint unsigned;not null;default:0" json:"last_login_at"`
	LastLoginIP  string `gorm:"column:last_login_ip;type:varchar(64);not null;default:''" json:"last_login_ip"`
	Remark       string `gorm:"column:remark;type:varchar(255);not null;default:''" json:"remark"`

	CreatedAt uint64 `gorm:"column:created_at;type:bigint unsigned;not null;default:0;index:idx_admin_created_at" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;type:bigint unsigned;not null;default:0" json:"updated_at"`
	DeletedAt uint64 `gorm:"column:deleted_at;type:bigint unsigned;not null;default:0;index:idx_admin_deleted_at" json:"deleted_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}

func (u AdminUser) IsActive() bool {
	return u.Status == AdminUserStatusActive && u.DeletedAt == 0
}

func (u AdminUser) IsDisabled() bool {
	return u.Status == AdminUserStatusDisabled
}

func (u AdminUser) IsDeleted() bool {
	return u.Status == AdminUserStatusDeleted || u.DeletedAt > 0
}

func (u AdminUser) IsSuper() bool {
	return u.IsSuperAdmin == 1
}
