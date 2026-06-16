package dto

type RoleInfo struct {
	TenantID  uint64 `json:"tenant_id"`
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Status    uint8  `json:"status"`
	Sort      int32  `json:"sort"`
	IsSystem  uint8  `json:"is_system"`
	Remark    string `json:"remark"`
	CreatedAt uint64 `json:"created_at"`
	UpdatedAt uint64 `json:"updated_at"`
	DeletedAt uint64 `json:"deleted_at"`
}

type AdminUserItem struct {
	TenantID     uint64   `json:"tenant_id"`
	ID           uint64   `json:"id"`
	Username     string   `json:"username"`
	Nickname     string   `json:"nickname"`
	Email        string   `json:"email"`
	Mobile       string   `json:"mobile"`
	Avatar       string   `json:"avatar"`
	Status       uint8    `json:"status"`
	IsSuperAdmin uint8    `json:"is_super_admin"`
	LastLoginAt  uint64   `json:"last_login_at"`
	LastLoginIP  string   `json:"last_login_ip"`
	Remark       string   `json:"remark"`
	CreatedAt    uint64   `json:"created_at"`
	UpdatedAt    uint64   `json:"updated_at"`
	Roles        []string // 角色
	Perms        []string // 权限点（超管可能为 ["*"]）
}

type Snapshot struct {
	Roles           []RoleInfo
	PermissionCodes []string
}
