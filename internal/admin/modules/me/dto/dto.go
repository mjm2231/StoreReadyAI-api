package dto

type CurrentUser struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	TenantID uint64 `json:"tenant_id"`
	Status   uint8  `json:"status"`
}

type CurrentRole struct {
	ID   uint64 `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type MeResponse struct {
	User        CurrentUser   `json:"admin_user"`
	Roles       []CurrentRole `json:"roles"`
	Permissions []string      `json:"perms"`
}
