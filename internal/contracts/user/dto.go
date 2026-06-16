package user

type UserVO struct {
	ID           uint64  `json:"id"`        // 内部自增ID（可不对外暴露，保留调试/后台使用）
	UID          uint64  `json:"uid"`       // 对外用户ID（业务ID）
	TenantID     uint64  `json:"tenant_id"` // 租户ID（MVP=0）
	Status       uint8   `json:"status"`    // 1=active...
	Email        *string `json:"email,omitempty"`
	Name         *string `json:"name,omitempty"`
	Avatar       *string `json:"avatar,omitempty"`
	Locale       *string `json:"locale,omitempty"`
	Timezone     *string `json:"timezone,omitempty"`
	IsVIP        uint8   `json:"is_vip"`
	VIPStartedAt uint64  `json:"vip_started_at"`
	VIPExpired   uint64  `json:"vip_expired_at"`
	LastLogin    uint64  `json:"last_login_at"`
	CreatedAt    uint64  `json:"created_at"`
	UpdatedAt    uint64  `json:"updated_at"`
}

type QueryUserFilter struct {
	Keyword *string `json:"keyword,omitempty"`
	IsVIP   *uint8  `json:"is_vip,omitempty"`
	Status  *uint8  `json:"status,omitempty"`
	StartAt *uint64 `json:"start_at,omitempty"`
	EndAt   *uint64 `json:"end_at,omitempty"`
	Page    PageReq `json:"page_req"`
}

type UpdateUserReq struct {
	Status     *uint8  `json:"status"` // 1=active...
	Email      *string `json:"email,omitempty"`
	Name       *string `json:"name,omitempty"`
	Avatar     *string `json:"avatar,omitempty"`
	IsVIP      *uint8  `json:"is_vip,omitempty"`
	VIPExpired *uint64 `json:"vip_expired_at"`
}

type PageReq struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}
