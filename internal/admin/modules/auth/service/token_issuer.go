package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	adminjwt "storeready_ai/internal/admin/auth/jwt"
	authdto "storeready_ai/internal/admin/modules/auth/dto"
)

var (
	ErrTokenIssuerManagerRequired   = errors.New("token issuer jwt manager is required")
	ErrTokenIssuerLoadUserMissing   = errors.New("token issuer load admin user func is required")
	ErrTokenIssuerAdminUserNotFound = errors.New("token issuer admin user not found")
)

// AdminUserSnapshot 是 token issuer 刷新 token 时所需的最小管理员信息快照。
// 这里故意保持轻量，避免 service 直接绑定具体 repo/entity 实现。
// app/bootstrap 层可通过闭包把 repo/entity 映射成这个结构。
//
// Status 说明：
// - 仅做透传，不在这里强耦合具体状态枚举；
// - 若调用方希望在 refresh 时校验禁用/删除状态，可在 LoadAdminUserFunc 内完成。
//
// IsSuperAdmin 当前不写入 jwt claims，但保留在快照中，便于后续扩展。
// 若后续 claims 需要增加 is_super_admin，可直接在 IssueTokenPair/RefreshToken 中补入。
//
// Username 允许为空，但通常建议返回最新用户名用于重新签发 token。
// ID 必须有效。
//
// 该结构仅用于包装层内部与 app 装配解耦。
// 不建议直接复用为 handler / dto / entity。
//
//nolint:revive
type AdminUserSnapshot struct {
	ID           uint64
	Username     string
	Status       uint8
	IsSuperAdmin uint8
}

// LoadAdminUserFunc 按 admin_user_id 加载管理员最新信息。
// 推荐在实现中完成：
// 1. 不存在判断；
// 2. 状态可用性判断；
// 3. 需要时记录关键日志。
type LoadAdminUserFunc func(ctx context.Context, tenantID uint64, adminUserID uint64) (AdminUserSnapshot, error)

// LoadRoleCodesFunc 按 admin_user_id 加载角色 code 列表。
// 返回空切片或 nil 均可，包装层会做清洗。
type LoadRoleCodesFunc func(ctx context.Context, tenantID uint64, adminUserID uint64) ([]string, error)

// RevokeRefreshTokenFunc 用于刷新 token 撤销。
// MVP 阶段若暂不做 refresh token 黑名单/持久化撤销，可传 nil。
type RevokeRefreshTokenFunc func(ctx context.Context, refreshToken string) error

type tokenIssuer struct {
	jwtManager         *adminjwt.Manager
	loadAdminUser      LoadAdminUserFunc
	loadRoleCodes      LoadRoleCodesFunc
	revokeRefreshToken RevokeRefreshTokenFunc
}

var _ TokenIssuer = (*tokenIssuer)(nil)

// NewTokenIssuer 使用 admin jwt manager 包装出 service 所需的 TokenIssuer。
//
// 设计说明：
// 1. service 继续依赖 TokenIssuer 抽象，不直接依赖具体 jwt.Manager；
// 2. app/bootstrap 层负责把 repo/service 通过闭包形式注入进来；
// 3. refresh token 刷新时会重新查管理员与角色，避免直接信任旧 token 中的角色信息；
// 4. 若暂不做 refresh token 撤销，revokeRefreshToken 可传 nil。
func NewTokenIssuer(
	jwtManager *adminjwt.Manager,
	loadAdminUser LoadAdminUserFunc,
	loadRoleCodes LoadRoleCodesFunc,
	revokeRefreshToken RevokeRefreshTokenFunc,
) (TokenIssuer, error) {
	if jwtManager == nil {
		return nil, ErrTokenIssuerManagerRequired
	}
	if loadAdminUser == nil {
		return nil, ErrTokenIssuerLoadUserMissing
	}
	return &tokenIssuer{
		jwtManager:         jwtManager,
		loadAdminUser:      loadAdminUser,
		loadRoleCodes:      loadRoleCodes,
		revokeRefreshToken: revokeRefreshToken,
	}, nil
}

func (i *tokenIssuer) IssueTokenPair(ctx context.Context, claims TokenIssueClaims) (authdto.TokenPair, error) {
	_ = ctx

	pair, err := i.jwtManager.SignTokenPair(adminjwt.SignInput{
		TenantID:    claims.TenantID,
		AdminUserID: claims.AdminUserID,
		Username:    strings.TrimSpace(claims.Username),
		Roles:       sanitizeStrings(claims.Roles),
	})
	if err != nil {
		return authdto.TokenPair{}, fmt.Errorf("issue admin token pair: %w", err)
	}
	return toAuthDTOTokenPair(pair), nil
}

func (i *tokenIssuer) RefreshToken(ctx context.Context, refreshToken string) (authdto.TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return authdto.TokenPair{}, ErrInvalidRefreshToken
	}

	claims, err := i.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return authdto.TokenPair{}, ErrInvalidRefreshToken
	}

	adminUserID := claims.GetAdminUserID()
	if adminUserID == 0 {
		return authdto.TokenPair{}, ErrInvalidRefreshToken
	}

	tenantID := claims.GetTenantID()
	if tenantID == 0 {
		return authdto.TokenPair{}, ErrInvalidRefreshToken
	}

	user, err := i.loadAdminUser(ctx, tenantID, adminUserID)
	if err != nil {
		return authdto.TokenPair{}, fmt.Errorf("load admin user for refresh token: %w", err)
	}
	if user.ID == 0 {
		return authdto.TokenPair{}, ErrTokenIssuerAdminUserNotFound
	}

	var roles []string
	if i.loadRoleCodes != nil {
		roles, err = i.loadRoleCodes(ctx, tenantID, user.ID)
		if err != nil {
			return authdto.TokenPair{}, fmt.Errorf("load admin roles for refresh token: %w", err)
		}
	}

	pair, err := i.jwtManager.SignTokenPair(adminjwt.SignInput{
		AdminUserID: user.ID,
		Username:    strings.TrimSpace(user.Username),
		Roles:       sanitizeStrings(roles),
	})
	if err != nil {
		return authdto.TokenPair{}, fmt.Errorf("refresh admin token pair: %w", err)
	}
	return toAuthDTOTokenPair(pair), nil
}

func (i *tokenIssuer) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}
	if i.revokeRefreshToken == nil {
		return nil
	}
	if err := i.revokeRefreshToken(ctx, refreshToken); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func toAuthDTOTokenPair(pair *adminjwt.TokenPair) authdto.TokenPair {
	if pair == nil {
		return authdto.TokenPair{}
	}
	return authdto.TokenPair{
		AccessToken:           pair.AccessToken,
		RefreshToken:          pair.RefreshToken,
		AccessTokenExpiresAt:  uint64(pair.AccessExpiresAt.Unix()),
		RefreshTokenExpiresAt: uint64(pair.RefreshExpiresAt.Unix()),
		TokenType:             "Bearer",
	}
}

func sanitizeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
