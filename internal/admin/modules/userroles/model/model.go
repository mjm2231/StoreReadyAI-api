package model

// AdminUserRole 对应表：admin_user_roles
//
// 说明：
// 1. 用于后台管理员与角色的多对多关联；
// 2. tenant_id + admin_user_id + role_id 保持唯一，避免重复绑定；
// 3. 时间字段统一使用秒级时间戳，与当前 admin 模块其它表保持一致；
// 4. 该表没有 deleted_at，角色绑定变更时通常直接新增/删除关联记录。
type AdminUserRole struct {
	ID          uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TenantID    uint64 `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;uniqueIndex:uk_admin_ur_tenant_user_role;index:idx_admin_ur_tenant_admin_user_id;index:idx_admin_ur_tenant_role_id" json:"tenant_id"`
	AdminUserID uint64 `gorm:"column:admin_user_id;type:bigint unsigned;not null;uniqueIndex:uk_admin_ur_tenant_user_role;index:idx_admin_ur_tenant_admin_user_id" json:"admin_user_id"`
	RoleID      uint64 `gorm:"column:role_id;type:bigint unsigned;not null;uniqueIndex:uk_admin_ur_tenant_user_role;index:idx_admin_ur_tenant_role_id" json:"role_id"`
	CreatedAt   uint64 `gorm:"column:created_at;type:bigint unsigned;not null;default:0;index:idx_admin_ur_created_at" json:"created_at"`
	UpdatedAt   uint64 `gorm:"column:updated_at;type:bigint unsigned;not null;default:0" json:"updated_at"`
}

func (AdminUserRole) TableName() string {
	return "admin_user_roles"
}
