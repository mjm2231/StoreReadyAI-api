package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authdto "storeready_ai/internal/admin/modules/auth/dto"
	usermodel "storeready_ai/internal/admin/modules/user/model"
	userrepo "storeready_ai/internal/admin/modules/user/repo"
	utils "storeready_ai/internal/pkg/uitls"
)

var (
	ErrNilUserRepo         = errors.New("admin auth service: user repo is nil")
	ErrNilPasswordHasher   = errors.New("admin auth service: password hasher is nil")
	ErrNilTokenIssuer      = errors.New("admin auth service: token issuer is nil")
	ErrInvalidUsername     = errors.New("admin auth service: invalid username")
	ErrInvalidPassword     = errors.New("admin auth service: invalid password")
	ErrPasswordNotMatch    = errors.New("admin auth service: password and confirm password do not match")
	ErrAdminUserExists     = errors.New("admin auth service: admin user already exists")
	ErrAdminUserNotFound   = errors.New("admin auth service: admin user not found")
	ErrAdminUserDisabled   = errors.New("admin auth service: admin user disabled")
	ErrAdminUserDeleted    = errors.New("admin auth service: admin user deleted")
	ErrInvalidCredentials  = errors.New("admin auth service: invalid credentials")
	ErrInvalidRefreshToken = errors.New("admin auth service: invalid refresh token")
	ErrInvalidOldPassword  = errors.New("admin auth service: invalid old password")
)

// PasswordHasher 负责后台管理员密码哈希与比对。
type PasswordHasher interface {
	HashPassword(password string, cost int) (string, error)
	ComparePassword(hash, password string) bool
}

// TokenIssuer 负责签发与撤销后台管理员 token。
//
// 说明：
// 1. service 只依赖抽象的 token issuer，不直接依赖具体 admin jwt manager；
// 2. 具体实现可由 jwt.Manager + refresh token store / blacklist 适配完成；
// 3. roles 由 service 侧装配后传入，用于写入 access/refresh token claims；
type TokenIssuer interface {
	// IssueTokenPair 根据后台管理员身份信息签发 access_token / refresh_token。
	IssueTokenPair(ctx context.Context, claims TokenIssueClaims) (authdto.TokenPair, error)
	// RefreshToken 基于 refresh token 完成校验并签发新的 token pair。
	RefreshToken(ctx context.Context, refreshToken string) (authdto.TokenPair, error)
	// RevokeRefreshToken 撤销 refresh token；若当前实现不做持久化撤销，可先返回 nil。
	RevokeRefreshToken(ctx context.Context, refreshToken string) error
}

// RoleProvider 负责查询后台管理员角色。
//
// 说明：
// 1. 当前为可选依赖；
// 2. 若未提供，则 Roles 返回 nil；
// 3. 后续可对接 admin user_roles / roles 模块。
type RoleProvider interface {
	GetRoleCodesByAdminUserID(ctx context.Context, tenantID uint64, adminUserID uint64) ([]string, error)
}

// TokenIssueClaims 是签发后台管理员 token 所需的最小身份信息。
type TokenIssueClaims struct {
	TenantID     uint64
	AdminUserID  uint64
	Username     string
	IsSuperAdmin uint8
	Roles        []string
}

// Service 是后台认证服务接口。
type Service interface {
	Register(ctx context.Context, req authdto.RegisterRequest) (authdto.RegisterResponse, error)
	Login(ctx context.Context, req authdto.LoginRequest, loginIP string) (authdto.LoginResponse, error)
	RefreshToken(ctx context.Context, req authdto.RefreshTokenRequest) (authdto.RefreshTokenResponse, error)
	Logout(ctx context.Context, req authdto.LogoutRequest) error
	ChangePassword(ctx context.Context, adminUserID uint64, req authdto.ChangePasswordRequest) error
	BuildProfile(ctx context.Context, user *usermodel.AdminUser) (authdto.AdminUserProfile, error)
}

// service 是 Service 的默认实现。
type service struct {
	userRepo       userrepo.Repository
	passwordHasher PasswordHasher
	tokenIssuer    TokenIssuer
	roleProvider   RoleProvider
	now            func() time.Time
}

func New(userRepo userrepo.Repository, passwordHasher PasswordHasher, tokenIssuer TokenIssuer, roleProvider RoleProvider) (Service, error) {
	if userRepo == nil || userRepo.DB() == nil {
		return nil, ErrNilUserRepo
	}
	if passwordHasher == nil {
		return nil, ErrNilPasswordHasher
	}
	if tokenIssuer == nil {
		return nil, ErrNilTokenIssuer
	}

	return &service{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		tokenIssuer:    tokenIssuer,
		roleProvider:   roleProvider,
		now:            time.Now,
	}, nil
}

