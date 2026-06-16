package dto

// 权益（VIP）相关 DTO（HTTP 请求/响应）。
//
// 目标（MVP）：
// - 提供 VIP 状态查询接口（用于客户端展示/开关功能）。
// - 预留“手动开通/撤销”请求结构（仅后台/调试用，客户端不暴露）。
//
// 说明：
// - 所有接口统一使用 POST。
// - user_id 从服务端上下文（common.GetUID）获取，客户端不传。

// GetVIPStatusReq 获取 VIP 状态请求
// Route: POST /v1/vip/api/status
//
// 说明：一般无需 body 字段，保留扩展位。
type GetVIPStatusReq struct {
	// 预留：未来可支持查询不同 entitlement
}

// VIPStatusResp VIP 状态返回
//
// 约定：
// - is_vip：是否当前生效
// - expired_at：到期时间（Unix 秒），0 表示未知/永久（或未设置）
// - status：权益记录状态（1=active,2=expired,3=revoked），当 is_vip=false 时可用于展示原因
// - source：来源（0=manual,1=ios_iap,2=google_play,3=promo）
// - auto_renew：是否自动续期（0/1）
// - ref_id：外部交易/订单ID（可选）
type VIPStatusResp struct {
	Entitlement string  `json:"entitlement"` // 固定 vip
	IsVIP       bool    `json:"is_vip"`
	Status      uint8   `json:"status"`
	Source      uint8   `json:"source"`
	AutoRenew   uint8   `json:"auto_renew"`
	StartedAt   uint64  `json:"started_at"`
	ExpiredAt   uint64  `json:"expired_at"`
	RefID       *string `json:"ref_id"`
	UpdatedAt   uint64  `json:"updated_at"`
}

// GrantVIPReq 手动开通/延长 VIP（仅后台/调试）
// Route: POST /v1/vip/api/grant
//
// 注意：
// - 该接口建议仅 admin/内网可用。
// - duration_seconds 不传时可按默认 30 天。
type GrantVIPReq struct {
	DurationSeconds *uint64 `json:"duration_seconds"` // 授权时长（秒）
	ExpiredAt       *uint64 `json:"expired_at"`       // 也可直接指定到期时间（Unix 秒），优先级更高
	RefID           *string `json:"ref_id"`           // 可选：外部订单/活动ID
	AutoRenew       *uint8  `json:"auto_renew"`       // 0/1，可选
	Source          *uint8  `json:"source"`           // 0/1/2/3，可选；默认 manual
}

// RevokeVIPReq 撤销 VIP（仅后台/调试）
// Route: POST /v1/vip/api/revoke

type RevokeVIPReq struct {
	Reason *string `json:"reason"` // 可选：撤销原因（MVP 不落库，仅日志）
}
