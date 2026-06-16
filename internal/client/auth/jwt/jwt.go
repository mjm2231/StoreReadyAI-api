package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"time"

	contractsauth "storeready_ai/internal/contracts/auth"

	jwt "github.com/golang-jwt/jwt/v5"
)

// =====================
// 配置
// =====================

var (
	ErrUnauthorized     = errors.New("app jwt unauthorized")
	ErrInvalidTokenType = errors.New("app jwt invalid token type")
)

type Algorithm string

const (
	AlgHS256 Algorithm = "HS256"
	AlgRS256 Algorithm = "RS256"
)

type Config struct {
	Alg Algorithm

	// HS256
	HMACSecret []byte

	// RS256
	RSAPublicKey  *rsa.PublicKey
	RSAPrivateKey *rsa.PrivateKey

	// 校验项
	Issuer   string // iss
	Audience string // aud（可选）
	Leeway   time.Duration

	// 有效期
	AccessTTL time.Duration
}

// =====================
// 工具主体
// =====================

type Manager struct {
	cfg Config
}

var _ contractsauth.AppJWTVerifier = (*Manager)(nil)

func New(cfg Config) (*Manager, error) {
	// 基础校验
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = 24 * time.Hour
	}
	if cfg.Leeway < 0 {
		cfg.Leeway = 0
	}
	switch cfg.Alg {
	case AlgHS256:
		if len(cfg.HMACSecret) == 0 {
			return nil, fmt.Errorf("jwt: missing HMACSecret for HS256")
		}
	case AlgRS256:
		if cfg.RSAPublicKey == nil || cfg.RSAPrivateKey == nil {
			return nil, fmt.Errorf("jwt: missing RSA keys for RS256")
		}
	default:
		return nil, fmt.Errorf("jwt: unsupported alg: %s", cfg.Alg)
	}
	if cfg.Issuer == "" {
		return nil, fmt.Errorf("jwt: missing issuer")
	}
	return &Manager{cfg: cfg}, nil
}

// =====================
// 签发
// =====================

type SignInput struct {
	UID      string
	TenantID string
	Role     string
	Scopes   []string
	TokenVer int64

	// 可选：自定义过期时间（为空用 cfg.AccessTTL）
	ExpiresAt time.Time
	// 可选：自定义 subject（为空用 UID）
	Subject string
}

func (m *Manager) SignAccessToken(in SignInput) (string, error) {
	now := time.Now()

	exp := in.ExpiresAt
	if exp.IsZero() {
		exp = now.Add(m.cfg.AccessTTL)
	}
	sub := in.Subject
	if sub == "" {
		sub = in.UID
	}

	c := Claims{
		UID:       in.UID,
		TenantID:  in.TenantID,
		Role:      in.Role,
		Scopes:    in.Scopes,
		TokenVer:  in.TokenVer,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:  m.cfg.Issuer,
			Subject: sub,
			// 如果配置了 audience，则在签发时写入 aud，避免验签阶段因为缺失 aud 而误判失败。
			Audience: func() jwt.ClaimStrings {
				if m.cfg.Audience == "" {
					return nil
				}
				return jwt.ClaimStrings{m.cfg.Audience}
			}(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-m.cfg.Leeway)),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}

	var method jwt.SigningMethod
	switch m.cfg.Alg {
	case AlgHS256:
		method = jwt.SigningMethodHS256
	case AlgRS256:
		method = jwt.SigningMethodRS256
	default:
		return "", fmt.Errorf("jwt: unsupported alg: %s", m.cfg.Alg)
	}

	t := jwt.NewWithClaims(method, c)

	switch m.cfg.Alg {
	case AlgHS256:
		return t.SignedString(m.cfg.HMACSecret)
	case AlgRS256:
		return t.SignedString(m.cfg.RSAPrivateKey)
	default:
		return "", fmt.Errorf("jwt: unsupported alg: %s", m.cfg.Alg)
	}
}

// =====================
// 验证：实现 contracts/auth.AppJWTVerifier
// =====================

// ParseAccessToken 仅负责 access token 解析与 claims 基础校验。
// 它不依赖 Gin，也不向 middleware.AuthClaims 做任何映射。
func (m *Manager) ParseAccessToken(token string) (contractsauth.AppClaims, error) {
	claims := &Claims{}

	opts := []jwt.ParserOption{
		jwt.WithLeeway(m.cfg.Leeway),
		jwt.WithIssuer(m.cfg.Issuer),
	}
	if m.cfg.Audience != "" {
		opts = append(opts, jwt.WithAudience(m.cfg.Audience))
	}

	parsed, err := jwt.ParseWithClaims(token, claims, m.keyFunc(), opts...)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if !parsed.Valid {
		return nil, ErrUnauthorized
	}
	if strings.TrimSpace(claims.UID) == "" || strings.TrimSpace(claims.TenantID) == "" {
		return nil, ErrUnauthorized
	}
	if tokenType := claims.GetTokenType(); tokenType != "" && tokenType != "access" {
		return nil, ErrInvalidTokenType
	}
	if claims.GetTokenType() == "" {
		claims.TokenType = "access"
	}
	return claims, nil
}

func (m *Manager) keyFunc() jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) {
		// 限制算法，防止 alg none / 算法混淆攻击
		switch m.cfg.Alg {
		case AlgHS256:
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, errors.New("jwt: unexpected signing method")
			}
			return m.cfg.HMACSecret, nil
		case AlgRS256:
			if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, errors.New("jwt: unexpected signing method")
			}
			return m.cfg.RSAPublicKey, nil
		default:
			return nil, errors.New("jwt: unsupported alg")
		}
	}
}
