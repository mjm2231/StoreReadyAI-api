package repo

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"storeready_ai/internal/client/modules/user/model"
	"storeready_ai/internal/common"
	"storeready_ai/internal/contracts/stats"
	"storeready_ai/internal/contracts/user"
)

// UserRepo 用户领域的持久化接口（Repo层）。
// 说明：与传输层无关（HTTP / 后期 gRPC 复用同一套接口）。
// 约定：时间字段均为 Unix 秒（与表结构 BIGINT UNSIGNED 对齐）。
type UserRepo interface {
	// --- 用户查询 ---
	// GetUserByID 按内部自增主键查询用户。
	// GetUserByID returns a user by internal auto-increment id.
	GetUserByID(ctx context.Context, tenantID, id uint64) (*model.User, error)
	// GetUserByUID 按对外 UID 查询用户。
	// GetUserByUID returns a user by externally exposed uid.
	GetUserByUID(ctx context.Context, tenantID, uid uint64) (*model.User, error)

	GetUserByEmail(ctx context.Context, tenantID uint64, email string) (*model.User, error)
	// UpdatePasswordHash 更新账号密码登录的密码哈希。
	UpdatePasswordHash(ctx context.Context, tenantID, userID uint64, passwordHash string) error

	// --- 第三方身份（Firebase） ---
	// GetIdentity 按 provider + provider_uid 查询第三方身份绑定。
	// GetIdentity finds identity by provider + provider_uid.
	GetIdentity(ctx context.Context, tenantID uint64, provider, providerUID string) (*model.UserIdentity, error)
	// GetUserByIdentity 通过身份绑定加载用户信息。
	// GetUserByIdentity loads user via identity.
	GetUserByIdentity(ctx context.Context, tenantID uint64, provider, providerUID string) (*model.User, *model.UserIdentity, error)

	// --- 登录核心：用户 + 身份 upsert（事务） ---
	// UpsertUserWithIdentity 登录核心：按身份查找用户，不存在则创建（事务保证一致性）。
	// UpsertUserWithIdentity creates or updates a user+identity in a single transaction.
	// created indicates whether a new user record was created.
	UpsertUserWithIdentity(
		ctx context.Context,
		tenantID uint64,
		provider string,
		providerUID string,
		email *string,
		name *string,
		avatar *string,
		rawProfile datatypes.JSON,
	) (u *model.User, it *model.UserIdentity, created bool, err error)

	// CreatePasswordUserWithIdentity 创建账号密码用户和 password identity（事务）。
	CreatePasswordUserWithIdentity(
		ctx context.Context,
		tenantID uint64,
		email string,
		passwordHash string,
		name *string,
	) (*model.User, *model.UserIdentity, error)

	// --- Refresh Token（多端/续期） ---
	// CreateRefreshToken 创建一条 refresh token 记录（只存 hash）。
	// CreateRefreshToken creates a refresh token record (only hash is stored).
	CreateRefreshToken(
		ctx context.Context,
		tenantID, userID uint64,
		tokenHash string,
		deviceID, deviceName, ip, userAgent *string,
		expiredAt uint64,
	) (*model.UserRefreshToken, error)
	// GetRefreshTokenByHash 按 tenant_id + token_hash 查询 refresh token。
	// GetRefreshTokenByHash queries refresh token by tenant_id + token_hash.
	GetRefreshTokenByHash(ctx context.Context, tenantID uint64, tokenHash string) (*model.UserRefreshToken, error)
	// TouchRefreshToken 刷新 refresh token 的最近使用时间。
	// TouchRefreshToken updates refresh token's last used time.
	TouchRefreshToken(ctx context.Context, tokenHash string) error
	// RevokeRefreshToken 吊销单个 refresh token。
	// RevokeRefreshToken revokes a single refresh token.
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	// RevokeAllRefreshTokensByUser 吊销用户的全部有效 refresh token。
	// RevokeAllRefreshTokensByUser revokes all valid refresh tokens for a user.
	RevokeAllRefreshTokensByUser(ctx context.Context, tenantID, userID uint64) error

	// --- 统计能力（供 admin stats/service 聚合调用） ---
	// CountUsers 统计用户总数。
	CountUsers(ctx context.Context, filter stats.UserFilter) (int64, error)
	CountNewUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
	CountActiveUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
	CountReturnUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
	CountLoginEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
	CountLoginMethodEvent(ctx context.Context, tenantID uint64, method, eventName, startDate, endDate string) (int64, error)
	CountVipEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
	// GetUserCreatedTrend 获取用户新增趋势。
	GetUserCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]stats.TrendPoint, error)

	// ListUsers 列表查询用户（分页、过滤）。
	ListUsers(ctx context.Context, tenantID uint64, filter user.QueryUserFilter) ([]*model.User, uint64, error)
	// UpdateUser 更新用户信息（可选：后续用到再加）。
	UpdateUser(ctx context.Context, tenantID, userID uint64, req user.UpdateUserReq) error
}

