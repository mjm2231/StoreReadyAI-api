package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	androidpublisher "google.golang.org/api/androidpublisher/v3"
	"google.golang.org/api/option"

	errx "storeready_ai/internal/pkg/errors"
)

// GooglePlayExecutorConfig Google Play 执行器配置。
//
// 说明：
// 1. CredentialsFile 为 Google service account json 文件路径；
// 2. Scopes 为空时，默认使用 Android Publisher scope；
// 3. 这里不绑定 packageName，因为真正查询时会优先读取请求中的 PackageName。
type GooglePlayExecutorConfig struct {
	CredentialsFile string
	Scopes          []string
}

// NewGooglePlaySubscriptionExecutor 创建真实的 Google Play 订阅查询执行器。
//
// 当前职责：
// 1. 使用 service account 初始化 Android Publisher client；
// 2. 调 Google Play Developer API 查询 subscription purchase；
// 3. 返回 map[string]any，供上层 client 再做统一 DTO 映射。
func NewGooglePlaySubscriptionExecutor(cfg GooglePlayExecutorConfig) (GooglePlaySubscriptionExecutor, error) {
	credentialsFile := strings.TrimSpace(cfg.CredentialsFile)
	if credentialsFile == "" {
		return nil, errx.New(errx.CodeInvalidParam, "google play credentials file required")
	}
	if _, err := os.Stat(credentialsFile); err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "google play credentials file not found")
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{androidpublisher.AndroidpublisherScope}
	}

	exec := &googlePlaySubscriptionExecutor{
		credentialsFile: credentialsFile,
		scopes:          scopes,
	}
	return exec.Execute, nil
}

type googlePlaySubscriptionExecutor struct {
	credentialsFile string
	scopes          []string

	mu      sync.Mutex
	service *androidpublisher.Service
}

func (e *googlePlaySubscriptionExecutor) Execute(ctx context.Context, req GooglePlaySubscriptionQuery) (map[string]any, error) {
	if e == nil {
		return nil, errx.New(errx.CodeInternal, "google play executor is nil")
	}

	packageName := strings.TrimSpace(req.PackageName)
	purchaseToken := strings.TrimSpace(req.PurchaseToken)
	if packageName == "" {
		return nil, errx.New(errx.CodeInvalidParam, "google play package name required")
	}
	if purchaseToken == "" {
		return nil, errx.New(errx.CodeInvalidParam, "google play purchase token required")
	}

	svc, err := e.getService(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := svc.Purchases.Subscriptionsv2.Get(packageName, purchaseToken).Context(ctx).Do()
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "google play subscriptionsv2 get failed")
	}
	if resp == nil {
		return nil, errx.New(errx.CodeInternal, "google play subscriptionsv2 response is nil")
	}

	payload, err := structToMap(resp)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "google play response marshal failed")
	}

	// 补齐上层 client 常用的平铺字段，避免其重复解析嵌套结构。
	lineItem := firstLineItem(payload)
	if _, ok := payload["productId"]; !ok {
		payload["productId"] = firstString(
			stringValue(lineItem, "productId"),
			stringValue(lineItem, "product_id"),
			strings.TrimSpace(req.ProductID),
			strings.TrimSpace(req.SubscriptionID),
		)
	}
	if _, ok := payload["subscriptionId"]; !ok {
		payload["subscriptionId"] = firstString(
			stringValue(payload, "productId"),
			strings.TrimSpace(req.SubscriptionID),
			strings.TrimSpace(req.ProductID),
		)
	}
	if _, ok := payload["basePlanId"]; !ok {
		payload["basePlanId"] = firstString(
			stringValue(lineItem, "basePlanId"),
			stringValue(nestedMap(lineItem, "offerDetails"), "basePlanId"),
			strings.TrimSpace(req.BasePlanID),
		)
	}
	if _, ok := payload["latestOrderId"]; !ok && strings.TrimSpace(req.OrderID) != "" {
		payload["latestOrderId"] = strings.TrimSpace(req.OrderID)
	}
	if _, ok := payload["purchaseToken"]; !ok {
		payload["purchaseToken"] = purchaseToken
	}
	if _, ok := payload["lineItems.productId"]; !ok {
		payload["lineItems.productId"] = stringValue(payload, "productId")
	}
	if _, ok := payload["lineItems.basePlanId"]; !ok {
		payload["lineItems.basePlanId"] = stringValue(payload, "basePlanId")
	}
	if _, ok := payload["startTimeMillis"]; !ok {
		payload["startTimeMillis"] = firstString(
			stringValue(lineItem, "startTime"),
			stringValue(lineItem, "startTimeMillis"),
		)
	}
	if _, ok := payload["expiryTimeMillis"]; !ok {
		payload["expiryTimeMillis"] = firstString(
			stringValue(lineItem, "expiryTime"),
			stringValue(lineItem, "expiryTimeMillis"),
		)
	}
	if _, ok := payload["priceAmountMicros"]; !ok {
		payload["priceAmountMicros"] = firstString(
			stringValue(lineItem, "priceAmountMicros"),
			stringValue(nestedMap(lineItem, "pricingPhase"), "priceAmountMicros"),
		)
	}
	if _, ok := payload["priceCurrencyCode"]; !ok {
		payload["priceCurrencyCode"] = firstString(
			stringValue(lineItem, "priceCurrencyCode"),
			stringValue(nestedMap(lineItem, "pricingPhase"), "priceCurrencyCode"),
		)
	}
	if _, ok := payload["acknowledged"]; !ok {
		payload["acknowledged"] = normalizeAcknowledgedValue(payload)
	}
	if _, ok := payload["autoRenewing"]; !ok {
		payload["autoRenewing"] = normalizeAutoRenewingValue(lineItem)
	}

	return payload, nil
}

func (e *googlePlaySubscriptionExecutor) getService(ctx context.Context) (*androidpublisher.Service, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.service != nil {
		return e.service, nil
	}

	svc, err := androidpublisher.NewService(
		ctx,
		option.WithCredentialsFile(e.credentialsFile),
		option.WithScopes(e.scopes...),
	)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create android publisher service failed")
	}
	e.service = svc
	return e.service, nil
}

func structToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func firstLineItem(payload map[string]any) map[string]any {
	items, ok := payload["lineItems"].([]any)
	if !ok || len(items) == 0 {
		return nil
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		return nil
	}
	return first
}

func nestedMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	out, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return out
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return fmt.Sprintf("%.0f", t)
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
		return strings.Trim(strings.TrimSpace(string(b)), `"`)
	}
}

func firstString(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func normalizeAcknowledgedValue(payload map[string]any) string {
	state := strings.TrimSpace(strings.ToLower(stringValue(payload, "acknowledgementState")))
	switch state {
	case "acknowledgement_state_acknowledged", "acknowledged", "1", "true":
		return "true"
	default:
		return "false"
	}
}

func normalizeAutoRenewingValue(lineItem map[string]any) string {
	autoRenewingPlan := nestedMap(lineItem, "autoRenewingPlan")
	v := strings.TrimSpace(strings.ToLower(firstString(
		stringValue(autoRenewingPlan, "autoRenewEnabled"),
		stringValue(lineItem, "autoRenewing"),
	)))
	switch v {
	case "1", "true", "enabled", "on":
		return "true"
	default:
		return "false"
	}
}
