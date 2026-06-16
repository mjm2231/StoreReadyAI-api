package model

// AdminRolePermission 对应表：admin_role_permissions
//
// 说明：
// 1. 用于后台角色与权限的多对多关联；
// 2. tenant_id + role_id + permission_id 保持唯一，避免重复授权；
// 3. 时间字段统一使用秒级时间戳，与当前 admin 模块其它表保持一致。
// 4. 该表没有 deleted_at，变更授权时通常直接新增/删除关联记录。
//
// 放置位置说明：
// - 当前先放在 permissions/model 下，便于与 AdminPermission 一起维护权限域相关模型；
// - 后续如果 role_permission 关系操作逐渐增多，也可以再独立拆到 rolepermissions 模块。
type AdminRolePermission struct {
	ID           uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TenantID     uint64 `gorm:"column:tenant_id;type:bigint unsigned;not null;default:0;uniqueIndex:uk_admin_rp_tenant_role_permission;index:idx_admin_rp_tenant_role_id;index:idx_admin_rp_tenant_permission_id" json:"tenant_id"`
	RoleID       uint64 `gorm:"column:role_id;type:bigint unsigned;not null;uniqueIndex:uk_admin_rp_tenant_role_permission;index:idx_admin_rp_tenant_role_id" json:"role_id"`
	PermissionID uint64 `gorm:"column:permission_id;type:bigint unsigned;not null;uniqueIndex:uk_admin_rp_tenant_role_permission;index:idx_admin_rp_tenant_permission_id" json:"permission_id"`
	CreatedAt    uint64 `gorm:"column:created_at;type:bigint unsigned;not null;default:0;index:idx_admin_rp_created_at" json:"created_at"`
	UpdatedAt    uint64 `gorm:"column:updated_at;type:bigint unsigned;not null;default:0" json:"updated_at"`
}

func (AdminRolePermission) TableName() string {
	return "admin_role_permissions"
}