// gormRepo UserRepo 的 GORM 实现（对外隐藏具体类型，只暴露接口）。
type gormRepo struct {
	db *gorm.DB
}

// New 创建 Repo（返回接口，便于 mock / 替换实现）。
func New(db *gorm.DB) UserRepo {
	return &gormRepo{db: db}
}

// nowSec 当前时间（Unix 秒）。
func nowSec() uint64 { return uint64(time.Now().Unix()) }

// GetUserByID 按内部自增主键查询用户。
// GetUserByID returns a user by internal auto-increment id.
func (r *gormRepo) GetUserByID(ctx context.Context, tenantID, id uint64) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, id).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByUID 按对外 UID 查询用户。
// GetUserByUID returns a user by externally exposed uid.
func (r *gormRepo) GetUserByUID(ctx context.Context, tenantID, uid uint64) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND uid = ?", tenantID, uid).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByEmail 按 email 查询用户。
func (r *gormRepo) GetUserByEmail(ctx context.Context, tenantID uint64, email string) (*model.User, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, errors.New("email required")
	}

	var u model.User
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ?", tenantID, email).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdatePasswordHash 更新账号密码登录的密码哈希。
func (r *gormRepo) UpdatePasswordHash(ctx context.Context, tenantID, userID uint64, passwordHash string) error {
	passwordHash = strings.TrimSpace(passwordHash)
	if passwordHash == "" {
		return errors.New("password_hash required")
	}

	now := nowSec()
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("tenant_id = ? AND id = ?", tenantID, userID).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"updated_at":    now,
		}).Error
}

// GetIdentity 按 provider + provider_uid 查询第三方身份绑定。
// GetIdentity finds identity by provider + provider_uid.
func (r *gormRepo) GetIdentity(ctx context.Context, tenantID uint64, provider, providerUID string) (*model.UserIdentity, error) {
	provider = strings.TrimSpace(provider)
	providerUID = strings.TrimSpace(providerUID)
	if provider == "" || providerUID == "" {
		return nil, errors.New("provider/provider_uid required")
	}
	if provider == model.IdentityProviderPassword {
		providerUID = normalizeEmail(providerUID)
	}
	var it model.UserIdentity
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND provider = ? AND provider_uid = ?", tenantID, provider, providerUID).
		First(&it).Error
	if err != nil {
		return nil, err
	}
	return &it, nil
}

// GetUserByIdentity 通过身份绑定加载用户信息。
// GetUserByIdentity loads user via identity.
func (r *gormRepo) GetUserByIdentity(ctx context.Context, tenantID uint64, provider, providerUID string) (*model.User, *model.UserIdentity, error) {
	it, err := r.GetIdentity(ctx, tenantID, provider, providerUID)
	if err != nil {
		return nil, nil, err
	}
	u, err := r.GetUserByID(ctx, tenantID, it.UserID)
	if err != nil {
		return nil, nil, err
	}
	return u, it, nil
}

