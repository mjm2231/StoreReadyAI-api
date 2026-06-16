package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"storeready_ai/internal/client/modules/billing/dto"
	"storeready_ai/internal/client/modules/billing/model"
	errx "storeready_ai/internal/pkg/errors"
)

// GooglePlaySubscriptionExecutor Google Play 订阅查询执行器。
//
// 说明：
// 1. 这里先把“真正调用 Google Play Developer API”的动作抽象成 executor；
// 2. 便于当前先接你现有 HTTP / SDK 封装，后续再平滑替换；
// 3. 入参和返回值都保持 map[string]any，降低接入门槛。
type GooglePlaySubscriptionExecutor func(ctx context.Context, req GooglePlaySubscriptionQuery) (map[string]any, error)

// GooglePlaySubscriptionQuery Google Play 订阅查询请求。
type GooglePlaySubscriptionQuery struct {
	PackageName    string
	ProductID      string
	SubscriptionID string
	BasePlanID     string
	OrderID        string
	PurchaseToken  string
}

// GooglePlayClient Google Play 客户端。
type GooglePlayClient struct {
	packageName string
	executor    GooglePlaySubscriptionExecutor
	nowFunc     func() time.Time
}

// NewGooglePlayClient 创建 Google Play client。
func NewGooglePlayClient(packageName string, executor GooglePlaySubscriptionExecutor) *GooglePlayClient {
	return &GooglePlayClient{
		packageName: strings.TrimSpace(packageName),
		executor:    executor,
		nowFunc:     time.Now,
	}
}

// VerifySubscription 实现 GoogleVerifier。
//
// 当前设计：
// 1. client 负责和 Google Play Developer API 或其封装交互；
// 2. 并把外部返回收口成 GoogleSubscriptionPurchase；
// 3. 上层 googleVerifierImpl 再继续适配为 PlatformVerifyResult。
func (c *GooglePlayClient) VerifySubscription(ctx context.Context, req dto.GoogleVerifySubscriptionReq) (*dto.GoogleSubscriptionPurchase, error) {
	if c == nil || c.executor == nil {
		return nil, errx.New(errx.CodeInternal, "google play executor not configured")
	}
	if strings.TrimSpace(c.packageName) == "" {
		return nil, errx.New(errx.CodeInternal, "google play package name not configured")
	}

	productID := strings.TrimSpace(req.ProductID)
	purchaseToken := strings.TrimSpace(req.PurchaseToken)
	if productID == "" {
		return nil, errx.New(errx.CodeInvalidParam, "google product_id required")
	}
	if purchaseToken == "" {
		return nil, errx.New(errx.CodeInvalidParam, "google purchase_token required")
	}

	payload, err := c.executor(ctx, GooglePlaySubscriptionQuery{
		PackageName:    c.packageName,
		ProductID:      productID,
		SubscriptionID: firstNonEmpty(strings.TrimSpace(req.SubscriptionID), productID),
		BasePlanID:     strings.TrimSpace(req.BasePlanID),
		OrderID:        strings.TrimSpace(req.OrderID),
		PurchaseToken:  purchaseToken,
	})
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "execute google play subscription query failed")
	}
	if payload == nil {
		return nil, errx.New(errx.CodeInternal, "google play subscription payload is nil")
	}

	return c.mapSubscriptionPayload(req, payload)
}

