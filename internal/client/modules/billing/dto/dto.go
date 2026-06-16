package dto

// Billing 模块 DTO 定义。
//
// 说明：
// 1. DTO 负责 handler / service 间的数据传递；
// 2. 不直接承载数据库模型；
// 3. 时间字段统一使用秒级 Unix 时间戳，和当前项目保持一致。

// VerifyPurchaseReq 客户端上报购买校验请求。
//
// 用途：
// 1. 用户完成支付后，客户端将商店返回的购买凭证发给后端；
// 2. 后端再向 Google / Apple 校验，并更新订单与 entitlement。
type VerifyPurchaseReq struct {
	Platform      string `json:"platform" binding:"required"`
	ProductID     string `json:"product_id" binding:"required"`
	OrderID       string `json:"order_id"`
	PurchaseToken string `json:"purchase_token" binding:"required"`
	ReceiptData   string `json:"receipt_data"`
	PurchaseTime  uint64 `json:"purchase_time"`
	Source        string `json:"source"`
}

// RestorePurchaseReq 恢复购买请求。
//
// 说明：
// - 语义上与 verify 类似，但用于“恢复已有购买”。
// - 当前先沿用购买校验所需核心字段。
type RestorePurchaseReq struct {
	Platform      string `json:"platform" binding:"required"`
	ProductID     string `json:"product_id" binding:"required"`
	PurchaseToken string `json:"purchase_token" binding:"required"`
	OrderID       string `json:"order_id"`
	ReceiptData   string `json:"receipt_data"`
	Source        string `json:"source"`
}

// BillingConfigReq Billing 配置查询请求。
//
// 用途：
// 1. 客户端进入会员页时上报当前平台；
// 2. 服务端按平台返回当前可售商品配置；
// 3. 避免复用 VerifyPurchaseReq 这类与购买校验强相关的请求结构。
type BillingConfigReq struct {
	Platform string `json:"platform" binding:"required"`
}

// GoogleVerifySubscriptionReq Google Play 订阅校验请求。
//
// 用途：
// 1. 供 billing service / google play client 调用 Google Play Developer API 前使用；
// 2. 收口 package_name、subscription_id、purchase_token 等查询参数；
// 3. 与客户端直接上报的 VerifyPurchaseReq 区分开，避免混淆内部平台校验参数与外部 API 参数。
//
// 说明：
// - package_name 为 Google Play 应用包名；
// - subscription_id / product_id 在部分场景下可能相同，但这里都保留，便于兼容不同 Google API 版本；
// - base_plan_id 为可选字段。
type GoogleVerifySubscriptionReq struct {
	PackageName    string `json:"package_name" binding:"required"`
	ProductID      string `json:"product_id"`
	SubscriptionID string `json:"subscription_id" binding:"required"`
	BasePlanID     string `json:"base_plan_id"`
	OrderID        string `json:"order_id"`
	PurchaseToken  string `json:"purchase_token" binding:"required"`
	ReceiptData    string `json:"receipt_data"`
	PurchaseTime   uint64 `json:"purchase_time"`
}

// GoogleSubscriptionPurchase Google Play 订阅校验响应。
//
// 用途：
// 1. 承载 Google Play Developer API 校验后的核心字段；
// 2. 由 google verifier / google play client 转为 service 层统一结果前使用；
// 3. 保留最小必要字段，避免 dto 直接镜像 Google 官方所有返回。
type GoogleSubscriptionPurchase struct {
	ProductID       string `json:"product_id"`
	SubscriptionID  string `json:"subscription_id"`
	BasePlanID      string `json:"base_plan_id"`
	OrderID         string `json:"order_id"`
	OriginalOrderID string `json:"original_order_id"`
	PurchaseToken   string `json:"purchase_token"`
	PurchaseState   string `json:"purchase_state"`
	Acknowledged    bool   `json:"acknowledged"`
	AutoRenewing    bool   `json:"auto_renewing"`
	PurchaseTime    uint64 `json:"purchase_time"`
	ExpireTime      uint64 `json:"expire_time"`
	Currency        string `json:"currency"`
	AmountMicros    uint64 `json:"amount_micros"`
	ReceiptData     string `json:"receipt_data"`
	RawPayload      string `json:"raw_payload"`
	ProductCode     string `json:"product_code"`
}

