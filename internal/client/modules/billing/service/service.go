package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"storeready_ai/internal/client/modules/billing/dto"
	"storeready_ai/internal/client/modules/billing/model"
	"storeready_ai/internal/client/modules/billing/repo"
	contractent "storeready_ai/internal/contracts/entitlement"
	errx "storeready_ai/internal/pkg/errors"
)

// PlatformVerifier 平台校验器。
//
// 说明：
// 1. Google / Apple 的校验细节放在平台实现内；
// 2. service 层只关心统一后的校验结果。
type PlatformVerifier interface {
	VerifyPurchase(ctx context.Context, req PlatformVerifyReq) (*PlatformVerifyResult, error)
}

// PlatformVerifyReq 平台校验入参。
type PlatformVerifyReq struct {
	Platform      string
	ProductID     string
	OrderID       string
	PurchaseToken string
	ReceiptData   string
	PurchaseTime  uint64
	Source        string
}

// PlatformVerifyResult 平台统一校验结果。
type PlatformVerifyResult struct {
	Platform        string
	ProductID       string
	SubscriptionID  string
	BasePlanID      string
	OrderID         string
	OriginalOrderID string
	PurchaseToken   string
	PurchaseState   string
	Acknowledged    bool
	AutoRenewing    bool
	PurchaseTime    uint64
	ExpireTime      uint64
	Currency        string
	AmountMicros    uint64
	ReceiptData     string
	RawPayload      string
	ProductCode     string
}

// Service Billing 服务接口。
type Service interface {
	VerifyPurchase(ctx context.Context, tenantID, userID, uid uint64, req dto.VerifyPurchaseReq) (*dto.VerifyPurchaseResp, error)
	RestorePurchase(ctx context.Context, tenantID, userID, uid uint64, req dto.RestorePurchaseReq) (*dto.RestorePurchaseResp, error)
	GetEntitlement(ctx context.Context, tenantID, uid uint64) (*dto.EntitlementResp, error)
	GetConfig(ctx context.Context, tenantID uint64, platform string) (*dto.BillingConfigResp, error)
}

type serviceImpl struct {
	repos       *repo.Repos
	entitlement contractent.Service
	verifier    PlatformVerifier
	nowFunc     func() time.Time
	defaultCode string
}

// New 创建 Billing 服务。
func New(repos *repo.Repos, entitlement contractent.Service, verifier PlatformVerifier, defaultEntitlementCode string) Service {
	defaultEntitlementCode = strings.TrimSpace(defaultEntitlementCode)
	if defaultEntitlementCode == "" {
		defaultEntitlementCode = "vip"
	}
	return &serviceImpl{
		repos:       repos,
		entitlement: entitlement,
		verifier:    verifier,
		nowFunc:     time.Now,
		defaultCode: defaultEntitlementCode,
	}
}

func (s *serviceImpl) VerifyPurchase(
	ctx context.Context,
	tenantID, userID, uid uint64,
	req dto.VerifyPurchaseReq,
) (*dto.VerifyPurchaseResp, error) {
	if err := s.validateVerifyReq(req); err != nil {
		return nil, err
	}
	if s.verifier == nil {
		return nil, errx.New(errx.CodeInternal, "billing verifier not configured")
	}
	if s.entitlement == nil {
		return nil, errx.New(errx.CodeInternal, "entitlement service not configured")
	}
	if s.repos == nil || s.repos.Orders == nil || s.repos.Transactions == nil || s.repos.Events == nil {
		return nil, errx.New(errx.CodeInternal, "billing repos not configured")
	}

	now := uint64(s.nowFunc().Unix())
	platform := strings.TrimSpace(strings.ToLower(req.Platform))

	verifyResult, err := s.verifier.VerifyPurchase(ctx, PlatformVerifyReq{
		Platform:      platform,
		ProductID:     strings.TrimSpace(req.ProductID),
		OrderID:       strings.TrimSpace(req.OrderID),
		PurchaseToken: strings.TrimSpace(req.PurchaseToken),
		ReceiptData:   req.ReceiptData,
		PurchaseTime:  req.PurchaseTime,
		Source:        strings.TrimSpace(req.Source),
	})
	if err != nil {
		event := s.newVerifyEvent(tenantID, userID, uid, req, model.EventStatusFailed, now, err)
		_ = s.repos.Events.Create(ctx, event)
		return nil, s.wrapVerifyErr(err)
	}

	// 先记事件流水，便于审计与排查。
	event := s.newVerifySuccessEvent(tenantID, userID, uid, req, verifyResult, now)
	if err := s.repos.Events.Create(ctx, event); err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create billing event failed")
	}

	order, err := s.upsertOrderFromVerify(ctx, tenantID, userID, uid, req, verifyResult, now)
	if err != nil {
		return nil, err
	}

	txRow := s.newTransactionFromVerify(tenantID, userID, uid, verifyResult, now)
	if err := s.repos.Transactions.Create(ctx, txRow); err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create billing transaction failed")
	}

	entitlementResp, err := s.entitlement.RefreshByBilling(ctx, contractent.RefreshByBillingReq{
		TenantID:        tenantID,
		UserID:          userID,
		UID:             uid,
		EntitlementCode: s.defaultCode,
		ProductCode:     strings.TrimSpace(verifyResult.ProductCode),
		ProductID:       strings.TrimSpace(verifyResult.ProductID),
		Platform:        strings.TrimSpace(verifyResult.Platform),
		Status:          s.mapEntitlementStatus(verifyResult.PurchaseState, verifyResult.ExpireTime, now),
		AutoRenew:       verifyResult.AutoRenewing,
		ExpiresAt:       verifyResult.ExpireTime,
		OriginalOrderID: verifyResult.OriginalOrderID,
		PurchaseToken:   verifyResult.PurchaseToken,
		Source:          model.EventSourceClient,
	})
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "refresh entitlement failed")
	}

	return &dto.VerifyPurchaseResp{
		Order:       toBillingOrderVO(order),
		Entitlement: toEntitlementResp(entitlementResp),
	}, nil
}