// UpsertUserWithIdentity 登录核心：按身份查找用户，不存在则创建（事务保证一致性）。
// UpsertUserWithIdentity creates or updates a user+identity in a single transaction.
// created indicates whether a new user record was created.
func (r *gormRepo) UpsertUserWithIdentity(
	ctx context.Context,
	tenantID uint64,
	provider string,
	providerUID string,
	email *string,
	name *string,
	avatar *string,
	rawProfile datatypes.JSON,
) (u *model.User, it *model.UserIdentity, created bool, err error) {
	provider = strings.TrimSpace(provider)
	providerUID = strings.TrimSpace(providerUID)
	if provider == "" || providerUID == "" {
		return nil, nil, false, errors.New("provider/provider_uid required")
	}
	if provider == model.IdentityProviderPassword {
		providerUID = normalizeEmail(providerUID)
	}
	if email != nil {
		normalized := normalizeEmail(*email)
		if normalized == "" {
			email = nil
		} else {
			email = &normalized
		}
	}

	now := nowSec()

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock identity row if exists (avoid concurrent double-creates)
		var existing model.UserIdentity
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND provider = ? AND provider_uid = ?", tenantID, provider, providerUID).
			First(&existing).Error
		if err == nil {
			// Identity exists -> load user and update login time
			var user model.User
			if err := tx.Where("tenant_id = ? AND id = ?", tenantID, existing.UserID).First(&user).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", existing.ID).First(&existing).Error; err != nil {
				return err
			}

			// Patch user profile if values provided (MVP strategy: only fill when empty)
			updates := map[string]any{
				"last_login_at": now,
				"updated_at":    now,
			}
			if email != nil && (user.Email == nil || *user.Email == "") {
				updates["email"] = *email
			}
			if name != nil && (user.Name == nil || *user.Name == "") {
				updates["name"] = *name
			}
			if avatar != nil && (user.Avatar == nil || *user.Avatar == "") {
				updates["avatar"] = *avatar
			}
			if err := tx.Model(&model.User{}).
				Where("tenant_id = ? AND id = ?", tenantID, user.ID).
				Updates(updates).Error; err != nil {
				return err
			}

			// Patch identity email/profile if provided
			idUpdates := map[string]any{"updated_at": now}
			if email != nil {
				idUpdates["email"] = *email
			}
			if len(rawProfile) > 0 {
				idUpdates["raw_profile"] = rawProfile
			}
			if err := tx.Model(&model.UserIdentity{}).
				Where("id = ?", existing.ID).
				Updates(idUpdates).Error; err != nil {
				return err
			}

			u = &user
			it = &existing
			created = false
			return nil
		}

		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		uid, _ := common.NextUserID() // 生成对外 UID（MVP: 直接用自增 ID，后续可改为雪花 ID 或其他方案）
		// Create new user
		loginType := loginTypeFromProvider(provider)
		user := model.User{
			TenantID:     tenantID,
			UID:          uid,
			Status:       model.UserStatusActive,
			LoginType:    loginType,
			Email:        email,
			Name:         name,
			Avatar:       avatar,
			LastLoginAt:  now,
			CreatedAt:    now,
			UpdatedAt:    now,
			IsVIP:        0,
			VIPStartedAt: 0,
			VIPExpiredAt: 0,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Set externally exposed uid (MVP: equal to id)
		if user.UID == 0 {
			if err := tx.Model(&model.User{}).
				Where("tenant_id = ? AND id = ?", tenantID, user.ID).
				Update("uid", user.ID).Error; err != nil {
				return err
			}
			user.UID = user.ID
		}

		// Create identity (handle rare race with unique key)
		ident := model.UserIdentity{
			TenantID:    tenantID,
			UserID:      user.ID, // 关联用户对外 UID（MVP 方案）
			Provider:    provider,
			ProviderUID: providerUID,
			Email:       email,
			RawProfile:  rawProfile,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(&ident).Error; err != nil {
			// If duplicate due to race, load existing and continue
			if isDuplicateKey(err) {
				var ex model.UserIdentity
				if err2 := tx.Where("tenant_id = ? AND provider = ? AND provider_uid = ?", tenantID, provider, providerUID).First(&ex).Error; err2 != nil {
					return err
				}
				var exUser model.User
				if err2 := tx.Where("tenant_id = ? AND id = ?", tenantID, ex.UserID).First(&exUser).Error; err2 != nil {
					return err
				}
				u = &exUser
				it = &ex
				created = false
				return nil
			}
			return err
		}

		u = &user
		it = &ident
		created = true
		return nil
	})

	return u, it, created, err
}

// CreatePasswordUserWithIdentity 创建账号密码用户和 password identity（事务）。
func (r *gormRepo) CreatePasswordUserWithIdentity(
	ctx context.Context,
	tenantID uint64,
	email string,
	passwordHash string,
	name *string,
) (*model.User, *model.UserIdentity, error) {
	email = normalizeEmail(email)
	passwordHash = strings.TrimSpace(passwordHash)
	if email == "" {
		return nil, nil, errors.New("email required")
	}
	if passwordHash == "" {
		return nil, nil, errors.New("password_hash required")
	}

	var outUser *model.User
	var outIdentity *model.UserIdentity
	now := nowSec()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		uid, _ := common.NextUserID()
		emailPtr := email
		user := model.User{
			TenantID:     tenantID,
			UID:          uid,
			Status:       model.UserStatusActive,
			LoginType:    model.UserLoginTypePassword,
			Email:        &emailPtr,
			PasswordHash: &passwordHash,
			Name:         name,
			LastLoginAt:  0,
			CreatedAt:    now,
			UpdatedAt:    now,
			IsVIP:        0,
			VIPStartedAt: 0,
			VIPExpiredAt: 0,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if user.UID == 0 {
			if err := tx.Model(&model.User{}).
				Where("tenant_id = ? AND id = ?", tenantID, user.ID).
				Update("uid", user.ID).Error; err != nil {
				return err
			}
			user.UID = user.ID
		}

		ident := model.UserIdentity{
			TenantID:    tenantID,
			UserID:      user.ID,
			Provider:    model.IdentityProviderPassword,
			ProviderUID: email,
			Email:       &emailPtr,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.Create(&ident).Error; err != nil {
			return err
		}

		outUser = &user
		outIdentity = &ident
		return nil
	})
	return outUser, outIdentity, err
}

