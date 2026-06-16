package repo

import (
	"context"

	"gorm.io/gorm"

	"storeready_ai/internal/client/modules/billing/model"
)

// BillingOrderRepo 订单当前态仓储接口。
//
// 职责：
// 1. 以 purchase_token 为核心做幂等查询；
// 2. 维护订单当前快照；
// 3. 为 verify / restore / webhook 提供订单读写能力。
type BillingOrderRepo interface {
	Create(ctx context.Context, order *model.BillingOrder) error
	Update(ctx context.Context, order *model.BillingOrder) error
	Save(ctx context.Context, order *model.BillingOrder) error
	GetByID(ctx context.Context, id uint64) (*model.BillingOrder, error)
	GetByPlatformAndPurchaseToken(ctx context.Context, platform, purchaseToken string) (*model.BillingOrder, error)
	GetByPlatformAndOrderID(ctx context.Context, platform, orderID string) (*model.BillingOrder, error)
	ListByUserID(ctx context.Context, tenantID, userID uint64, limit, offset int) ([]*model.BillingOrder, error)
}

// BillingTransactionRepo 交易流水仓储接口。
//
// 职责：
// 1. 记录 purchase / renew / refund / revoke / restore 历史交易；
// 2. 支持按订单号、purchase_token、用户查询历史流水。
type BillingTransactionRepo interface {
	Create(ctx context.Context, tx *model.BillingTransaction) error
	BatchCreate(ctx context.Context, rows []*model.BillingTransaction) error
	GetByID(ctx context.Context, id uint64) (*model.BillingTransaction, error)
	ListByPlatformAndOrderID(ctx context.Context, platform, orderID string, limit, offset int) ([]*model.BillingTransaction, error)
	ListByPlatformAndPurchaseToken(ctx context.Context, platform, purchaseToken string, limit, offset int) ([]*model.BillingTransaction, error)
	ListByUserID(ctx context.Context, tenantID, userID uint64, limit, offset int) ([]*model.BillingTransaction, error)
}

// BillingEventRepo 事件流水仓储接口。
//
// 职责：
// 1. 记录 verify / restore / RTDN / Apple notification 等事件；
// 2. 跟踪处理状态，便于补偿与排查。
type BillingEventRepo interface {
	Create(ctx context.Context, event *model.BillingEvent) error
	Update(ctx context.Context, event *model.BillingEvent) error
	GetByID(ctx context.Context, id uint64) (*model.BillingEvent, error)
	ListByStatus(ctx context.Context, status string, limit, offset int) ([]*model.BillingEvent, error)
	ListByPlatformAndOrderID(ctx context.Context, platform, orderID string, limit, offset int) ([]*model.BillingEvent, error)
	ListByPlatformAndPurchaseToken(ctx context.Context, platform, purchaseToken string, limit, offset int) ([]*model.BillingEvent, error)
}

// BillingProductRepo 商品配置仓储接口。
//
// 职责：
// 1. 维护业务商品编码与商店商品 ID 映射；
// 2. 支持启停、推荐商品、排序读取。
type BillingProductRepo interface {
	Create(ctx context.Context, product *model.BillingProduct) error
	Update(ctx context.Context, product *model.BillingProduct) error
	GetByID(ctx context.Context, id uint64) (*model.BillingProduct, error)
	GetByTenantCodeAndPlatform(ctx context.Context, tenantID uint64, productCode, platform string) (*model.BillingProduct, error)
	GetByPlatformAndStoreProductID(ctx context.Context, platform, storeProductID string) (*model.BillingProduct, error)
	ListEnabledByPlatform(ctx context.Context, tenantID uint64, platform string) ([]*model.BillingProduct, error)
}

// Repos Billing 模块仓储集合，便于 service 层统一注入。
type Repos struct {
	Orders       BillingOrderRepo
	Transactions BillingTransactionRepo
	Events       BillingEventRepo
	Products     BillingProductRepo
}

type billingOrderRepo struct {
	db *gorm.DB
}

type billingTransactionRepo struct {
	db *gorm.DB
}

type billingEventRepo struct {
	db *gorm.DB
}

type billingProductRepo struct {
	db *gorm.DB
}

// NewBillingOrderRepo 创建订单仓储实现。
func NewBillingOrderRepo(db *gorm.DB) BillingOrderRepo {
	return &billingOrderRepo{db: db}
}

// NewBillingTransactionRepo 创建交易流水仓储实现。
func NewBillingTransactionRepo(db *gorm.DB) BillingTransactionRepo {
	return &billingTransactionRepo{db: db}
}

// NewBillingEventRepo 创建事件流水仓储实现。
func NewBillingEventRepo(db *gorm.DB) BillingEventRepo {
	return &billingEventRepo{db: db}
}

// NewBillingProductRepo 创建商品配置仓储实现。
func NewBillingProductRepo(db *gorm.DB) BillingProductRepo {
	return &billingProductRepo{db: db}
}

