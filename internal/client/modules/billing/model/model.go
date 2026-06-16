package model

// Billing 模块模型定义。
//
// 说明：
// 1. orders：订单当前态 / 幂等主表；
// 2. transactions：交易历史流水；
// 3. events：外部/内部事件流水；
// 4. products：业务商品配置。
//
// 时间字段统一使用秒级 Unix 时间戳，便于和当前项目其他模块保持一致。

const (
	PlatformIOS     = "ios"
	PlatformAndroid = "android"
)

const (
	ProductTypeSubscription  = "subscription"
	ProductTypeConsumable    = "consumable"
	ProductTypeNonConsumable = "non_consumable"
)

const (
	BillingProductStatusEnabled  = 1
	BillingProductStatusDisabled = 2
)

const (
	VerifyStatusPending = "pending"
	VerifyStatusSuccess = "success"
	VerifyStatusFailed  = "failed"
)

const (
	PurchaseStatePurchased = "purchased"
	PurchaseStatePending   = "pending"
	PurchaseStateCanceled  = "canceled"
	PurchaseStateRefunded  = "refunded"
	PurchaseStateExpired   = "expired"
	PurchaseStateRevoked   = "revoked"
)

const (
	EventTypeVerify            = "verify"
	EventTypeRestore           = "restore"
	EventTypeRTDN              = "rtdn"
	EventTypeAppleNotification = "apple_notification"
	EventTypeRefund            = "refund"
	EventTypeRevoke            = "revoke"
	EventTypeRenew             = "renew"
	EventTypeExpire            = "expire"
)

const (
	EventSourceClient = "client"
	EventSourceGoogle = "google"
	EventSourceApple  = "apple"
	EventSourceSystem = "system"
)

const (
	EventStatusPending   = "pending"
	EventStatusProcessed = "processed"
	EventStatusFailed    = "failed"
	EventStatusIgnored   = "ignored"
)

const (
	TransactionTypePurchase = "purchase"
	TransactionTypeRenew    = "renew"
	TransactionTypeRefund   = "refund"
	TransactionTypeRevoke   = "revoke"
	TransactionTypeRestore  = "restore"
	TransactionTypeExpire   = "expire"
)

const (
	TransactionStateSuccess = "success"
	TransactionStateFailed  = "failed"
	TransactionStatePending = "pending"
)

// BillingOrder 订单当前态表。
//
// 用途：
// 1. 作为购买校验后的当前快照；
// 2. 作为 purchase_token 幂等主表；
// 3. 给 entitlement 模块提供最新订单状态依据。
type BillingOrder struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_billing_order_user,priority:1;index:idx_billing_order_uid,priority:1" json:"tenant_id"`
	UserID   uint64 `gorm:"column:user_id;not null;default:0;index:idx_billing_order_user,priority:2" json:"user_id"`
	UID      uint64 `gorm:"column:uid;not null;default:0;index:idx_billing_order_uid,priority:2" json:"uid"`

	Platform       string `gorm:"column:platform;type:varchar(16);not null;uniqueIndex:uk_billing_order_platform_token,priority:1;index:idx_billing_order_order_id,priority:1;index:idx_billing_order_original_order_id,priority:1" json:"platform"`
	ProductID      string `gorm:"column:product_id;type:varchar(128);not null;default:''" json:"product_id"`
	SubscriptionID string `gorm:"column:subscription_id;type:varchar(128);not null;default:''" json:"subscription_id"`
	BasePlanID     string `gorm:"column:base_plan_id;type:varchar(128);not null;default:''" json:"base_plan_id"`

	OrderID         string `gorm:"column:order_id;type:varchar(128);not null;default:'';index:idx_billing_order_order_id,priority:2" json:"order_id"`
	OriginalOrderID string `gorm:"column:original_order_id;type:varchar(128);not null;default:'';index:idx_billing_order_original_order_id,priority:2" json:"original_order_id"`
	PurchaseToken   string `gorm:"column:purchase_token;type:varchar(512);not null;default:'';uniqueIndex:uk_billing_order_platform_token,priority:2" json:"purchase_token"`
	ReceiptData     string `gorm:"column:receipt_data;type:mediumtext" json:"receipt_data"`

	PurchaseState string `gorm:"column:purchase_state;type:varchar(32);not null;default:''" json:"purchase_state"`
	Acknowledged  bool   `gorm:"column:acknowledged;not null;default:0" json:"acknowledged"`
	AutoRenewing  bool   `gorm:"column:auto_renewing;not null;default:0" json:"auto_renewing"`

	PurchaseTime uint64 `gorm:"column:purchase_time;not null;default:0" json:"purchase_time"`
	ExpireTime   uint64 `gorm:"column:expire_time;not null;default:0;index:idx_billing_order_expire_time" json:"expire_time"`

	Currency     string `gorm:"column:currency;type:varchar(8);not null;default:''" json:"currency"`
	AmountMicros uint64 `gorm:"column:amount_micros;not null;default:0" json:"amount_micros"`

	VerifyStatus       string `gorm:"column:verify_status;type:varchar(32);not null;default:'';index:idx_billing_order_verify_status" json:"verify_status"`
	VerifyErrorCode    string `gorm:"column:verify_error_code;type:varchar(64);not null;default:''" json:"verify_error_code"`
	VerifyErrorMessage string `gorm:"column:verify_error_message;type:varchar(255);not null;default:''" json:"verify_error_message"`
	LastVerifiedAt     uint64 `gorm:"column:last_verified_at;not null;default:0" json:"last_verified_at"`

	RawPayload string `gorm:"column:raw_payload;type:json" json:"raw_payload"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0;index:idx_billing_order_updated_at" json:"updated_at"`
}