func (s *serviceImpl) RestorePurchase(
	ctx context.Context,
	tenantID, userID, uid uint64,
	req dto.RestorePurchaseReq,
) (*dto.RestorePurchaseResp, error) {
	verifyResp, err := s.VerifyPurchase(ctx, tenantID, userID, uid, dto.VerifyPurchaseReq{
		Platform:      req.Platform,
		ProductID:     req.ProductID,
		OrderID:       req.OrderID,
		PurchaseToken: req.PurchaseToken,
		ReceiptData:   req.ReceiptData,
		Source:        s.normalizeSource(req.Source, model.EventSourceClient),
	})
	if err != nil {
		return nil, err
	}

	return &dto.RestorePurchaseResp{
		Order:       verifyResp.Order,
		Entitlement: verifyResp.Entitlement,
	}, nil
}

func (s *serviceImpl) GetEntitlement(ctx context.Context, tenantID, uid uint64) (*dto.EntitlementResp, error) {
	if s.entitlement == nil {
		return nil, errx.New(errx.CodeInternal, "entitlement service not configured")
	}
	resp, err := s.entitlement.GetCurrent(ctx, tenantID, uid, s.defaultCode)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "get entitlement failed")
	}
	return dtoPtrEntitlementResp(resp), nil
}

func (s *serviceImpl) GetConfig(ctx context.Context, tenantID uint64, platform string) (*dto.BillingConfigResp, error) {
	if s.repos == nil || s.repos.Products == nil {
		return nil, errx.New(errx.CodeInternal, "billing product repo not configured")
	}
	platform = strings.TrimSpace(strings.ToLower(platform))
	if platform == "" {
		return nil, errx.New(errx.CodeInvalidParam, "platform required")
	}

	products, err := s.repos.Products.ListEnabledByPlatform(ctx, tenantID, platform)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "list billing products failed")
	}

	resp := &dto.BillingConfigResp{
		Products: make([]dto.BillingProductVO, 0, len(products)),
	}
	for _, p := range products {
		resp.Products = append(resp.Products, toBillingProductVO(p))
		if p.IsRecommended && resp.RecommendedProductCode == "" {
			resp.RecommendedProductCode = p.ProductCode
		}
	}
	return resp, nil
}

func (s *serviceImpl) validateVerifyReq(req dto.VerifyPurchaseReq) error {
	platform := strings.TrimSpace(strings.ToLower(req.Platform))
	if platform == "" {
		return errx.New(errx.CodeInvalidParam, "platform required")
	}
	if platform != model.PlatformIOS && platform != model.PlatformAndroid {
		return errx.New(errx.CodeInvalidParam, "invalid platform")
	}
	if strings.TrimSpace(req.ProductID) == "" {
		return errx.New(errx.CodeInvalidParam, "product_id required")
	}
	if strings.TrimSpace(req.PurchaseToken) == "" {
		return errx.New(errx.CodeInvalidParam, "purchase_token required")
	}
	return nil
}

