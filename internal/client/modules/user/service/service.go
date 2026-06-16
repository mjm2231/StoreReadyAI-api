package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	statsservice "storeready_ai/internal/admin/modules/stats/service"
	"storeready_ai/internal/client/modules/user/dto"
	"storeready_ai/internal/client/modules/user/model"
	"storeready_ai/internal/client/modules/user/repo"
	"storeready_ai/internal/contracts/entitlement"
	"storeready_ai/internal/contracts/stats"
	"storeready_ai/internal/contracts/user"
	errx "storeready_ai/internal/pkg/errors"
)

type StatsUserService interface {
	GetUserCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]stats.TrendPoint, error)
	CountUsers(ctx context.Context, filter stats.UserFilter) (int64, error)
}

// 下面这些 provider 接口用于承接后台 stats 对 user 模块的扩展统计能力。
// 采用 service 内部小接口断言方式，避免立刻扩大 repo.UserRepo 主接口，后续 repo 实现后可直接被复用。
type activeUsersCounter interface {
	CountActiveUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
}

type loginEventCounter interface {
	CountLoginEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
}

type loginMethodEventCounter interface {
	CountLoginMethodEvent(ctx context.Context, tenantID uint64, method, eventName, startDate, endDate string) (int64, error)
}

type newUsersCounter interface {
	CountNewUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
}

type returnUsersCounter interface {
	CountReturnUsers(ctx context.Context, tenantID uint64, startDate, endDate string) (int64, error)
}

type vipEventCounter interface {
	CountVipEvent(ctx context.Context, tenantID uint64, eventName, startDate, endDate string) (int64, error)
}

// UserService 用户领域 Service 层接口。
// 说明：只依赖接口（repo/verifier/token issuer），便于后期接入 gRPC 或替换实现。
type UserService interface {
	StatsUserService
	CountNewUsers(ctx context.Context, tenantID uint64, startDate string, endDate string) (int64, error)
	CountActiveUsers(ctx context.Context, tenantID uint64, startDate string, endDate string) (int64, error)
	CountReturnUsers(ctx context.Context, tenantID uint64, startDate string, endDate string) (int64, error)
	CountLoginEvent(ctx context.Context, tenantID uint64, eventName string, startDate string, endDate string) (int64, error)
	CountLoginMethodEvent(ctx context.Context, tenantID uint64, method string, eventName string, startDate string, endDate string) (int64, error)
	CountVipEvent(ctx context.Context, tenantID uint64, eventName string, startDate string, endDate string) (int64, error)
	// FirebaseLogin Firebase 第三方登录：校验 id_token -> upsert user+identity -> 颁发 access/refresh token。
	FirebaseLogin(ctx context.Context, tenantID uint64, req dto.FirebaseLoginReq) (*dto.FirebaseLoginResp, error)

	// AccountRegister 账号密码注册：创建用户 + password identity -> 颁发 access/refresh token。
	AccountRegister(ctx context.Context, tenantID uint64, req dto.AccountRegisterReq) (*dto.AccountRegisterResp, error)

	// AccountLogin 账号密码登录：校验邮箱密码 -> 颁发 access/refresh token。
	AccountLogin(ctx context.Context, tenantID uint64, req dto.AccountLoginReq) (*dto.AccountLoginResp, error)

	// RefreshAccessToken 续期（可选）：用 refresh_token 换新的 access_token。
	RefreshAccessToken(ctx context.Context, tenantID uint64, req dto.RefreshTokenReq) (*dto.RefreshTokenResp, error)

	// Logout 登出（可选）：吊销 refresh_token。
	Logout(ctx context.Context, tenantID uint64, req dto.LogoutReq) error

	// ListUsers 列表查询用户（分页、过滤）。
	ListUsers(ctx context.Context, tenantID uint64, filter user.QueryUserFilter) ([]*user.UserVO, uint64, error)
	// UpdateUser 更新用户信息（可选：后续用到再加）。
	UpdateUser(ctx context.Context, tenantID, userID uint64, req user.UpdateUserReq) error
	// GetUserByID returns a user by externally exposed id.
	GetUserByID(ctx context.Context, tenantID, id uint64) (*user.UserVO, error)
}

// ---- 依赖接口（由外部注入具体实现） ----

// FirebaseVerifier 用于验证 Firebase ID Token。
// service 层不直接依赖 firebase sdk，避免耦合。
type FirebaseVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*FirebaseClaims, error)
}