// CreateRefreshToken 创建一条 refresh token 记录（只存 hash）。
// CreateRefreshToken creates a refresh token record (only hash is stored).
func (r *gormRepo) CreateRefreshToken(
	ctx context.Context,
	tenantID, userID uint64,
	tokenHash string,
	deviceID, deviceName, ip, userAgent *string,
	expiredAt uint64,
) (*model.UserRefreshToken, error) {
	now := nowSec()
	row := &model.UserRefreshToken{
		TenantID:   tenantID,
		UserID:     userID,
		TokenHash:  tokenHash,
		DeviceID:   deviceID,
		DeviceName: deviceName,
		IP:         ip,
		UserAgent:  userAgent,
		Status:     model.RefreshTokenStatusActive,
		ExpiredAt:  expiredAt,
		LastUsedAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// GetRefreshTokenByHash 按 tenant_id + token_hash 查询 refresh token。
// GetRefreshTokenByHash queries refresh token by tenant_id + token_hash.
func (r *gormRepo) GetRefreshTokenByHash(ctx context.Context, tenantID uint64, tokenHash string) (*model.UserRefreshToken, error) {
	var rt model.UserRefreshToken
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND token_hash = ?", tenantID, tokenHash).
		First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// TouchRefreshToken 刷新 refresh token 的最近使用时间。
// TouchRefreshToken updates refresh token's last used time.
func (r *gormRepo) TouchRefreshToken(ctx context.Context, tokenHash string) error {
	now := nowSec()
	return r.db.WithContext(ctx).
		Model(&model.UserRefreshToken{}).
		Where("token_hash = ? AND status = ?", tokenHash, model.RefreshTokenStatusActive).
		Updates(map[string]any{"last_used_at": now, "updated_at": now}).Error
}

// RevokeRefreshToken 吊销单个 refresh token。
// RevokeRefreshToken revokes a single refresh token.
func (r *gormRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	now := nowSec()
	return r.db.WithContext(ctx).
		Model(&model.UserRefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Updates(map[string]any{"status": model.RefreshTokenStatusRevoked, "updated_at": now}).Error
}

// RevokeAllRefreshTokensByUser 吊销用户的全部有效 refresh token。
// RevokeAllRefreshTokensByUser revokes all valid refresh tokens for a user.
func (r *gormRepo) RevokeAllRefreshTokensByUser(ctx context.Context, tenantID, userID uint64) error {
	now := nowSec()
	return r.db.WithContext(ctx).
		Model(&model.UserRefreshToken{}).
		Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, model.RefreshTokenStatusActive).
		Updates(map[string]any{"status": model.RefreshTokenStatusRevoked, "updated_at": now}).Error
}

// CountUsers 统计用户总数。
func (r *gormRepo) CountUsers(ctx context.Context, filter stats.UserFilter) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.User{})
	if filter.TenantID > 0 {
		query = query.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.CreatedAfter != nil {
		query = query.Where("created_at >= ?", *filter.CreatedAfter)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountNewUsers 统计指定时间窗口内新增用户数。
func (r *gormRepo) CountNewUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.User{})
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if startDate != "" {
		query = query.Where("DATE(FROM_UNIXTIME(created_at)) >= ?", strings.TrimSpace(startDate))
	}
	if endDate != "" {
		query = query.Where("DATE(FROM_UNIXTIME(created_at)) <= ?", strings.TrimSpace(endDate))
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountActiveUsers 统计指定时间窗口内有活跃事件的去重用户数。
// 当前口径：client_events 中 uid>0 的去重用户数。
func (r *gormRepo) CountActiveUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error) {
	return r.countDistinctEventUsers(ctx, tenantID, nil, startDate, endDate)
}

// CountReturnUsers 统计指定时间窗口内的回流用户数。
// 当前先采用保守占位口径：有活跃事件，且在窗口开始前 30 天内无活跃事件。
func (r *gormRepo) CountReturnUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error) {
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	if tenantID == 0 {
		return 0, nil
	}
	if startDate == "" || endDate == "" {
		return 0, nil
	}

	start, err := time.ParseInLocation(time.DateOnly, startDate, time.Local)
	if err != nil {
		return 0, err
	}
	windowStart := start.AddDate(0, 0, -30).Format(time.DateOnly)
	windowEnd := start.AddDate(0, 0, -1).Format(time.DateOnly)

	activeUIDs, err := r.listDistinctEventUsers(ctx, tenantID, nil, startDate, endDate)
	if err != nil {
		return 0, err
	}
	if len(activeUIDs) == 0 {
		return 0, nil
	}

	priorUIDs, err := r.listDistinctEventUsers(ctx, tenantID, nil, windowStart, windowEnd)
	if err != nil {
		return 0, err
	}
	priorSet := make(map[uint64]struct{}, len(priorUIDs))
	for _, uid := range priorUIDs {
		priorSet[uid] = struct{}{}
	}

	var count int64
	for _, uid := range activeUIDs {
		if _, ok := priorSet[uid]; !ok {
			count++
		}
	}
	return count, nil
}

// CountLoginEvent 统计登录链路指定事件次数。
func (r *gormRepo) CountLoginEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error) {
	return r.countEventByNameAndOptionalCode(ctx, tenantID, eventName, nil, startDate, endDate)
}

// CountLoginMethodEvent 按登录方式统计事件次数。
// 当前约定：优先使用 event_code 作为 method 区分，如 google/apple/email。
func (r *gormRepo) CountLoginMethodEvent(ctx context.Context, tenantID uint64, method, eventName, startDate, endDate string) (int64, error) {
	method = strings.TrimSpace(method)
	if method == "" {
		return r.countEventByNameAndOptionalCode(ctx, tenantID, eventName, nil, startDate, endDate)
	}
	return r.countEventByNameAndOptionalCode(ctx, tenantID, eventName, &method, startDate, endDate)
}

// CountVipEvent 统计 VIP 转化链路事件次数。
func (r *gormRepo) CountVipEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error) {
	return r.countEventByNameAndOptionalCode(ctx, tenantID, eventName, nil, startDate, endDate)
}