func (s *service) SetNow(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = now
}

func (s *service) Register(ctx context.Context, req authdto.RegisterRequest) (authdto.RegisterResponse, error) {
	if s == nil {
		return authdto.RegisterResponse{}, ErrNilUserRepo
	}
	req = req.Normalize()
	if err := validateRegisterRequest(req); err != nil {
		return authdto.RegisterResponse{}, err
	}

	exists, err := s.userRepo.ExistsByUsername(ctx, req.TenantID, req.Username, 0)
	if err != nil {
		return authdto.RegisterResponse{}, err
	}
	if exists {
		return authdto.RegisterResponse{}, ErrAdminUserExists
	}
	if req.Email != "" {
		exists, err = s.userRepo.ExistsByEmail(ctx, req.TenantID, req.Email, 0)
		if err != nil {
			return authdto.RegisterResponse{}, err
		}
		if exists {
			return authdto.RegisterResponse{}, ErrAdminUserExists
		}
	}

	passwordHash, err := s.passwordHasher.HashPassword(req.Password, 12)
	if err != nil {
		return authdto.RegisterResponse{}, err
	}

	nowUnix := s.nowUnix()
	m := &usermodel.AdminUser{
		TenantID:     req.TenantID,
		Username:     req.Username,
		PasswordHash: strings.TrimSpace(passwordHash),
		Nickname:     req.Nickname,
		Email:        req.Email,
		Mobile:       req.Mobile,
		Avatar:       req.Avatar,
		Status:       usermodel.AdminUserStatusActive,
		IsSuperAdmin: normalizeBoolFlag(req.IsSuperAdmin),
		Remark:       req.Remark,
		CreatedAt:    nowUnix,
		UpdatedAt:    nowUnix,
	}
	if err := s.userRepo.Create(ctx, m); err != nil {
		return authdto.RegisterResponse{}, err
	}

	profile, err := s.BuildProfile(ctx, m)
	if err != nil {
		return authdto.RegisterResponse{}, err
	}
	return authdto.RegisterResponse{AdminUser: profile}, nil
}

func (s *service) Login(ctx context.Context, req authdto.LoginRequest, loginIP string) (authdto.LoginResponse, error) {
	if s == nil {
		return authdto.LoginResponse{}, ErrNilUserRepo
	}
	req = req.Normalize()
	if strings.TrimSpace(req.Username) == "" {
		return authdto.LoginResponse{}, ErrInvalidUsername
	}
	if strings.TrimSpace(req.Password) == "" {
		return authdto.LoginResponse{}, ErrInvalidPassword
	}
	tenantId64, err := utils.ToUint64(req.TenantID)
	if err != nil {
		return authdto.LoginResponse{}, err
	}
	user, err := s.userRepo.GetByUsername(ctx, tenantId64, req.Username)
	if err != nil {
		return authdto.LoginResponse{}, ErrInvalidCredentials
	}
	if err := ensureAdminUserAvailable(user); err != nil {
		return authdto.LoginResponse{}, err
	}
	if !s.passwordHasher.ComparePassword(user.PasswordHash, req.Password) {
		return authdto.LoginResponse{}, ErrInvalidCredentials
	}

	roles, err := s.loadRoles(ctx, user.TenantID, user.ID)
	fmt.Printf("admin user %d roles: %v\n", user.ID, roles)
	if err != nil {
		return authdto.LoginResponse{}, err
	}
	pair, err := s.tokenIssuer.IssueTokenPair(ctx, TokenIssueClaims{
		TenantID:     user.TenantID,
		AdminUserID:  user.ID,
		Username:     user.Username,
		IsSuperAdmin: user.IsSuperAdmin,
		Roles:        roles,
	})
	if err != nil {
		return authdto.LoginResponse{}, err
	}

	nowUnix := s.nowUnix()
	if err := s.userRepo.UpdateLoginInfo(ctx, user.TenantID, user.ID, nowUnix, strings.TrimSpace(loginIP), nowUnix); err != nil {
		return authdto.LoginResponse{}, err
	}
	user.LastLoginAt = nowUnix
	user.LastLoginIP = strings.TrimSpace(loginIP)
	user.UpdatedAt = nowUnix

	profile := toAdminUserProfile(user, roles)
	return authdto.LoginResponse{
		AdminUser: profile,
		Token:     pair,
	}, nil
}