// NewRepos 创建 Billing 仓储集合。
func NewRepos(db *gorm.DB) *Repos {
	return &Repos{
		Orders:       NewBillingOrderRepo(db),
		Transactions: NewBillingTransactionRepo(db),
		Events:       NewBillingEventRepo(db),
		Products:     NewBillingProductRepo(db),
	}
}

func (r *billingOrderRepo) Create(ctx context.Context, order *model.BillingOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *billingOrderRepo) Update(ctx context.Context, order *model.BillingOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *billingOrderRepo) Save(ctx context.Context, order *model.BillingOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}

func (r *billingOrderRepo) GetByID(ctx context.Context, id uint64) (*model.BillingOrder, error) {
	var row model.BillingOrder
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingOrderRepo) GetByPlatformAndPurchaseToken(ctx context.Context, platform, purchaseToken string) (*model.BillingOrder, error) {
	var row model.BillingOrder
	if err := r.db.WithContext(ctx).
		Where("platform = ? AND purchase_token = ?", platform, purchaseToken).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingOrderRepo) GetByPlatformAndOrderID(ctx context.Context, platform, orderID string) (*model.BillingOrder, error) {
	var row model.BillingOrder
	if err := r.db.WithContext(ctx).
		Where("platform = ? AND order_id = ?", platform, orderID).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingOrderRepo) ListByUserID(ctx context.Context, tenantID, userID uint64, limit, offset int) ([]*model.BillingOrder, error) {
	var rows []*model.BillingOrder
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *billingTransactionRepo) Create(ctx context.Context, tx *model.BillingTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *billingTransactionRepo) BatchCreate(ctx context.Context, rows []*model.BillingTransaction) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *billingTransactionRepo) GetByID(ctx context.Context, id uint64) (*model.BillingTransaction, error) {
	var row model.BillingTransaction
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingTransactionRepo) ListByPlatformAndOrderID(ctx context.Context, platform, orderID string, limit, offset int) ([]*model.BillingTransaction, error) {
	var rows []*model.BillingTransaction
	query := r.db.WithContext(ctx).
		Where("platform = ? AND order_id = ?", platform, orderID).
		Order("transaction_time DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *billingTransactionRepo) ListByPlatformAndPurchaseToken(ctx context.Context, platform, purchaseToken string, limit, offset int) ([]*model.BillingTransaction, error) {
	var rows []*model.BillingTransaction
	query := r.db.WithContext(ctx).
		Where("platform = ? AND purchase_token = ?", platform, purchaseToken).
		Order("transaction_time DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *billingTransactionRepo) ListByUserID(ctx context.Context, tenantID, userID uint64, limit, offset int) ([]*model.BillingTransaction, error) {
	var rows []*model.BillingTransaction
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("transaction_time DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *billingEventRepo) Create(ctx context.Context, event *model.BillingEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *billingEventRepo) Update(ctx context.Context, event *model.BillingEvent) error {
	return r.db.WithContext(ctx).Save(event).Error
}

func (r *billingEventRepo) GetByID(ctx context.Context, id uint64) (*model.BillingEvent, error) {
	var row model.BillingEvent
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingEventRepo) ListByStatus(ctx context.Context, status string, limit, offset int) ([]*model.BillingEvent, error) {
	var rows []*model.BillingEvent
	query := r.db.WithContext(ctx).
		Where("event_status = ?", status).
		Order("updated_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *billingEventRepo) ListByPlatformAndOrderID(ctx context.Context, platform, orderID string, limit, offset int) ([]*model.BillingEvent, error) {
	var rows []*model.BillingEvent
	query := r.db.WithContext(ctx).
		Where("platform = ? AND order_id = ?", platform, orderID).
		Order("event_time DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *billingEventRepo) ListByPlatformAndPurchaseToken(ctx context.Context, platform, purchaseToken string, limit, offset int) ([]*model.BillingEvent, error) {
	var rows []*model.BillingEvent
	query := r.db.WithContext(ctx).
		Where("platform = ? AND purchase_token = ?", platform, purchaseToken).
		Order("event_time DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *billingProductRepo) Create(ctx context.Context, product *model.BillingProduct) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *billingProductRepo) Update(ctx context.Context, product *model.BillingProduct) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *billingProductRepo) GetByID(ctx context.Context, id uint64) (*model.BillingProduct, error) {
	var row model.BillingProduct
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingProductRepo) GetByTenantCodeAndPlatform(ctx context.Context, tenantID uint64, productCode, platform string) (*model.BillingProduct, error) {
	var row model.BillingProduct
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND product_code = ? AND platform = ?", tenantID, productCode, platform).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingProductRepo) GetByPlatformAndStoreProductID(ctx context.Context, platform, storeProductID string) (*model.BillingProduct, error) {
	var row model.BillingProduct
	if err := r.db.WithContext(ctx).
		Where("platform = ? AND store_product_id = ?", platform, storeProductID).
		Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *billingProductRepo) ListEnabledByPlatform(ctx context.Context, tenantID uint64, platform string) ([]*model.BillingProduct, error) {
	var rows []*model.BillingProduct
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND platform = ? AND status = ?", tenantID, platform, model.BillingProductStatusEnabled).
		Order("sort ASC, id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