func (BillingOrder) TableName() string {
	return "billing_orders"
}

// BillingTransaction 交易流水表。
//
// 用途：
// 1. 记录 purchase / renew / refund / revoke / restore 等历史交易变化；
// 2. 便于排查、统计、对账。
type BillingTransaction struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_billing_tx_user,priority:1;index:idx_billing_tx_uid,priority:1" json:"tenant_id"`
	UserID   uint64 `gorm:"column:user_id;not null;default:0;index:idx_billing_tx_user,priority:2" json:"user_id"`
	UID      uint64 `gorm:"column:uid;not null;default:0;index:idx_billing_tx_uid,priority:2" json:"uid"`

	Platform  string `gorm:"column:platform;type:varchar(16);not null;index:idx_billing_tx_platform_order,priority:1;index:idx_billing_tx_original_order,priority:1;index:idx_billing_tx_purchase_token,priority:1" json:"platform"`
	ProductID string `gorm:"column:product_id;type:varchar(128);not null;default:''" json:"product_id"`

	OrderID         string `gorm:"column:order_id;type:varchar(128);not null;default:'';index:idx_billing_tx_platform_order,priority:2" json:"order_id"`
	OriginalOrderID string `gorm:"column:original_order_id;type:varchar(128);not null;default:'';index:idx_billing_tx_original_order,priority:2" json:"original_order_id"`
	PurchaseToken   string `gorm:"column:purchase_token;type:varchar(512);not null;default:'';index:idx_billing_tx_purchase_token,priority:2" json:"purchase_token"`

	TransactionType  string `gorm:"column:transaction_type;type:varchar(32);not null;default:'';index:idx_billing_tx_type" json:"transaction_type"`
	TransactionState string `gorm:"column:transaction_state;type:varchar(32);not null;default:''" json:"transaction_state"`

	AmountMicros uint64 `gorm:"column:amount_micros;not null;default:0" json:"amount_micros"`
	Currency     string `gorm:"column:currency;type:varchar(8);not null;default:''" json:"currency"`

	TransactionTime uint64 `gorm:"column:transaction_time;not null;default:0;index:idx_billing_tx_time" json:"transaction_time"`
	RawPayload      string `gorm:"column:raw_payload;type:json" json:"raw_payload"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0" json:"updated_at"`
}

func (BillingTransaction) TableName() string {
	return "billing_transactions"
}

