package entitlement

import (
	"context"
	"fmt"
	"os"
	"storeready_ai/internal/client/modules/user/model"
	"time"
)

const DefaultFreeLimit uint8 = 5

// Service 定义 entitlement 模块对外暴露的跨模块能力契约。
//
// 说明：
// 1. 该接口供 billing 等其他模块依赖；
// 2. 放在 contracts 层，避免 billing 与 entitlement 互相直接依赖；
// 3. 这里只放跨模块真正需要的最小集合，不放 HTTP DTO、数据库模型等实现细节。
type Service interface {
	// RefreshByBilling 根据 billing 订单结果刷新权益。
	RefreshByBilling(ctx context.Context, req RefreshByBillingReq) (*CurrentEntitlement, error)

	// GetCurrent 查询当前用户的指定权益。
	GetCurrent(ctx context.Context, tenantID, userID uint64, entitlementCode string) (*CurrentEntitlement, error)
}

// RefreshByBillingReq billing -> entitlement 的刷新请求。
//
// 说明：
// 1. 由 billing 校验成功后发起；
// 2. entitlement 以此更新 user_entitlements；
// 3. 这里只保留跨模块真正需要的字段。
type RefreshByBillingReq struct {
	TenantID        uint64
	UserID          uint64
	UID             uint64
	EntitlementCode string
	ProductCode     string
	ProductID       string
	Platform        string
	Status          string
	AutoRenew       bool
	ExpiresAt       uint64
	OriginalOrderID string
	PurchaseToken   string
	Source          string
}

// CurrentEntitlement 表示当前权益快照。
//
// 说明：
// 1. 这是跨模块契约对象，不是 HTTP DTO；
// 2. handler 层如需返回给客户端，可再转换为模块自己的响应结构；
// 3. 命名避免使用 dto，减少和接口层对象混淆。
type CurrentEntitlement struct {
	Entitlement     Entitlement
	EntitlementCode string
	ProductCode     string
	ProductID       string
	Platform        string
	Status          string
	AutoRenew       bool
	ExpiresAt       uint64
}

// /VIP权益相关
type Entitlement struct {
	IsVIP                  uint8  `json:"is_vip"`
	VIPStartedAt           uint64 `json:"vip_started_at"`
	VIPExpiredAt           uint64 `json:"vip_expired_at"`
	FreeLimit              uint8  `json:"free_limit"`
	SyncEnabled            bool   `json:"sync_enabled"`
	UnlimitedSubscriptions bool   `json:"unlimited_subscriptions"`
}

func NewEntitlement(user model.User) Entitlement {
	now := uint64(time.Now().Unix())

	isVipValid := user.IsVIP == 1 &&
		user.VIPStartedAt > 0 &&
		user.VIPExpiredAt > 0 &&
		user.VIPStartedAt <= now &&
		now < user.VIPExpiredAt
	if !isVipValid {
		fmtStr := "NewEntitlement invalid vip: uid=%d is_vip=%d now=%d vip_started_at=%d vip_expired_at=%d started_ok=%v not_expired=%v\n"
		_, _ = fmt.Fprintf(
			os.Stdout,
			fmtStr,
			user.UID,
			user.IsVIP,
			now,
			user.VIPStartedAt,
			user.VIPExpiredAt,
			user.VIPStartedAt > 0 && user.VIPStartedAt <= now,
			user.VIPExpiredAt > 0 && now < user.VIPExpiredAt,
		)
	}
	var isVIP uint8
	if isVipValid {
		isVIP = 1
	}

	resp := Entitlement{
		IsVIP:                  isVIP,
		VIPStartedAt:           user.VIPStartedAt,
		VIPExpiredAt:           user.VIPExpiredAt,
		FreeLimit:              DefaultFreeLimit,
		SyncEnabled:            isVipValid,
		UnlimitedSubscriptions: isVipValid,
	}

	return resp
}

func (r *Entitlement) Normalize() {
	if r.FreeLimit == 0 {
		r.FreeLimit = DefaultFreeLimit
	}
}
