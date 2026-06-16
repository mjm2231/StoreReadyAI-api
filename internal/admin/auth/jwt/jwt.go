package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	contractsauth "storeready_ai/internal/contracts/auth"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidSecret        = errors.New("admin jwt secret is empty")
	ErrInvalidToken         = errors.New("invalid admin jwt token")
	ErrInvalidSigningMethod = errors.New("invalid admin jwt signing method")
	ErrUnexpectedClaims     = errors.New("unexpected admin jwt claims")
	ErrInvalidTenantID      = errors.New("invalid tenant id")
	ErrInvalidAdminUserID   = errors.New("invalid admin user id")
	ErrInvalidTokenType     = errors.New("invalid token type")
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Manager struct {
	issuer     string
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

var _ contractsauth.AdminJWTVerifier = (*Manager)(nil)

type SignInput struct {
	TenantID    uint64
	AdminUserID uint64
	Username    string
	Roles       []string
	TokenType   string
	ExpiresAt   time.Time
	NotBefore   time.Time
	IssuedAt    time.Time
	Audience    []string
	Subject     string
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

func NewManager(issuer string, secret string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, ErrInvalidSecret
	}
	if accessTTL <= 0 {
		accessTTL = 2 * time.Hour
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	return &Manager{
		issuer:     strings.TrimSpace(issuer),
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

func (m *Manager) SignAccessToken(input SignInput) (string, time.Time, error) {
	return m.sign(input, TokenTypeAccess, m.accessTTL)
}

func (m *Manager) SignRefreshToken(input SignInput) (string, time.Time, error) {
	return m.sign(input, TokenTypeRefresh, m.refreshTTL)
}

func (m *Manager) SignTokenPair(input SignInput) (*TokenPair, error) {
	accessToken, accessExpiresAt, err := m.SignAccessToken(input)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExpiresAt, err := m.SignRefreshToken(input)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

// ParseToken 负责解析并校验 Admin JWT 的基础合法性。
// 它只返回标准 claims，不依赖 Gin / middleware。
func (m *Manager) ParseToken(token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrInvalidToken
	}

	parsedToken, err := jwtv5.ParseWithClaims(token, &Claims{}, func(t *jwtv5.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSigningMethod
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse admin jwt token: %w", err)
	}

	claims, ok := parsedToken.Claims.(*Claims)
	if !ok {
		return nil, ErrUnexpectedClaims
	}
	if !parsedToken.Valid {
		return nil, ErrInvalidToken
	}
	if claims.GetTenantID() == 0 {
		return nil, ErrInvalidTenantID
	}
	if claims.GetAdminUserID() == 0 {
		return nil, ErrInvalidAdminUserID
	}
	if tokenType := claims.GetTokenType(); tokenType != "" && tokenType != TokenTypeAccess && tokenType != TokenTypeRefresh {
		return nil, ErrInvalidTokenType
	}
	if claims.GetTokenType() == "" {
		claims.TokenType = TokenTypeAccess
	}
	return claims, nil
}

// ParseAccessToken 仅负责 access token 解析与 claims 基础校验。
// 它不依赖 Gin，也不向任何 context snapshot 做映射。
func (m *Manager) ParseAccessToken(token string) (contractsauth.AdminClaims, error) {
	claims, err := m.ParseToken(token)
	if err != nil {
		return nil, err
	}
	if claims.GetTokenType() != TokenTypeAccess {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}

func (m *Manager) ParseRefreshToken(token string) (*Claims, error) {
	claims, err := m.ParseToken(token)
	if err != nil {
		return nil, err
	}
	if claims.GetTokenType() != TokenTypeRefresh {
		return nil, ErrInvalidTokenType
	}
	return claims, nil
}

func (m *Manager) sign(input SignInput, tokenType string, ttl time.Duration) (string, time.Time, error) {
	if input.TenantID == 0 {
		return "", time.Time{}, ErrInvalidTenantID
	}
	if input.AdminUserID == 0 {
		return "", time.Time{}, ErrInvalidAdminUserID
	}
	if tokenType != TokenTypeAccess && tokenType != TokenTypeRefresh {
		return "", time.Time{}, ErrInvalidTokenType
	}

	now := time.Now()
	issuedAt := input.IssuedAt
	if issuedAt.IsZero() {
		issuedAt = now
	}

	notBefore := input.NotBefore
	if notBefore.IsZero() {
		notBefore = issuedAt
	}

	expiresAt := input.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = issuedAt.Add(ttl)
	}

	subject := strings.TrimSpace(input.Subject)
	if subject == "" {
		subject = fmt.Sprintf("admin:%d", input.AdminUserID)
	}

	claims := Claims{
		TenantID:    input.TenantID,
		AdminUserID: input.AdminUserID,
		Username:    strings.TrimSpace(input.Username),
		Roles:       cloneStrings(input.Roles),
		TokenType:   tokenType,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   subject,
			Audience:  jwtv5.ClaimStrings(cloneStrings(input.Audience)),
			IssuedAt:  jwtv5.NewNumericDate(issuedAt),
			NotBefore: jwtv5.NewNumericDate(notBefore),
			ExpiresAt: jwtv5.NewNumericDate(expiresAt),
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign admin jwt token: %w", err)
	}
	return signed, expiresAt, nil
}

func cloneStrings(in []string) []string {
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