func (c *GooglePlayClient) mapSubscriptionPayload(req dto.GoogleVerifySubscriptionReq, payload map[string]any) (*dto.GoogleSubscriptionPurchase, error) {
	rawPayload, _ := json.Marshal(payload)

	productID := firstNonEmpty(asString(payload["productId"]), req.ProductID)
	subscriptionID := firstNonEmpty(
		asString(payload["subscriptionId"]),
		asString(payload["lineItems.productId"]),
		req.SubscriptionID,
		productID,
	)
	basePlanID := firstNonEmpty(
		asString(payload["basePlanId"]),
		asString(payload["lineItems.basePlanId"]),
		req.BasePlanID,
	)
	orderID := firstNonEmpty(
		asString(payload["latestOrderId"]),
		asString(payload["orderId"]),
		req.OrderID,
	)
	originalOrderID := firstNonEmpty(
		asString(payload["linkedPurchaseTokenOrderId"]),
		asString(payload["originalOrderId"]),
		orderID,
	)
	purchaseToken := firstNonEmpty(asString(payload["purchaseToken"]), req.PurchaseToken)

	purchaseTime := firstPositiveUint64(
		parseUnixMillisString(asString(payload["startTimeMillis"])),
		parseRFC3339ToUnix(asString(payload["startTime"])),
		req.PurchaseTime,
	)
	expireTime := firstPositiveUint64(
		parseUnixMillisString(asString(payload["expiryTimeMillis"])),
		parseRFC3339ToUnix(asString(payload["expiryTime"])),
	)

	ackState := firstNonEmpty(asString(payload["acknowledgementState"]), asString(payload["acknowledged"]))
	autoRenewState := firstNonEmpty(asString(payload["autoRenewingPlan.autoRenewEnabled"]), asString(payload["autoRenewing"]))
	priceMicros := firstPositiveUint64(
		parseUint64String(asString(payload["priceAmountMicros"])),
		parseUint64String(asString(payload["lineItems.priceAmountMicros"])),
	)
	currency := firstNonEmpty(asString(payload["priceCurrencyCode"]), asString(payload["lineItems.priceCurrencyCode"]))
	productCode := firstNonEmpty(asString(payload["productCode"]), req.ProductID)

	purchaseState := c.normalizePurchaseState(payload, expireTime)

	return &dto.GoogleSubscriptionPurchase{
		ProductID:       productID,
		SubscriptionID:  subscriptionID,
		BasePlanID:      basePlanID,
		OrderID:         orderID,
		OriginalOrderID: originalOrderID,
		PurchaseToken:   purchaseToken,
		PurchaseState:   purchaseState,
		Acknowledged:    parseBoolLoose(ackState),
		AutoRenewing:    parseBoolLoose(autoRenewState),
		PurchaseTime:    purchaseTime,
		ExpireTime:      expireTime,
		Currency:        currency,
		AmountMicros:    priceMicros,
		ReceiptData:     req.ReceiptData,
		RawPayload:      string(rawPayload),
		ProductCode:     productCode,
	}, nil
}

func (c *GooglePlayClient) normalizePurchaseState(payload map[string]any, expireAt uint64) string {
	state := strings.TrimSpace(strings.ToLower(firstNonEmpty(
		asString(payload["subscriptionState"]),
		asString(payload["purchaseState"]),
		asString(payload["state"]),
	)))
	if state == "" {
		if expireAt > 0 && expireAt <= uint64(c.nowFunc().Unix()) {
			return model.PurchaseStateExpired
		}
		return model.PurchaseStatePurchased
	}

	switch state {
	case "active", "subscription_state_active", "purchased", "purchase_state_purchased":
		if expireAt > 0 && expireAt <= uint64(c.nowFunc().Unix()) {
			return model.PurchaseStateExpired
		}
		return model.PurchaseStatePurchased
	case "pending", "purchase_state_pending", "payment_pending":
		return model.PurchaseStatePending
	case "canceled", "cancelled", "subscription_state_canceled":
		return model.PurchaseStateCanceled
	case "expired", "subscription_state_expired":
		return model.PurchaseStateExpired
	case "revoked", "subscription_state_revoked":
		return model.PurchaseStateRevoked
	case "refunded", "refund":
		return model.PurchaseStateRefunded
	default:
		if expireAt > 0 && expireAt <= uint64(c.nowFunc().Unix()) {
			return model.PurchaseStateExpired
		}
		return model.PurchaseStatePurchased
	}
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	case float64:
		return fmt.Sprintf("%.0f", t)
	case float32:
		return fmt.Sprintf("%.0f", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case int32:
		return fmt.Sprintf("%d", t)
	case uint64:
		return fmt.Sprintf("%d", t)
	case uint32:
		return fmt.Sprintf("%d", t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
}

func parseBoolLoose(v string) bool {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case "1", "true", "yes", "enabled", "on", "acknowledged":
		return true
	default:
		return false
	}
}

func parseUint64String(v string) uint64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	var n uint64
	for _, ch := range v {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + uint64(ch-'0')
	}
	return n
}

func parseUnixMillisString(v string) uint64 {
	ms := parseUint64String(v)
	if ms == 0 {
		return 0
	}
	return ms / 1000
}

func parseRFC3339ToUnix(v string) uint64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return 0
	}
	return uint64(t.Unix())
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func firstPositiveUint64(values ...uint64) uint64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
