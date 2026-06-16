package firebase

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
	"gorm.io/datatypes"

	usersvc "storeready_ai/internal/client/modules/user/service"
	"storeready_ai/internal/config"
)

// Client Firebase 客户端（Admin SDK）。
// 用途：后端校验客户端传来的 Firebase ID Token，抽取必要 claims 供业务登录使用。
//
// 设计原则：
// - 仅封装“验签 + 提取 claims”，避免把 Firebase SDK 泄漏到业务层。
// - 通过 interface（usersvc.FirebaseVerifier）对外暴露，便于后期替换/Mock。

type Client struct {
	cfg    config.FirebaseConfig
	authCl *auth.Client
}

// NewClient 初始化 Firebase Admin SDK。
// - project_id + credentials_file 必填（当 firebase.auth.enabled=true 时已在 config.validate 中校验）。
// - proxy_url 可选（用于解决部分环境访问 googleapis EOF/网络问题）。
func NewClient(ctx context.Context, cfg config.FirebaseConfig) (*Client, error) {
	opts := []option.ClientOption{}

	// credentials file
	if strings.TrimSpace(cfg.CredentialsFile) != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	// Optional HTTP client (proxy / timeout)
	hc, err := buildHTTPClient(cfg.ProxyURL)
	if err != nil {
		return nil, err
	}
	if hc != nil {
		opts = append(opts, option.WithHTTPClient(hc))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase: new app: %w", err)
	}

	authCl, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase: new auth client: %w", err)
	}

	return &Client{cfg: cfg, authCl: authCl}, nil
}

// VerifyIDToken 校验 Firebase ID Token 并抽取登录所需的字段。
// 返回 usersvc.FirebaseClaims（用于 UserService.FirebaseLogin）。
func (c *Client) VerifyIDToken(ctx context.Context, idToken string) (*usersvc.FirebaseClaims, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return nil, fmt.Errorf("firebase: empty id_token")
	}
	if c == nil || c.authCl == nil {
		return nil, fmt.Errorf("firebase: client not initialized")
	}

	// 1) 验签
	tok, err := c.authCl.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("firebase: verify id_token: %w", err)
	}

	// 2) 基本字段
	claims := &usersvc.FirebaseClaims{
		ProviderUID: tok.UID, // Firebase UID
	}

	// 3) firebase.sign_in_provider（不同端可能略有差异）
	if fb, ok := tok.Claims["firebase"].(map[string]any); ok {
		if sp, ok := fb["sign_in_provider"].(string); ok {
			claims.Provider = sp
		}
	}

	// 4) 常用 profile 字段（注意：Apple/匿名可能没有）
	if v, ok := tok.Claims["email"].(string); ok && strings.TrimSpace(v) != "" {
		vv := v
		claims.Email = &vv
	}
	if v, ok := tok.Claims["name"].(string); ok && strings.TrimSpace(v) != "" {
		vv := v
		claims.Name = &vv
	}
	if v, ok := tok.Claims["picture"].(string); ok && strings.TrimSpace(v) != "" {
		vv := v
		claims.Avatar = &vv
	}

	// 5) 原始 profile（可选：用于排查/扩展）
	// 这里做一个尽量“轻”的采集：把 token claims 的一部分序列化为 JSON。
	// 注意：datatypes.JSON 只是 []byte 的别名，业务层可按需存库。
	claims.RawProfile = buildRawProfileJSON(tok.Claims)

	// 6) provider 白名单校验（可配置开关/灰度）
	if c.cfg.Auth.Enabled && len(c.cfg.Auth.Providers) > 0 {
		allowed := make(map[string]struct{}, len(c.cfg.Auth.Providers))
		for _, p := range c.cfg.Auth.Providers {
			p = strings.TrimSpace(strings.ToLower(p))
			if p != "" {
				allowed[p] = struct{}{}
			}
		}
		// 将 firebase 返回的 provider 归一化：google.com -> google
		np := normalizeProviderForCheck(claims.Provider)
		if _, ok := allowed[np]; !ok {
			return nil, fmt.Errorf("firebase: provider not allowed: %s", np)
		}
	}

	// 7) leeway（时钟偏移）
	// Firebase Admin SDK VerifyIDToken 已做 exp/iat 校验。
	// 这里额外做一个“可选的宽松检查”：如果配置了 leeway，我们允许服务器时间轻微偏差。
	// 说明：Admin SDK 内部并未暴露 leeway 参数，这里仅做补充检查（不影响已验签通过的 token）。
	if c.cfg.Auth.IDToken.Leeway > 0 {
		// tok.Claims["exp"] 通常是 float64
		if exp, ok := tok.Claims["exp"].(float64); ok {
			expAt := time.Unix(int64(exp), 0)
			// 如果 token 过期时间距离现在小于 -leeway（也就是已过期超过 leeway），则拒绝
			if time.Since(expAt) > c.cfg.Auth.IDToken.Leeway {
				return nil, fmt.Errorf("firebase: id_token expired beyond leeway")
			}
		}
	}

	return claims, nil
}

// Ensure Client implements usersvc.FirebaseVerifier.
var _ usersvc.FirebaseVerifier = (*Client)(nil)

// -------------------------
// helpers
// -------------------------

func buildHTTPClient(proxyURL string) (*http.Client, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return nil, nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("firebase: invalid proxy_url: %w", err)
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	tr := &http.Transport{
		Proxy:       http.ProxyURL(u),
		DialContext: dialer.DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
	}

	return &http.Client{Transport: tr, Timeout: 25 * time.Second}, nil
}

func normalizeProviderForCheck(p string) string {
	p = strings.TrimSpace(strings.ToLower(p))
	switch p {
	case "google.com", "google":
		return "google"
	case "apple.com", "apple":
		return "apple"
	case "facebook.com", "facebook":
		return "facebook"
	case "anonymous":
		return "anonymous"
	default:
		// 兜底：原样返回
		return p
	}
}

func buildRawProfileJSON(claims map[string]any) datatypes.JSON {
	// 只取一部分常用字段，避免把 token 全量 claims（可能很大）都塞库。
	m := map[string]any{}
	for _, k := range []string{"email", "email_verified", "name", "picture", "firebase"} {
		if v, ok := claims[k]; ok {
			m[k] = v
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return datatypes.JSON(b)
}