// FirebaseClaims 从 Firebase ID token 中抽取的必要字段。
type FirebaseClaims struct {
	Provider    string // google/apple/anonymous 或 google.com/apple.com
	ProviderUID string // Firebase UID
	Email       *string
	Name        *string
	Avatar      *string
	RawProfile  datatypes.JSON // 可选：留给排查/扩展
}

// TokenIssuer 你自己系统的 token 颁发器（access JWT + refresh token）。
// access 一般是 JWT；refresh 建议是随机串 + DB 存 hash。
type TokenIssuer interface {
	IssueAccessToken(uid uint64, tenantID uint64) (token string, exp time.Time, err error)
}

// serviceImpl UserService 的默认实现。
type serviceImpl struct {
	users      repo.UserRepo
	verifier   FirebaseVerifier
	issuer     TokenIssuer
	refreshTTL time.Duration
}

// CountActiveUsers 统计指定时间窗口内的活跃用户数。
func (s *serviceImpl) CountActiveUsers(ctx context.Context, tenantID uint64, startDate string, endDate string) (int64, error) {
	if s == nil || s.users == nil {
		return 0, errx.New(errx.CodeInternal, "user repo not configured")
	}
	if tenantID == 0 {
		return 0, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	provider, ok := s.users.(activeUsersCounter)
	if !ok {
		return 0, errx.New(errx.CodeInternal, "user repo does not implement CountActiveUsers")
	}
	return provider.CountActiveUsers(ctx, tenantID, startDate, endDate)
}

// CountLoginEvent 统计登录链路指定事件次数。
func (s *serviceImpl) CountLoginEvent(ctx context.Context, tenantID uint64, eventName string, startDate string, endDate string) (int64, error) {
	if s == nil || s.users == nil {
		return 0, errx.New(errx.CodeInternal, "user repo not configured")
	}
	if tenantID == 0 {
		return 0, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	provider, ok := s.users.(loginEventCounter)
	if !ok {
		return 0, errx.New(errx.CodeInternal, "user repo does not implement CountLoginEvent")
	}
	return provider.CountLoginEvent(ctx, tenantID, eventName, startDate, endDate)
}

// CountLoginMethodEvent 按登录方式统计事件次数。
func (s *serviceImpl) CountLoginMethodEvent(ctx context.Context, tenantID uint64, method string, eventName string, startDate string, endDate string) (int64, error) {
	if s == nil || s.users == nil {
		return 0, errx.New(errx.CodeInternal, "user repo not configured")
	}
	if tenantID == 0 {
		return 0, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	provider, ok := s.users.(loginMethodEventCounter)
	if !ok {
		return 0, errx.New(errx.CodeInternal, "user repo does not implement CountLoginMethodEvent")
	}
	return provider.CountLoginMethodEvent(ctx, tenantID, method, eventName, startDate, endDate)
}

// CountNewUsers 统计指定时间窗口内新增用户数。
func (s *serviceImpl) CountNewUsers(ctx context.Context, tenantID uint64, startDate string, endDate string) (int64, error) {
	if s == nil || s.users == nil {
		return 0, errx.New(errx.CodeInternal, "user repo not configured")
	}
	if tenantID == 0 {
		return 0, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	provider, ok := s.users.(newUsersCounter)
	if !ok {
		return 0, errx.New(errx.CodeInternal, "user repo does not implement CountNewUsers")
	}
	return provider.CountNewUsers(ctx, tenantID, startDate, endDate)
}

// CountReturnUsers 统计指定时间窗口内的回流用户数。
func (s *serviceImpl) CountReturnUsers(ctx context.Context, tenantID uint64, startDate string, endDate string) (int64, error) {
	if s == nil || s.users == nil {
		return 0, errx.New(errx.CodeInternal, "user repo not configured")
	}
	if tenantID == 0 {
		return 0, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	provider, ok := s.users.(returnUsersCounter)
	if !ok {
		return 0, errx.New(errx.CodeInternal, "user repo does not implement CountReturnUsers")
	}
	return provider.CountReturnUsers(ctx, tenantID, startDate, endDate)
}

// CountVipEvent 统计 VIP 转化链路事件次数。
func (s *serviceImpl) CountVipEvent(ctx context.Context, tenantID uint64, eventName string, startDate string, endDate string) (int64, error) {
	if s == nil || s.users == nil {
		return 0, errx.New(errx.CodeInternal, "user repo not configured")
	}
	if tenantID == 0 {
		return 0, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	provider, ok := s.users.(vipEventCounter)
	if !ok {
		return 0, errx.New(errx.CodeInternal, "user repo does not implement CountVipEvent")
	}
	return provider.CountVipEvent(ctx, tenantID, eventName, startDate, endDate)
}

var _ statsservice.UserService = (*serviceImpl)(nil)

// New 创建 UserService。
func New(users repo.UserRepo, verifier FirebaseVerifier, issuer TokenIssuer, refreshTTL time.Duration) UserService {
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	return &serviceImpl{
		users:      users,
		verifier:   verifier,
		issuer:     issuer,
		refreshTTL: refreshTTL,
	}
}

// FirebaseLogin Firebase 第三方登录。
func (s *serviceImpl) FirebaseLogin(ctx context.Context, tenantID uint64, req dto.FirebaseLoginReq) (*dto.FirebaseLoginResp, error) {
	idToken := strings.TrimSpace(req.IDToken)
	if idToken == "" {
		return nil, errx.New(errx.CodeInvalidParam, "id_token required")
	}
	if s.verifier == nil {
		return nil, errx.New(errx.CodeAuthNotConfigured, "firebase verifier not configured")
	}
	if s.issuer == nil {
		return nil, errx.New(errx.CodeAuthNotConfigured, "token issuer not configured")
	}

	claims, err := s.verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeAuthInvalidToken, "firebase verify id_token failed")
	}
	provider := normalizeProvider(claims.Provider)
	providerUID := strings.TrimSpace(claims.ProviderUID)
	if providerUID == "" {
		return nil, errx.New(errx.CodeAuthInvalidToken, "invalid provider_uid")
	}

	// upsert user + identity（事务）
	u, _, _, err := s.users.UpsertUserWithIdentity(
		ctx,
		tenantID,
		provider,
		providerUID,
		claims.Email,
		claims.Name,
		claims.Avatar,
		claims.RawProfile,
	)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "upsert user failed")
	}

	// access token（JWT）
	access, aexp, err := s.issuer.IssueAccessToken(u.UID, tenantID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "issue access token failed")
	}

	// refresh token：随机串 + DB 存 hash
	refreshPlain, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "generate refresh token failed")
	}
	rexp := time.Now().Add(s.refreshTTL)
	deviceID := req.DeviceID
	deviceName := req.DeviceName
	// IP/UA 由 handler 传更合适；这里暂时留空
	_, err = s.users.CreateRefreshToken(ctx, tenantID, u.ID, refreshHash, deviceID, deviceName, nil, nil, uint64(rexp.Unix()))
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create refresh token failed")
	}
	entitlement := entitlement.NewEntitlement(*u)
	entitlement.Normalize()
	resp := &dto.FirebaseLoginResp{
		Token: dto.TokenPair{
			AccessToken:  access,
			ExpiresIn:    int64(time.Until(aexp).Seconds()),
			RefreshToken: refreshPlain,
			RefreshExpIn: int64(time.Until(rexp).Seconds()),
		},
		User:        toUserVO(u),
		Entitlement: entitlement,
		ServerNow:   nowSec(),
	}
	return resp, nil
}

