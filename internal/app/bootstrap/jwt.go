package bootstrap

import (
	"fmt"
	"strings"

	"storeready_ai/internal/client/auth/jwt"
	"storeready_ai/internal/config"
)

// NewJWTManager 从配置组装 JWT Manager。
// 职责边界：这里只做“依赖装配”（config -> jwt.Config），不涉及 gin/middleware/router。
// 兼容策略：
//   - HS256：优先使用 security.jwt.hmac_secret；为空则回退到 security.jwt_secret
//   - RS256：必须提供 public/private key 路径
func NewJWTManager(sec config.SecurityConfig) (*jwt.Manager, error) {
	alg := strings.ToUpper(strings.TrimSpace(sec.JWT.Alg))
	if alg == "" {
		alg = "HS256"
	}

	cfg := jwt.Config{
		Alg:       jwt.Algorithm(alg),
		Issuer:    strings.TrimSpace(sec.JWT.Issuer),
		Audience:  strings.TrimSpace(sec.JWT.Audience),
		Leeway:    sec.JWT.Leeway,
		AccessTTL: sec.JWT.AccessTTL,
	}

	switch alg {
	case "HS256":
		secret := strings.TrimSpace(sec.JWT.HMACSecret)
		if secret == "" {
			secret = strings.TrimSpace(sec.JWTSecret)
		}
		if secret == "" {
			return nil, fmt.Errorf("bootstrap: missing jwt secret (security.jwt.hmac_secret or security.jwt_secret)")
		}
		cfg.HMACSecret = []byte(secret)

	case "RS256":
		pubPath := strings.TrimSpace(sec.JWT.PublicKeyPath)
		priPath := strings.TrimSpace(sec.JWT.PrivateKeyPath)
		if pubPath == "" || priPath == "" {
			return nil, fmt.Errorf("bootstrap: missing rsa key paths (security.jwt.public_key_path/private_key_path)")
		}

		pub, err := jwt.LoadRSAPublicKeyFromPEMFile(pubPath)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: load jwt public key: %w", err)
		}
		pri, err := jwt.LoadRSAPrivateKeyFromPEMFile(priPath)
		if err != nil {
			return nil, fmt.Errorf("bootstrap: load jwt private key: %w", err)
		}
		cfg.RSAPublicKey = pub
		cfg.RSAPrivateKey = pri

	default:
		return nil, fmt.Errorf("bootstrap: unsupported jwt alg: %s", alg)
	}

	m, err := jwt.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: init jwt manager: %w", err)
	}
	return m, nil
}