func (r *gormRepo) countEventByNameAndOptionalCode(ctx context.Context, tenantID uint64, eventName string, eventCode *string, startDate, endDate string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	if tenantID == 0 || strings.TrimSpace(eventName) == "" {
		return 0, nil
	}

	query := r.db.WithContext(ctx).
		Table("client_events").
		Where("tenant_id = ?", tenantID).
		Where("event_name = ?", strings.TrimSpace(eventName))
	if eventCode != nil && strings.TrimSpace(*eventCode) != "" {
		query = query.Where("event_code = ?", strings.TrimSpace(*eventCode))
	}
	query = applyClientEventDateRange(query, startDate, endDate)

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *gormRepo) countDistinctEventUsers(ctx context.Context, tenantID uint64, eventNames []string, startDate, endDate string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, gorm.ErrInvalidDB
	}
	if tenantID == 0 {
		return 0, nil
	}

	query := r.db.WithContext(ctx).
		Table("client_events").
		Where("tenant_id = ?", tenantID).
		Where("uid > 0")
	if len(eventNames) > 0 {
		query = query.Where("event_name IN ?", cleanStrings(eventNames))
	}
	query = applyClientEventDateRange(query, startDate, endDate)

	var count int64
	if err := query.Distinct("uid").Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *gormRepo) listDistinctEventUsers(ctx context.Context, tenantID uint64, eventNames []string, startDate, endDate string) ([]uint64, error) {
	if r == nil || r.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	if tenantID == 0 {
		return nil, nil
	}

	query := r.db.WithContext(ctx).
		Table("client_events").
		Select("DISTINCT uid").
		Where("tenant_id = ?", tenantID).
		Where("uid > 0")
	if len(eventNames) > 0 {
		query = query.Where("event_name IN ?", cleanStrings(eventNames))
	}
	query = applyClientEventDateRange(query, startDate, endDate)

	rows := make([]struct {
		UID uint64 `gorm:"column:uid"`
	}, 0)
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row.UID > 0 {
			result = append(result, row.UID)
		}
	}
	return result, nil
}