func (s *serviceImpl) upsertOrderFromVerify(
	ctx context.Context,
	tenantID, userID, uid uint64,
	req dto.VerifyPurchaseReq,
	verifyResult *PlatformVerifyResult,
	now uint64,
) (*model.BillingOrder, error) {
	order, err := s.repos.Orders.GetByPlatformAndPurchaseToken(ctx, verifyResult.Platform, verifyResult.PurchaseToken)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errx.Wrap(err, errx.CodeInternal, "query billing order failed")
	}

	if errors.Is(err, gorm.ErrRecordNotFound) || order == nil {
		order = &model.BillingOrder{
			TenantID:  tenantID,
			UserID:    userID,
			UID:       uid,
			Platform:  verifyResult.Platform,
			CreatedAt: now,
		}
	}

	order.TenantID = tenantID
	order.UserID = userID
	order.UID = uid
	order.Platform = verifyResult.Platform
	order.ProductID = verifyResult.ProductID
	order.SubscriptionID = verifyResult.SubscriptionID
	order.BasePlanID = verifyResult.BasePlanID
	order.OrderID = verifyResult.OrderID
	order.OriginalOrderID = verifyResult.OriginalOrderID
	order.PurchaseToken = verifyResult.PurchaseToken
	order.ReceiptData = s.pickNonEmpty(verifyResult.ReceiptData, req.ReceiptData)
	order.PurchaseState = verifyResult.PurchaseState
	order.Acknowledged = verifyResult.Acknowledged
	order.AutoRenewing = verifyResult.AutoRenewing
	order.PurchaseTime = verifyResult.PurchaseTime
	order.ExpireTime = verifyResult.ExpireTime
	order.Currency = verifyResult.Currency
	order.AmountMicros = verifyResult.AmountMicros
	order.VerifyStatus = model.VerifyStatusSuccess
	order.VerifyErrorCode = ""
	order.VerifyErrorMessage = ""
	order.LastVerifiedAt = now
	order.RawPayload = verifyResult.RawPayload
	order.UpdatedAt = now
	if order.CreatedAt == 0 {
		order.CreatedAt = now
	}

	if order.ID == 0 {
		if err := s.repos.Orders.Create(ctx, order); err != nil {
			return nil, errx.Wrap(err, errx.CodeInternal, "create billing order failed")
		}
	} else {
		if err := s.repos.Orders.Update(ctx, order); err != nil {
			return nil, errx.Wrap(err, errx.CodeInternal, "update billing order failed")
		}
	}
	return order, nil
}