// nowSec 当前时间（Unix 秒）。
func nowSec() uint64 { return uint64(time.Now().Unix()) }

// AccountRegister 账号密码注册。
func (s *serviceImpl) AccountRegister(ctx context.Context, tenantID uint64, req dto.AccountRegisterReq) (*dto.AccountRegisterResp, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, errx.New(errx.CodeInvalidParam, "email required")
	}
	password := strings.TrimSpace(req.Password)
	if len(password) < 8 {
		return nil, errx.New(errx.CodeInvalidParam, "password length must be at least 8")
	}
	if s.issuer == nil {
		return nil, errx.New(errx.CodeAuthNotConfigured, "token issuer not configured")
	}

	if _, err := s.users.GetIdentity(ctx, tenantID, model.IdentityProviderPassword, email); err == nil {
		return nil, errx.New(errx.CodeInvalidParam, "email already registered")
	} else if !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errx.Wrap(err, errx.CodeInternal, "query identity failed")
	}

	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "hash password failed")
	}

	u, _, err := s.users.CreatePasswordUserWithIdentity(ctx, tenantID, email, string(passwordBytes), req.Name)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create password user failed")
	}

	access, aexp, err := s.issuer.IssueAccessToken(u.UID, tenantID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "issue access token failed")
	}

	refreshPlain, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "generate refresh token failed")
	}
	rexp := time.Now().Add(s.refreshTTL)
	_, err = s.users.CreateRefreshToken(ctx, tenantID, u.ID, refreshHash, req.DeviceID, req.DeviceName, nil, nil, uint64(rexp.Unix()))
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create refresh token failed")
	}

	entitlement := entitlement.NewEntitlement(*u)
	entitlement.Normalize()
	return &dto.AccountRegisterResp{
		Token: dto.TokenPair{
			AccessToken:  access,
			ExpiresIn:    int64(time.Until(aexp).Seconds()),
			RefreshToken: refreshPlain,
			RefreshExpIn: int64(time.Until(rexp).Seconds()),
		},
		User:        toUserVO(u),
		Entitlement: entitlement,
		ServerNow:   nowSec(),
	}, nil
}

