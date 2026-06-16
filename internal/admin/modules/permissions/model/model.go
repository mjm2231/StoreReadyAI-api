package model

const (
	AdminPermissionTypeMenu   uint8 = 1
	AdminPermissionTypePage   uint8 = 2
	AdminPermissionTypeAction uint8 = 3
)

const (
	AdminPermissionStatusActive   uint8 = 1
	AdminPermissionStatusDisabled uint8 = 2
)

// AdminPermission 对应表：admin_permissions
//
// 说明：
// 1. 用于后台权限定义，不与角色、管理员账号表混用；
// 2. 时间字段统一使用秒级时间戳，与当前 admin 模块其它表保持一致；
// 3. deleted_at 采用软删除时间戳语义，默认 0 表示未删除。
type AdminPermission struct {
	TenantID  uint64 `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;uniqueIndex:uk_admin_permission_tenant_code;index:idx_admin_permission_tenant_id" json:"tenant_id"`
	ID        uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Code      string `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uk_admin_permission_tenant_code" json:"code"`
	Module    string `gorm:"column:module;type:varchar(64);not null;default:'';index:idx_admin_permission_module" json:"module"`
	Type      uint8  `gorm:"column:type;type:tinyint unsigned;not null;default:1;index:idx_admin_permission_type" json:"type"`
	ParentID  uint64 `gorm:"column:parent_id;type:bigint unsigned;not null;default:0;index:idx_admin_permission_parent_id" json:"parent_id"`
	Path      string `gorm:"column:path;type:varchar(255);not null;default:''" json:"path"`
	Icon      string `gorm:"column:icon;type:varchar(64);not null;default:''" json:"icon"`
	Sort      int32  `gorm:"column:sort;type:int;not null;default:0;index:idx_admin_permission_sort" json:"sort"`
	Status    uint8  `gorm:"column:status;type:tinyint unsigned;not null;default:1;index:idx_admin_permission_status" json:"status"`
	IsSystem  uint8  `gorm:"column:is_system;type:tinyint unsigned;not null;default:0" json:"is_system"`
	Remark    string `gorm:"column:remark;type:varchar(255);not null;default:''" json:"remark"`
	CreatedAt uint64 `gorm:"column:created_at;type:bigint unsigned;not null;default:0" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;type:bigint unsigned;not null;default:0" json:"updated_at"`
	DeletedAt uint64 `gorm:"column:deleted_at;type:bigint unsigned;not null;default:0;index:idx_admin_permission_deleted_at" json:"deleted_at"`
}

func (AdminPermission) TableName() string {
	return "admin_permissions"
}

func (p AdminPermission) IsMenu() bool {
	return p.Type == AdminPermissionTypeMenu
}

func (p AdminPermission) IsPage() bool {
	return p.Type == AdminPermissionTypePage
}

func (p AdminPermission) IsAction() bool {
	return p.Type == AdminPermissionTypeAction
}

func (p AdminPermission) IsActive() bool {
	return p.Status == AdminPermissionStatusActive && p.DeletedAt == 0
}

func (p AdminPermission) IsDisabled() bool {
	return p.Status == AdminPermissionStatusDisabled
}

func (p AdminPermission) IsDeleted() bool {
	return p.DeletedAt > 0
}

func (p AdminPermission) IsSystemBuiltin() bool {
	return p.IsSystem == 1
}

func (p AdminPermission) IsRoot() bool {
	return p.ParentID == 0
}