func (s *serviceImpl) newTransactionFromVerify(
	tenantID, userID, uid uint64,
	verifyResult *PlatformVerifyResult,
	now uint64,
) *model.BillingTransaction {
	transactionType := model.TransactionTypePurchase
	if verifyResult.AutoRenewing && verifyResult.OriginalOrderID != "" && verifyResult.OrderID != "" && verifyResult.OrderID != verifyResult.OriginalOrderID {
		transactionType = model.TransactionTypeRenew
	}
	return &model.BillingTransaction{
		TenantID:         tenantID,
		UserID:           userID,
		UID:              uid,
		Platform:         verifyResult.Platform,
		ProductID:        verifyResult.ProductID,
		OrderID:          verifyResult.OrderID,
		OriginalOrderID:  verifyResult.OriginalOrderID,
		PurchaseToken:    verifyResult.PurchaseToken,
		TransactionType:  transactionType,
		TransactionState: model.TransactionStateSuccess,
		AmountMicros:     verifyResult.AmountMicros,
		Currency:         verifyResult.Currency,
		TransactionTime:  s.pickUint64(verifyResult.PurchaseTime, now),
		RawPayload:       verifyResult.RawPayload,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func (s *serviceImpl) newVerifyEvent(
	tenantID, userID, uid uint64,
	req dto.VerifyPurchaseReq,
	status string,
	now uint64,
	err error,
) *model.BillingEvent {
	errorCode := ""
	errorMessage := ""
	if err != nil {
		errorCode = "verify_failed"
		errorMessage = err.Error()
	}
	return &model.BillingEvent{
		TenantID:        tenantID,
		UserID:          userID,
		UID:             uid,
		Platform:        strings.TrimSpace(strings.ToLower(req.Platform)),
		EventType:       model.EventTypeVerify,
		EventSource:     s.normalizeSource(req.Source, model.EventSourceClient),
		OrderID:         strings.TrimSpace(req.OrderID),
		OriginalOrderID: "",
		PurchaseToken:   strings.TrimSpace(req.PurchaseToken),
		ProductID:       strings.TrimSpace(req.ProductID),
		EventStatus:     status,
		ErrorCode:       errorCode,
		ErrorMessage:    errorMessage,
		EventTime:       now,
		ProcessedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (s *serviceImpl) newVerifySuccessEvent(
	tenantID, userID, uid uint64,
	req dto.VerifyPurchaseReq,
	verifyResult *PlatformVerifyResult,
	now uint64,
) *model.BillingEvent {
	row := s.newVerifyEvent(tenantID, userID, uid, req, model.EventStatusProcessed, now, nil)
	row.Platform = verifyResult.Platform
	row.OrderID = verifyResult.OrderID
	row.OriginalOrderID = verifyResult.OriginalOrderID
	row.PurchaseToken = verifyResult.PurchaseToken
	row.ProductID = verifyResult.ProductID
	row.RawPayload = verifyResult.RawPayload
	return row
}

func (s *serviceImpl) wrapVerifyErr(err error) error {
	return errx.Wrap(err, errx.CodeInternal, "verify purchase failed")
}

func (s *serviceImpl) mapEntitlementStatus(purchaseState string, expireAt, now uint64) string {
	switch purchaseState {
	case model.PurchaseStateRefunded, model.PurchaseStateRevoked, model.PurchaseStateCanceled:
		return "revoked"
	case model.PurchaseStateExpired:
		return "expired"
	}
	if expireAt > 0 && expireAt <= now {
		return "expired"
	}
	return "active"
}

func (s *serviceImpl) normalizeSource(source, fallback string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	if source == "" {
		return fallback
	}
	return source
}

func (s *serviceImpl) pickNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func (s *serviceImpl) pickUint64(a, b uint64) uint64 {
	if a > 0 {
		return a
	}
	return b
}

func toEntitlementResp(row *contractent.CurrentEntitlement) dto.EntitlementResp {
	if row == nil {
		return dto.EntitlementResp{
			IsVIP:                  0,
			VIPStartedAt:           0,
			VIPExpiredAt:           0,
			FreeLimit:              contractent.DefaultFreeLimit,
			SyncEnabled:            false,
			UnlimitedSubscriptions: false,
			EntitlementCode:        sDefaultEntitlementCode(),
			ProductCode:            "",
			ProductID:              "",
			Platform:               "",
			Status:                 "expired",
			AutoRenew:              false,
			ExpiresAt:              0,
		}
	}

	return dto.EntitlementResp{
		// App 统一权益字段。
		IsVIP:                  row.Entitlement.IsVIP,
		VIPStartedAt:           row.Entitlement.VIPStartedAt,
		VIPExpiredAt:           row.Entitlement.VIPExpiredAt,
		FreeLimit:              row.Entitlement.FreeLimit,
		SyncEnabled:            row.Entitlement.SyncEnabled,
		UnlimitedSubscriptions: row.Entitlement.UnlimitedSubscriptions,

		// Billing 展示字段。
		EntitlementCode: row.EntitlementCode,
		ProductCode:     row.ProductCode,
		ProductID:       row.ProductID,
		Platform:        row.Platform,
		Status:          row.Status,
		AutoRenew:       row.AutoRenew,
		ExpiresAt:       row.ExpiresAt,
	}
}

func dtoPtrEntitlementResp(row *contractent.CurrentEntitlement) *dto.EntitlementResp {
	resp := toEntitlementResp(row)
	return &resp
}

func sDefaultEntitlementCode() string {
	return "vip"
}

func toBillingOrderVO(row *model.BillingOrder) dto.BillingOrderVO {
	if row == nil {
		return dto.BillingOrderVO{}
	}
	return dto.BillingOrderVO{
		ID:                 row.ID,
		Platform:           row.Platform,
		ProductID:          row.ProductID,
		SubscriptionID:     row.SubscriptionID,
		BasePlanID:         row.BasePlanID,
		OrderID:            row.OrderID,
		OriginalOrderID:    row.OriginalOrderID,
		PurchaseState:      row.PurchaseState,
		Acknowledged:       row.Acknowledged,
		AutoRenewing:       row.AutoRenewing,
		PurchaseTime:       row.PurchaseTime,
		ExpireTime:         row.ExpireTime,
		Currency:           row.Currency,
		AmountMicros:       row.AmountMicros,
		VerifyStatus:       row.VerifyStatus,
		VerifyErrorCode:    row.VerifyErrorCode,
		VerifyErrorMessage: row.VerifyErrorMessage,
		LastVerifiedAt:     row.LastVerifiedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func toBillingProductVO(row *model.BillingProduct) dto.BillingProductVO {
	if row == nil {
		return dto.BillingProductVO{}
	}
	return dto.BillingProductVO{
		ID:                row.ID,
		ProductCode:       row.ProductCode,
		Platform:          row.Platform,
		StoreProductID:    row.StoreProductID,
		ProductType:       row.ProductType,
		SubscriptionGroup: row.SubscriptionGroup,
		Status:            row.Status,
		IsRecommended:     row.IsRecommended,
		Sort:              row.Sort,
	}
}