func (s *service) RefreshToken(ctx context.Context, req authdto.RefreshTokenRequest) (authdto.RefreshTokenResponse, error) {
	if s == nil {
		return authdto.RefreshTokenResponse{}, ErrNilTokenIssuer
	}
	req = req.Normalize()
	if req.RefreshToken == "" {
		return authdto.RefreshTokenResponse{}, ErrInvalidRefreshToken
	}
	pair, err := s.tokenIssuer.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return authdto.RefreshTokenResponse{}, err
	}
	return authdto.RefreshTokenResponse{Token: pair}, nil
}

func (s *service) Logout(ctx context.Context, req authdto.LogoutRequest) error {
	if s == nil {
		return ErrNilTokenIssuer
	}
	req = req.Normalize()
	if req.RefreshToken == "" {
		return nil
	}
	if err := s.tokenIssuer.RevokeRefreshToken(ctx, req.RefreshToken); err != nil {
		return err
	}
	return nil
}

func (s *service) ChangePassword(ctx context.Context, adminUserID uint64, req authdto.ChangePasswordRequest) error {
	if s == nil {
		return ErrNilUserRepo
	}
	req = req.Normalize()
	if adminUserID == 0 {
		return ErrAdminUserNotFound
	}
	if strings.TrimSpace(req.OldPassword) == "" {
		return ErrInvalidOldPassword
	}
	if strings.TrimSpace(req.NewPassword) == "" {
		return ErrInvalidPassword
	}
	if req.NewPassword != req.ConfirmPassword {
		return ErrPasswordNotMatch
	}

	user, err := s.userRepo.GetByID(ctx, req.TenantID, adminUserID)
	if err != nil {
		return ErrAdminUserNotFound
	}
	if err := ensureAdminUserAvailable(user); err != nil {
		return err
	}
	if !s.passwordHasher.ComparePassword(user.PasswordHash, req.OldPassword) {
		return ErrInvalidOldPassword
	}

	passwordHash, err := s.passwordHasher.HashPassword(req.NewPassword, 12)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, req.TenantID, adminUserID, passwordHash, s.nowUnix())
}

func (s *service) BuildProfile(ctx context.Context, user *usermodel.AdminUser) (authdto.AdminUserProfile, error) {
	if err := ensureAdminUserAvailable(user); err != nil {
		return authdto.AdminUserProfile{}, err
	}
	roles, err := s.loadRoles(ctx, user.TenantID, user.ID)
	if err != nil {
		return authdto.AdminUserProfile{}, err
	}
	return toAdminUserProfile(user, roles), nil
}

func (s *service) loadRoles(ctx context.Context, tenantID uint64, adminUserID uint64) ([]string, error) {
	fmt.Printf("loadRoles admin user %d tenantID: %v\n", adminUserID, tenantID)
	if s == nil || s.roleProvider == nil || tenantID == 0 || adminUserID == 0 {
		return nil, nil
	}
	roles, err := s.roleProvider.GetRoleCodesByAdminUserID(ctx, tenantID, adminUserID)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *service) nowUnix() uint64 {
	if s == nil || s.now == nil {
		return uint64(time.Now().Unix())
	}
	return uint64(s.now().Unix())
}

func validateRegisterRequest(req authdto.RegisterRequest) error {
	if strings.TrimSpace(req.Username) == "" {
		return ErrInvalidUsername
	}
	if strings.TrimSpace(req.Password) == "" {
		return ErrInvalidPassword
	}
	if req.Password != req.ConfirmPassword {
		return ErrPasswordNotMatch
	}
	return nil
}

func ensureAdminUserAvailable(user *usermodel.AdminUser) error {
	if user == nil {
		return ErrAdminUserNotFound
	}
	if user.IsDeleted() {
		return ErrAdminUserDeleted
	}
	if user.IsDisabled() {
		return ErrAdminUserDisabled
	}
	if !user.IsActive() {
		return ErrInvalidCredentials
	}
	return nil
}

func toAdminUserProfile(user *usermodel.AdminUser, roles []string) authdto.AdminUserProfile {
	if user == nil {
		return authdto.AdminUserProfile{}
	}
	return authdto.AdminUserProfile{
		TenantID:     user.TenantID,
		ID:           user.ID,
		Username:     user.Username,
		Nickname:     user.Nickname,
		Email:        user.Email,
		Mobile:       user.Mobile,
		Avatar:       user.Avatar,
		Status:       user.Status,
		IsSuperAdmin: user.IsSuperAdmin,
		Roles:        roles,
		LastLoginAt:  user.LastLoginAt,
		LastLoginIP:  user.LastLoginIP,
		Remark:       user.Remark,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func normalizeBoolFlag(v uint8) uint8 {
	if v > 0 {
		return 1
	}
	return 0
}