// BillingEvent 事件流水表。
//
// 用途：
// 1. 记录客户端 verify / restore 请求；
// 2. 记录 Google RTDN / Apple Server Notifications；
// 3. 记录事件处理状态与原始 payload。
type BillingEvent struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;index:idx_billing_event_user,priority:1;index:idx_billing_event_uid,priority:1" json:"tenant_id"`
	UserID   uint64 `gorm:"column:user_id;not null;default:0;index:idx_billing_event_user,priority:2" json:"user_id"`
	UID      uint64 `gorm:"column:uid;not null;default:0;index:idx_billing_event_uid,priority:2" json:"uid"`

	Platform    string `gorm:"column:platform;type:varchar(16);not null;index:idx_billing_event_platform_type,priority:1;index:idx_billing_event_order_id,priority:1;index:idx_billing_event_original_order_id,priority:1;index:idx_billing_event_purchase_token,priority:1" json:"platform"`
	EventType   string `gorm:"column:event_type;type:varchar(64);not null;default:'';index:idx_billing_event_platform_type,priority:2" json:"event_type"`
	EventSource string `gorm:"column:event_source;type:varchar(32);not null;default:''" json:"event_source"`

	OrderID         string `gorm:"column:order_id;type:varchar(128);not null;default:'';index:idx_billing_event_order_id,priority:2" json:"order_id"`
	OriginalOrderID string `gorm:"column:original_order_id;type:varchar(128);not null;default:'';index:idx_billing_event_original_order_id,priority:2" json:"original_order_id"`
	PurchaseToken   string `gorm:"column:purchase_token;type:varchar(512);not null;default:'';index:idx_billing_event_purchase_token,priority:2" json:"purchase_token"`
	ProductID       string `gorm:"column:product_id;type:varchar(128);not null;default:''" json:"product_id"`

	EventStatus  string `gorm:"column:event_status;type:varchar(32);not null;default:'';index:idx_billing_event_status" json:"event_status"`
	ErrorCode    string `gorm:"column:error_code;type:varchar(64);not null;default:''" json:"error_code"`
	ErrorMessage string `gorm:"column:error_message;type:varchar(255);not null;default:''" json:"error_message"`

	EventTime   uint64 `gorm:"column:event_time;not null;default:0;index:idx_billing_event_event_time" json:"event_time"`
	ProcessedAt uint64 `gorm:"column:processed_at;not null;default:0" json:"processed_at"`

	RawPayload string `gorm:"column:raw_payload;type:json" json:"raw_payload"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0;index:idx_billing_event_updated_at" json:"updated_at"`
}

func (BillingEvent) TableName() string {
	return "billing_events"
}

// BillingProduct 商品配置表。
//
// 用途：
// 1. 维护业务商品编码和商店商品 ID 映射；
// 2. 支持推荐商品、展示排序、启停控制。
type BillingProduct struct {
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;uniqueIndex:uk_billing_product_code_platform,priority:1" json:"tenant_id"`

	ProductCode    string `gorm:"column:product_code;type:varchar(64);not null;uniqueIndex:uk_billing_product_code_platform,priority:2" json:"product_code"`
	Platform       string `gorm:"column:platform;type:varchar(16);not null;uniqueIndex:uk_billing_product_code_platform,priority:3;uniqueIndex:uk_billing_product_store_id,priority:1" json:"platform"`
	StoreProductID string `gorm:"column:store_product_id;type:varchar(128);not null;uniqueIndex:uk_billing_product_store_id,priority:2" json:"store_product_id"`

	ProductType       string `gorm:"column:product_type;type:varchar(32);not null;default:'subscription'" json:"product_type"`
	SubscriptionGroup string `gorm:"column:subscription_group;type:varchar(64);not null;default:''" json:"subscription_group"`

	Status        int32 `gorm:"column:status;not null;default:1;index:idx_billing_product_status" json:"status"`
	IsRecommended bool  `gorm:"column:is_recommended;not null;default:0" json:"is_recommended"`
	Sort          int32 `gorm:"column:sort;not null;default:0;index:idx_billing_product_sort" json:"sort"`

	CreatedAt uint64 `gorm:"column:created_at;not null;default:0" json:"created_at"`
	UpdatedAt uint64 `gorm:"column:updated_at;not null;default:0" json:"updated_at"`
}

func (BillingProduct) TableName() string {
	return "billing_products"
}
