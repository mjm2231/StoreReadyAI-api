package service

import (
	"context"
	"strings"
	"time"

	"storeready_ai/internal/client/modules/billing/dto"
	"storeready_ai/internal/client/modules/billing/model"
	errx "storeready_ai/internal/pkg/errors"
)

// GoogleVerifier Google Play 校验器接口。
//
// 说明：
// 1. 这里先定义一个更贴近 Google 平台语义的查询端口；
// 2. service 层通过 googleVerifierImpl 做统一适配，再转成 PlatformVerifyResult；
// 3. 后续接入 Google Play Developer API 时，只需要实现 client 即可。
type GoogleVerifier interface {
	VerifySubscription(ctx context.Context, req dto.GoogleVerifySubscriptionReq) (*dto.GoogleSubscriptionPurchase, error)
}

// googleVerifierImpl Google Play 平台校验实现。
//
// 当前职责：
// 1. 做参数规范化与基础校验；
// 2. 调用底层 GoogleVerifier client；
// 3. 转成 BillingService 统一使用的 PlatformVerifyResult。
type googleVerifierImpl struct {
	client  GoogleVerifier
	nowFunc func() time.Time
}

// NewGooglePlatformVerifier 创建 Google 平台校验器。
func NewGooglePlatformVerifier(client GoogleVerifier) PlatformVerifier {
	return &googleVerifierImpl{
		client:  client,
		nowFunc: time.Now,
	}
}

func (g *googleVerifierImpl) VerifyPurchase(ctx context.Context, req PlatformVerifyReq) (*PlatformVerifyResult, error) {
	if g == nil || g.client == nil {
		return nil, errx.New(errx.CodeInternal, "google verifier not configured")
	}

	platform := strings.TrimSpace(strings.ToLower(req.Platform))
	if platform != model.PlatformAndroid {
		return nil, errx.New(errx.CodeInvalidParam, "google verifier only supports android platform")
	}

	productID := strings.TrimSpace(req.ProductID)
	purchaseToken := strings.TrimSpace(req.PurchaseToken)
	if productID == "" {
		return nil, errx.New(errx.CodeInvalidParam, "product_id required")
	}
	if purchaseToken == "" {
		return nil, errx.New(errx.CodeInvalidParam, "purchase_token required")
	}

	googleResp, err := g.client.VerifySubscription(ctx, dto.GoogleVerifySubscriptionReq{
		ProductID:      productID,
		SubscriptionID: productID,
		OrderID:        strings.TrimSpace(req.OrderID),
		PurchaseToken:  purchaseToken,
		ReceiptData:    req.ReceiptData,
		PurchaseTime:   req.PurchaseTime,
	})
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "google verify subscription failed")
	}
	if googleResp == nil {
		return nil, errx.New(errx.CodeInternal, "google verify subscription returned nil")
	}

	result := &PlatformVerifyResult{
		Platform:        model.PlatformAndroid,
		ProductID:       firstNonEmpty(googleResp.ProductID, productID),
		SubscriptionID:  firstNonEmpty(googleResp.SubscriptionID, productID),
		BasePlanID:      firstNonEmpty(googleResp.BasePlanID),
		OrderID:         firstNonEmpty(googleResp.OrderID, strings.TrimSpace(req.OrderID)),
		OriginalOrderID: firstNonEmpty(googleResp.OriginalOrderID, googleResp.OrderID, strings.TrimSpace(req.OrderID)),
		PurchaseToken:   firstNonEmpty(googleResp.PurchaseToken, purchaseToken),
		PurchaseState:   normalizeGooglePurchaseState(googleResp.PurchaseState, googleResp.ExpireTime, uint64(g.nowFunc().Unix())),
		Acknowledged:    googleResp.Acknowledged,
		AutoRenewing:    googleResp.AutoRenewing,
		PurchaseTime:    pickUint64Value(googleResp.PurchaseTime, req.PurchaseTime),
		ExpireTime:      googleResp.ExpireTime,
		Currency:        strings.TrimSpace(googleResp.Currency),
		AmountMicros:    googleResp.AmountMicros,
		ReceiptData:     firstNonEmpty(googleResp.ReceiptData, req.ReceiptData),
		RawPayload:      googleResp.RawPayload,
		ProductCode:     strings.TrimSpace(googleResp.ProductCode),
	}

	return result, nil
}

func normalizeGooglePurchaseState(state string, expireAt, now uint64) string {
	state = strings.TrimSpace(strings.ToLower(state))
	switch state {
	case model.PurchaseStatePurchased,
		model.PurchaseStatePending,
		model.PurchaseStateCanceled,
		model.PurchaseStateRefunded,
		model.PurchaseStateExpired,
		model.PurchaseStateRevoked:
		return state
	}

	// Google 平台常见的外部状态收口。
	switch state {
	case "active", "success", "confirmed":
		if expireAt > 0 && expireAt <= now {
			return model.PurchaseStateExpired
		}
		return model.PurchaseStatePurchased
	case "pending_purchase", "payment_pending":
		return model.PurchaseStatePending
	case "cancelled", "canceled_by_user":
		return model.PurchaseStateCanceled
	case "refunded", "refund":
		return model.PurchaseStateRefunded
	case "revoked", "revoke":
		return model.PurchaseStateRevoked
	}

	if expireAt > 0 && expireAt <= now {
		return model.PurchaseStateExpired
	}
	return model.PurchaseStatePurchased
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

func pickUint64Value(values ...uint64) uint64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}
