package model

const (
	AdminRoleStatusActive   uint8 = 1
	AdminRoleStatusDisabled uint8 = 2
)

// AdminRole 对应表：admin_roles
//
// 说明：
// 1. 用于后台角色定义，不与管理员账号表混用；
// 2. 时间字段统一使用秒级时间戳，与当前 admin 模块其它表保持一致；
// 3. deleted_at 采用软删除时间戳语义，默认 0 表示未删除。
type AdminRole struct {
	TenantID  uint64 `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;uniqueIndex:uk_admin_role_tenant_code;index:idx_admin_role_tenant_id" json:"tenant_id"`
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Code      string `gorm:"column:code;type:varchar(64);not null;uniqueIndex:uk_admin_role_tenant_code" json:"code"`
	Status    uint8  `gorm:"column:status;type:tinyint unsigned;not null;default:1;index:idx_admin_role_status" json:"status"`
	Sort      int32  `gorm:"column:sort;type:int;not null;default:0;index:idx_admin_role_sort" json:"sort"`
	IsSystem  uint8  `gorm:"column:is_system;type:tinyint unsigned;not null;default:0" json:"is_system"`
	Remark    string `gorm:"column:remark;type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt uint64 `gorm:"column:created_at;type:bigint unsigned;not null;default:0" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;type:bigint unsigned;not null;default:0" json:"updated_at"`
	DeletedAt uint64 `gorm:"column:deleted_at;type:bigint unsigned;not null;default:0;index:idx_admin_role_deleted_at" json:"deleted_at"`
}

func (AdminRole) TableName() string {
	return "admin_roles"
}

func (r AdminRole) IsActive() bool {
	return r.Status == AdminRoleStatusActive && r.DeletedAt == 0
}

func (r AdminRole) IsDisabled() bool {
	return r.Status == AdminRoleStatusDisabled
}

func (r AdminRole) IsDeleted() bool {
	return r.DeletedAt > 0
}

func (r AdminRole) IsSystemBuiltin() bool {
	return r.IsSystem == 1
}