// AccountLogin 账号密码登录。
func (s *serviceImpl) AccountLogin(ctx context.Context, tenantID uint64, req dto.AccountLoginReq) (*dto.AccountLoginResp, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, errx.New(errx.CodeInvalidParam, "email required")
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return nil, errx.New(errx.CodeInvalidParam, "password required")
	}
	if s.issuer == nil {
		return nil, errx.New(errx.CodeAuthNotConfigured, "token issuer not configured")
	}

	identity, err := s.users.GetIdentity(ctx, tenantID, model.IdentityProviderPassword, email)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.New(errx.CodeInternal, "invalid email or password")
		}
		return nil, errx.Wrap(err, errx.CodeInternal, "query identity failed")
	}

	u, err := s.users.GetUserByID(ctx, tenantID, identity.UserID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "query user failed")
	}
	if u.Status != model.UserStatusActive {
		return nil, errx.New(errx.CodeInternal, "user is not active")
	}
	if u.PasswordHash == nil || strings.TrimSpace(*u.PasswordHash) == "" {
		return nil, errx.New(errx.CodeInternal, "invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return nil, errx.New(errx.CodeInternal, "invalid email or password")
	}

	access, aexp, err := s.issuer.IssueAccessToken(u.UID, tenantID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "issue access token failed")
	}

	refreshPlain, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "generate refresh token failed")
	}
	rexp := time.Now().Add(s.refreshTTL)
	_, err = s.users.CreateRefreshToken(ctx, tenantID, u.ID, refreshHash, req.DeviceID, req.DeviceName, nil, nil, uint64(rexp.Unix()))
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create refresh token failed")
	}

	entitlement := entitlement.NewEntitlement(*u)
	entitlement.Normalize()
	return &dto.AccountLoginResp{
		Token: dto.TokenPair{
			AccessToken:  access,
			ExpiresIn:    int64(time.Until(aexp).Seconds()),
			RefreshToken: refreshPlain,
			RefreshExpIn: int64(time.Until(rexp).Seconds()),
		},
		User:        toUserVO(u),
		Entitlement: entitlement,
		ServerNow:   nowSec(),
	}, nil
}

// RefreshAccessToken 用 refresh_token 换取新的 access_token。
func (s *serviceImpl) RefreshAccessToken(ctx context.Context, tenantID uint64, req dto.RefreshTokenReq) (*dto.RefreshTokenResp, error) {
	rt := strings.TrimSpace(req.RefreshToken)
	if rt == "" {
		return nil, errx.New(errx.CodeInvalidParam, "refresh_token required")
	}
	if s.issuer == nil {
		return nil, errx.New(errx.CodeAuthNotConfigured, "token issuer not configured")
	}

	hash := sha256Hex(rt)

	row, err := s.users.GetRefreshTokenByHash(ctx, tenantID, hash)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.New(errx.CodeAuthRefreshTokenInvalid, "invalid refresh_token")
		}
		return nil, errx.Wrap(err, errx.CodeInternal, "query refresh token failed")
	}
	if row.Status != model.RefreshTokenStatusActive {
		return nil, errx.New(errx.CodeAuthRefreshTokenRevoked, "refresh_token revoked")
	}
	if row.ExpiredAt > 0 && uint64(time.Now().Unix()) >= row.ExpiredAt {
		return nil, errx.New(errx.CodeAuthRefreshTokenExpired, "refresh_token expired")
	}

	// touch last_used_at
	_ = s.users.TouchRefreshToken(ctx, hash)

	// 颁发新的 access token（用用户对外 UID）
	u, err := s.users.GetUserByID(ctx, tenantID, row.UserID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "load user failed")
	}
	access, aexp, err := s.issuer.IssueAccessToken(u.UID, tenantID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "issue access token failed")
	}

	rexp := int64(0)
	if row.ExpiredAt > 0 {
		rexp = int64(int64(row.ExpiredAt) - time.Now().Unix())
		if rexp < 0 {
			rexp = 0
		}
	}
	entitlement := entitlement.NewEntitlement(*u)
	entitlement.Normalize()
	return &dto.RefreshTokenResp{
		Token: dto.TokenPair{
			AccessToken:  access,
			ExpiresIn:    int64(time.Until(aexp).Seconds()),
			RefreshToken: rt,
			RefreshExpIn: rexp,
		},
		User:        toUserVO(u),
		ServerNow:   nowSec(),
		Entitlement: entitlement,
	}, nil
}