// EntitlementResp 当前权益响应。
//
// 说明：
// 1. 前半部分为 App 统一权益快照字段，字段名需要和 auth/user 模块返回保持一致；
// 2. 客户端用这些字段更新本地 EntitlementSnapshots，供离线 VIP / 同步 / 免费上限判断；
// 3. 后半部分为 Billing 页面展示字段，用于展示商品、平台、自动续费和订单状态。
type EntitlementResp struct {
	// App 统一权益字段。
	IsVIP                  uint8  `json:"is_vip"`
	VIPStartedAt           uint64 `json:"vip_started_at"`
	VIPExpiredAt           uint64 `json:"vip_expired_at"`
	FreeLimit              uint8  `json:"free_limit"`
	SyncEnabled            bool   `json:"sync_enabled"`
	UnlimitedSubscriptions bool   `json:"unlimited_subscriptions"`

	// Billing 展示字段。
	EntitlementCode string `json:"entitlement_code"`
	ProductCode     string `json:"product_code"`
	ProductID       string `json:"product_id"`
	Platform        string `json:"platform"`
	Status          string `json:"status"`
	AutoRenew       bool   `json:"auto_renew"`
	ExpiresAt       uint64 `json:"expires_at"`
}

// VerifyPurchaseResp 购买校验响应。
//
// 说明：
// - 当前对客户端最重要的是最新权益状态；
// - 可按需附带订单信息，便于排查与前端埋点。
type VerifyPurchaseResp struct {
	Order       BillingOrderVO  `json:"order"`
	Entitlement EntitlementResp `json:"entitlement"`
}

// RestorePurchaseResp 恢复购买响应。
type RestorePurchaseResp struct {
	Order       BillingOrderVO  `json:"order"`
	Entitlement EntitlementResp `json:"entitlement"`
}

// BillingOrderVO 返回给客户端的订单视图。
type BillingOrderVO struct {
	ID                 uint64 `json:"id"`
	Platform           string `json:"platform"`
	ProductID          string `json:"product_id"`
	SubscriptionID     string `json:"subscription_id"`
	BasePlanID         string `json:"base_plan_id"`
	OrderID            string `json:"order_id"`
	OriginalOrderID    string `json:"original_order_id"`
	PurchaseState      string `json:"purchase_state"`
	Acknowledged       bool   `json:"acknowledged"`
	AutoRenewing       bool   `json:"auto_renewing"`
	PurchaseTime       uint64 `json:"purchase_time"`
	ExpireTime         uint64 `json:"expire_time"`
	Currency           string `json:"currency"`
	AmountMicros       uint64 `json:"amount_micros"`
	VerifyStatus       string `json:"verify_status"`
	VerifyErrorCode    string `json:"verify_error_code"`
	VerifyErrorMessage string `json:"verify_error_message"`
	LastVerifiedAt     uint64 `json:"last_verified_at"`
	CreatedAt          uint64 `json:"created_at"`
	UpdatedAt          uint64 `json:"updated_at"`
}

// BillingProductVO 商品配置视图。
//
// 说明：
// - 给客户端返回业务商品配置时使用；
// - 真正价格/标题/描述仍应以商店查询结果为准。
type BillingProductVO struct {
	ID                uint64 `json:"id"`
	ProductCode       string `json:"product_code"`
	Platform          string `json:"platform"`
	StoreProductID    string `json:"store_product_id"`
	ProductType       string `json:"product_type"`
	SubscriptionGroup string `json:"subscription_group"`
	Status            int32  `json:"status"`
	IsRecommended     bool   `json:"is_recommended"`
	Sort              int32  `json:"sort"`
	Title             string `json:"title"`
}

// BillingConfigResp Billing 页面配置响应。
//
// 用途：
// 1. 返回应该展示哪些商品；
// 2. 指定默认推荐商品；
// 3. 允许后端动态控制 Paywall 展示策略。
type BillingConfigResp struct {
	Products               []BillingProductVO `json:"products"`
	RecommendedProductCode string             `json:"recommended_product_code"`
}

// AppleNotificationReq Apple Server Notification 原始请求。
type AppleNotificationReq struct {
	SignedPayload string `json:"signedPayload" binding:"required"`
}

// GoogleNotificationReq Google RTDN / PubSub 推送原始请求。
//
// 当前先保留最外层通用负载，后续可按真实接入格式细化。
type GoogleNotificationReq struct {
	Message string `json:"message"`
	Data    string `json:"data"`
}