func applyClientEventDateRange(query *gorm.DB, startDate, endDate string) *gorm.DB {
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	if startDate != "" {
		query = query.Where("DATE(FROM_UNIXTIME(created_at)) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("DATE(FROM_UNIXTIME(created_at)) <= ?", endDate)
	}
	return query
}

func cleanStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	result := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

// GetUserCreatedTrend 获取用户新增趋势。
func (r *gormRepo) GetUserCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]stats.TrendPoint, error) {
	baseWhere := "created_at > 0"
	return r.queryUnixDateTrend(ctx, "users", "created_at", tenantID, startDate, endDate, baseWhere)
}

func (r *gormRepo) ListUsers(ctx context.Context, tenantID uint64, filter user.QueryUserFilter) ([]*model.User, uint64, error) {
	var users []*model.User
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("tenant_id = ?", tenantID)

	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.StartAt != nil {
		query = query.Where("created_at >= ?", *filter.StartAt)
	}
	if filter.EndAt != nil {
		query = query.Where("created_at <= ?", *filter.EndAt)
	}
	// 中文注释：keyword 模糊匹配 uid/name/email；纯数字时额外精确匹配 id/uid
	if filter.Keyword != nil {
		keyword := strings.TrimSpace(*filter.Keyword)
		if keyword != "" {
			like := "%" + keyword + "%"
			cond := "(CAST(uid AS CHAR) LIKE ? OR name LIKE ? OR email LIKE ?)"
			args := []any{like, like, like}
			if v, err := strconv.ParseUint(keyword, 10, 64); err == nil {
				cond = "(" + cond + " OR id = ? OR uid = ?)"
				args = append(args, v, v)
			}
			query = query.Where(cond, args...)
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (filter.Page.Page - 1) * filter.Page.PageSize
	if offset < 0 {
		offset = 0
	}
	limit := filter.Page.PageSize
	if limit <= 0 {
		limit = 20
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, uint64(total), nil
}

func (r *gormRepo) UpdateUser(ctx context.Context, tenantID, userID uint64, req user.UpdateUserReq) error {
	updates := map[string]any{
		"updated_at": nowSec(),
	}
	if req.Email != nil {
		updates["email"] = *req.Email
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.IsVIP != nil {
		updates["is_vip"] = *req.IsVIP
		if *req.IsVIP == 1 {
			updates["vip_started_at"] = nowSec()
			if req.VIPExpired != nil {
				updates["vip_expired_at"] = *req.VIPExpired
			}
		} else {
			updates["vip_expired_at"] = uint64(0)
		}
	}
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("tenant_id = ? AND id = ?", tenantID, userID).
		Updates(updates).Error
}

// --- helpers ---

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func loginTypeFromProvider(provider string) string {
	switch strings.TrimSpace(provider) {
	case model.IdentityProviderPassword:
		return model.UserLoginTypePassword
	case model.IdentityProviderGoogle:
		return model.UserLoginTypeGoogle
	case model.IdentityProviderApple:
		return model.UserLoginTypeApple
	case model.IdentityProviderGitHub:
		return model.UserLoginTypeGitHub
	default:
		return model.UserLoginTypePassword
	}
}

// isDuplicateKey 判断是否为 MySQL 唯一键冲突（Error 1062）。
// NOTE: Avoid importing driver-specific error types.
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	// MySQL duplicate key: Error 1062
	msg := err.Error()
	return strings.Contains(msg, "Error 1062") || strings.Contains(msg, "Duplicate entry")
}

func (r *gormRepo) queryUnixDateTrend(ctx context.Context, tableName, unixColumn string, tenantID uint64, startDate, endDate, baseWhere string) ([]stats.TrendPoint, error) {
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	query := r.db.WithContext(ctx).
		Table(tableName).
		Select("DATE(FROM_UNIXTIME(" + unixColumn + ")) AS date, COUNT(*) AS count")
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	if strings.TrimSpace(baseWhere) != "" {
		query = query.Where(baseWhere)
	}
	if startDate != "" {
		query = query.Where("DATE(FROM_UNIXTIME("+unixColumn+")) >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("DATE(FROM_UNIXTIME("+unixColumn+")) <= ?", endDate)
	}
	var rows []stats.TrendRow
	err := query.
		Group("DATE(FROM_UNIXTIME(" + unixColumn + "))").
		Order("DATE(FROM_UNIXTIME(" + unixColumn + ")) ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	points := make([]stats.TrendPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, stats.TrendPoint{
			Date:  row.Date,
			Count: row.Count,
		})
	}
	return points, nil
}