// Logout 吊销 refresh_token。
func (s *serviceImpl) Logout(ctx context.Context, tenantID uint64, req dto.LogoutReq) error {
	rt := strings.TrimSpace(req.RefreshToken)
	if rt == "" {
		return errx.New(errx.CodeInvalidParam, "refresh_token required")
	}
	hash := sha256Hex(rt)
	if err := s.users.RevokeRefreshToken(ctx, hash); err != nil {
		return errx.Wrap(err, errx.CodeInternal, "revoke refresh token failed")
	}
	return nil
}

// ---- helpers ----

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func toUserVO(u *model.User) user.UserVO {
	return user.UserVO{
		ID:           u.ID,
		UID:          u.UID,
		TenantID:     u.TenantID,
		Status:       u.Status,
		Email:        u.Email,
		Name:         u.Name,
		Avatar:       u.Avatar,
		Locale:       u.Locale,
		Timezone:     u.Timezone,
		IsVIP:        u.IsVIP,
		VIPStartedAt: u.VIPStartedAt,
		VIPExpired:   u.VIPExpiredAt,
		LastLogin:    u.LastLoginAt,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func normalizeProvider(p string) string {
	p = strings.TrimSpace(strings.ToLower(p))
	switch p {
	case "google", "google.com":
		return model.IdentityProviderGoogle
	case "apple", "apple.com":
		return model.IdentityProviderApple
	case "anonymous":
		return model.IdentityProviderAnonymous
	default:
		// 兜底：原样返回，避免因为未知 provider 导致无法登录
		if p == "" {
			return model.IdentityProviderAnonymous
		}
		return p
	}
}

func newRefreshToken() (plain string, hash string, err error) {
	// 32 bytes random -> 64 hex chars
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	hash = sha256Hex(plain)
	return plain, hash, nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// / 获取用户新增趋势
func (s *serviceImpl) GetUserCreatedTrend(ctx context.Context, tenantID uint64, startDate, endDate string) ([]stats.TrendPoint, error) {
	return s.users.GetUserCreatedTrend(ctx, tenantID, startDate, endDate)
}

// CountUsers 统计用户总数。
func (s *serviceImpl) CountUsers(ctx context.Context, filter stats.UserFilter) (int64, error) {
	return s.users.CountUsers(ctx, filter)
}

// ListUsers 列表查询用户（分页、过滤）。
func (s *serviceImpl) ListUsers(ctx context.Context, tenantID uint64, filter user.QueryUserFilter) ([]*user.UserVO, uint64, error) {
	users, total, err := s.users.ListUsers(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, errx.Wrap(err, errx.CodeInternal, "list users failed")
	}
	vos := make([]*user.UserVO, 0, len(users))
	for _, u := range users {
		vo := toUserVOWithEntitlement(u)
		vos = append(vos, &vo)
	}
	return vos, total, nil
}

func toUserVOWithEntitlement(u *model.User) user.UserVO {
	vo := toUserVO(u)

	entitlement := entitlement.NewEntitlement(*u)
	entitlement.Normalize()

	vo.IsVIP = entitlement.IsVIP

	return vo
}

// UpdateUser 更新用户信息（可选：后续用到再加）。
func (s *serviceImpl) UpdateUser(ctx context.Context, tenantID, userID uint64, req user.UpdateUserReq) error {
	return s.users.UpdateUser(ctx, tenantID, userID, req)
}

// GetUserByUID 根据用户 UID 获取用户信息。
func (s *serviceImpl) GetUserByID(ctx context.Context, tenantID, id uint64) (*user.UserVO, error) {
	u, err := s.users.GetUserByID(ctx, tenantID, id)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "get user by uid failed")
	}
	vo := toUserVOWithEntitlement(u)
	return &vo, nil
}
