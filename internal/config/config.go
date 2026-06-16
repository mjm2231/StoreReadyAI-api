package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 根配置。
//
// 说明：
//   - 与 configs/*.yaml 对应（mapstructure 标签）
//   - time.Duration 会由 viper 自动解析（例如 5s/30m/200ms）
//   - 这里仅做“轻量校验”，避免明显错误配置

type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	DB            DBConfig            `mapstructure:"db"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Firebase      FirebaseConfig      `mapstructure:"firebase"`
	Billing       BillingConfig       `mapstructure:"billing"`
	Security      SecurityConfig      `mapstructure:"security"`
	AntiAbuse     AntiAbuseConfig     `mapstructure:"anti_abuse"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

// -------------------------
// server
// -------------------------

type ServerConfig struct {
	Name            string        `mapstructure:"name"`
	Mode            string        `mapstructure:"mode"` // debug|release
	Listen          string        `mapstructure:"listen"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	GracefulTimeout time.Duration `mapstructure:"graceful_timeout"`

	// PublicBaseURL 服务对外基准地址（用于生成回调链接等；可选）
	PublicBaseURL string `mapstructure:"public_base_url"`

	// Pprof 性能分析（仅建议内网/调试开启）
	Pprof PprofConfig `mapstructure:"pprof"`

	HTTP HTTPConfig `mapstructure:"http"`
}

// PprofConfig pprof 配置。
type PprofConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	PathPrefix string `mapstructure:"path_prefix"`
}

// HTTPConfig HTTP/网关层配置（建议与中间件策略保持一致）。
type HTTPConfig struct {
	RealIP    HTTPRealIPConfig    `mapstructure:"real_ip"`
	Limits    HTTPLimitsConfig    `mapstructure:"limits"`
	Timeout   HTTPTimeoutConfig   `mapstructure:"timeout"`
	CORS      HTTPCORSConfig      `mapstructure:"cors"`
	Firewall  HTTPFirewallConfig  `mapstructure:"firewall"`
	RateLimit HTTPRateLimitConfig `mapstructure:"rate_limit"`
	AntiBrush HTTPAntiBrushConfig `mapstructure:"anti_brush"`
}

// HTTPRealIPConfig 真实客户端 IP（配合 RealIP 中间件）。
type HTTPRealIPConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	TrustedProxies []string `mapstructure:"trusted_proxies"`
	Headers        []string `mapstructure:"headers"`
}

// HTTPLimitsConfig 基础限制。
type HTTPLimitsConfig struct {
	MaxHeaderBytes    int64         `mapstructure:"max_header_bytes"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	MaxConns          int           `mapstructure:"max_conns"`
	MaxBodyBytes      int64         `mapstructure:"max_body_bytes"`
	MaxBodyBytesLarge int64         `mapstructure:"max_body_bytes_large"`
	LargePathPrefixes []string      `mapstructure:"large_path_prefixes"`
}

// HTTPTimeoutConfig 请求级超时（建议与 Timeout 中间件一致）。
type HTTPTimeoutConfig struct {
	Default           time.Duration            `mapstructure:"default"`
	UseGatewayTimeout bool                     `mapstructure:"use_gateway_timeout"`
	PerRoute          map[string]time.Duration `mapstructure:"per_route"`
}

// HTTPCORSConfig CORS 严格白名单。
type HTTPCORSConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	AllowCredentials bool          `mapstructure:"allow_credentials"`
	AllowMethods     []string      `mapstructure:"allow_methods"`
	AllowHeaders     []string      `mapstructure:"allow_headers"`
	ExposeHeaders    []string      `mapstructure:"expose_headers"`
	MaxAge           time.Duration `mapstructure:"max_age"`
	AllowOrigins     []string      `mapstructure:"allow_origins"`
}

// HTTPFirewallConfig 轻量 WAF。
type HTTPFirewallConfig struct {
	Enabled             bool     `mapstructure:"enabled"`
	AllowMethods        []string `mapstructure:"allow_methods"`
	BlockPathSubstrings []string `mapstructure:"block_path_substrings"`
	RejectEmptyUA       bool     `mapstructure:"reject_empty_ua"`
	RejectEmptyReferer  bool     `mapstructure:"reject_empty_referer"`
	BlockUAKeywords     []string `mapstructure:"block_ua_keywords"`
	JSONPathPrefixes    []string `mapstructure:"json_path_prefixes"`
	IPBlocklist         []string `mapstructure:"ip_blocklist"`
	IPGraylist          []string `mapstructure:"ip_graylist"`
}

// HTTPRateLimitConfig 全站基础限流。
type HTTPRateLimitConfig struct {
	Enabled      bool             `mapstructure:"enabled"`
	PublicGetRPM int64            `mapstructure:"public_get_rpm"`
	WriteRPM     int64            `mapstructure:"write_rpm"`
	AuthedRPM    int64            `mapstructure:"authed_rpm"`
	PerRoute     map[string]int64 `mapstructure:"per_route"`
}

// HTTPAntiBrushConfig 敏感接口 AntiBrush。
type HTTPAntiBrushConfig struct {
	Enabled bool  `mapstructure:"enabled"`
	Rules   []any `mapstructure:"rules"`
}

// -------------------------
// db / redis
// -------------------------

type DBConfig struct {
	Driver          string        `mapstructure:"driver"`
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	SlowThreshold   time.Duration `mapstructure:"slow_threshold"`
}

type RedisConfig struct {
	Addr         string        `mapstructure:"addr"`
	Username     string        `mapstructure:"username"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// -------------------------
// firebase
// -------------------------

// FirebaseConfig Firebase 配置（用于第三方登录验签 / Admin SDK）。
type FirebaseConfig struct {
	ProjectID       string             `mapstructure:"project_id"`
	CredentialsFile string             `mapstructure:"credentials_file"`
	ProxyURL        string             `mapstructure:"proxy_url"`
	Auth            FirebaseAuthConfig `mapstructure:"auth"`
}

// FirebaseAuthConfig Firebase Auth 能力开关。
type FirebaseAuthConfig struct {
	Enabled   bool                  `mapstructure:"enabled"`
	Providers []string              `mapstructure:"providers"`
	IDToken   FirebaseIDTokenConfig `mapstructure:"id_token"`
}

// FirebaseIDTokenConfig ID Token 验签相关配置。
type FirebaseIDTokenConfig struct {
	Leeway time.Duration `mapstructure:"leeway"`
}

// -------------------------
// billing
// -------------------------

// BillingConfig Billing 配置。
//
// 说明：
// 1. 当前用于 Google Play / App Store 支付校验相关配置；
// 2. default_entitlement_code 供 billing service 默认刷新会员权益时使用；
// 3. Google / Apple 平台配置后续可继续扩展。
type BillingConfig struct {
	Enabled                bool                    `mapstructure:"enabled"`
	DefaultEntitlementCode string                  `mapstructure:"default_entitlement_code"`
	GooglePlay             BillingGooglePlayConfig `mapstructure:"google_play"`
	AppStore               BillingAppStoreConfig   `mapstructure:"app_store"`
}

// BillingGooglePlayConfig Google Play Billing 配置。
type BillingGooglePlayConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	PackageName     string   `mapstructure:"package_name"`
	CredentialsFile string   `mapstructure:"credentials_file"`
	Scopes          []string `mapstructure:"scopes"`
}

// BillingAppStoreConfig App Store Billing 配置。
type BillingAppStoreConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	IssuerID       string `mapstructure:"issuer_id"`
	KeyID          string `mapstructure:"key_id"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	BundleID       string `mapstructure:"bundle_id"`
	Environment    string `mapstructure:"environment"`
}

// -------------------------
// security / anti_abuse / observability
// -------------------------

type SecurityConfig struct {
	JWTSecret string `mapstructure:"jwt_secret"`
	SignKey   string `mapstructure:"sign_key"`

	// JWT 企业级建议配置
	JWT      JWTConfig `mapstructure:"jwt"`
	AdminJWT JWTConfig `mapstructure:"admin_jwt"`

	// TokenVersion：用于强制失效旧 token（改密/踢下线/登出全部设备）
	TokenVersion TokenVersionConfig `mapstructure:"token_version"`

	AllowedOrigins []string      `mapstructure:"allowed_origins"`
	IPWhitelist    []string      `mapstructure:"ip_whitelist"`
	IPBlacklist    []string      `mapstructure:"ip_blacklist"`
	Captcha        CaptchaConfig `mapstructure:"captcha"`
}

// JWTConfig JWT 建议配置。
type JWTConfig struct {
	// Alg 算法：HS256（共享密钥）/ RS256（公私钥）
	Alg string `mapstructure:"alg"`

	Issuer   string `mapstructure:"issuer"`
	Audience string `mapstructure:"audience"`

	// Leeway 时钟漂移容忍（用于 nbf/iat/exp 校验）
	Leeway time.Duration `mapstructure:"leeway"`

	// HMACSecret HS256 共享密钥（建议作为唯一来源；为空可回退到 SecurityConfig.JWTSecret）
	HMACSecret string `mapstructure:"hmac_secret"`

	// RS256：公私钥路径（PEM）。当 Alg=RS256 时必须配置
	PublicKeyPath  string `mapstructure:"public_key_path"`
	PrivateKeyPath string `mapstructure:"private_key_path"`

	AccessTTL  time.Duration `mapstructure:"access_ttl"`
	RefreshTTL time.Duration `mapstructure:"refresh_ttl"`

	// RefreshMode：opaque|jwt
	RefreshMode string `mapstructure:"refresh_mode"`
}

// TokenVersionConfig TokenVersion 配置。
type TokenVersionConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type CaptchaConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Provider  string `mapstructure:"provider"` // turnstile|recaptcha
	SiteKey   string `mapstructure:"site_key"`
	SecretKey string `mapstructure:"secret_key"`
}

type AntiAbuseConfig struct {
	Enabled bool              `mapstructure:"enabled"`
	Primary string            `mapstructure:"primary"` // rate_limit|anti_abuse
	Global  AntiAbuseGlobal   `mapstructure:"global"`
	Login   AntiAbuseEndpoint `mapstructure:"login"`
	SMS     AntiAbuseEndpoint `mapstructure:"sms"`
	Routes  []AntiAbuseRoute  `mapstructure:"routes"`
}

type AntiAbuseGlobal struct {
	Rate       int    `mapstructure:"rate"` // rps
	Burst      int    `mapstructure:"burst"`
	KeyBy      string `mapstructure:"key_by"` // ip|real_ip|header
	HeaderName string `mapstructure:"header_name"`
}

type AntiAbuseEndpoint struct {
	Rate   int           `mapstructure:"rate"`
	Burst  int           `mapstructure:"burst"`
	Window time.Duration `mapstructure:"window"`
}

type AntiAbuseRoute struct {
	Method string        `mapstructure:"method"`
	Path   string        `mapstructure:"path"`
	Rate   int           `mapstructure:"rate"`
	Burst  int           `mapstructure:"burst"`
	Window time.Duration `mapstructure:"window"`
}

type ObservabilityConfig struct {
	Log     ObservabilityLogConfig     `mapstructure:"log"`
	FileLog ObservabilityFileLogConfig `mapstructure:"file_log"`
	Metrics ObservabilityMetricsConfig `mapstructure:"metrics"`
	Tracing ObservabilityTracingConfig `mapstructure:"tracing"`
}

type ObservabilityLogConfig struct {
	Level string `mapstructure:"level"` // debug|info|warn|error
	JSON  bool   `mapstructure:"json"`
}

type ObservabilityFileLogConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Dir     string `mapstructure:"dir"`
}

type ObservabilityMetricsConfig struct {
	Enabled bool                           `mapstructure:"enabled"`
	Path    string                         `mapstructure:"path"`
	Auth    ObservabilityMetricsAuthConfig `mapstructure:"auth"`
}

type ObservabilityMetricsAuthConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type ObservabilityTracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	Exporter    string  `mapstructure:"exporter"` // otlp|jaeger
	Endpoint    string  `mapstructure:"endpoint"`
	Insecure    bool    `mapstructure:"insecure"`
	SampleRatio float64 `mapstructure:"sample_ratio"`
}

// -------------------------
// Load / defaults / validate
// -------------------------

// Load 加载配置。
//
// cfgPath 为空时默认读取：configs/dev.yaml
func Load(cfgPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	path := strings.TrimSpace(cfgPath)
	if path == "" {
		path = filepath.Join("configs", "dev.yaml")
	}

	v.SetConfigFile(path)

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}

	if err := validate(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func setDefaults(v *viper.Viper) {
	// server 基础
	v.SetDefault("server.name", "storeready_ai")
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.listen", ":8080")
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.graceful_timeout", "10s")

	v.SetDefault("server.public_base_url", "http://localhost:8080")
	v.SetDefault("server.pprof.enabled", false)
	v.SetDefault("server.pprof.path_prefix", "/debug/pprof")

	// server.http
	v.SetDefault("server.http.real_ip.enabled", true)
	v.SetDefault("server.http.real_ip.trusted_proxies", []string{})
	v.SetDefault("server.http.real_ip.headers", []string{"X-Forwarded-For", "X-Real-IP"})

	v.SetDefault("server.http.limits.max_header_bytes", int64(1048576))
	v.SetDefault("server.http.limits.read_header_timeout", "2s")
	v.SetDefault("server.http.limits.max_conns", 0)
	v.SetDefault("server.http.limits.max_body_bytes", int64(2097152))
	v.SetDefault("server.http.limits.max_body_bytes_large", int64(20971520))
	v.SetDefault("server.http.limits.large_path_prefixes", []string{"/v1/files/upload", "/v1/reports/export"})

	v.SetDefault("server.http.timeout.default", "5s")
	v.SetDefault("server.http.timeout.use_gateway_timeout", true)
	v.SetDefault("server.http.timeout.per_route", map[string]any{
		"POST /v1/files/upload":  "30s",
		"GET /v1/reports/export": "60s",
	})

	v.SetDefault("server.http.cors.enabled", true)
	v.SetDefault("server.http.cors.allow_credentials", true)
	v.SetDefault("server.http.cors.allow_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	v.SetDefault("server.http.cors.allow_headers", []string{"Content-Type", "Authorization", "X-Request-Id", "X-Trace-Id", "X-Tenant-Id"})
	v.SetDefault("server.http.cors.expose_headers", []string{"X-Request-Id", "X-Trace-Id"})
	v.SetDefault("server.http.cors.max_age", "12h")
	v.SetDefault("server.http.cors.allow_origins", []string{"http://localhost:5173", "http://127.0.0.1:5173"})

	v.SetDefault("server.http.firewall.enabled", true)
	v.SetDefault("server.http.firewall.allow_methods", []string{"GET", "POST", "PUT", "PATCH", "DELETE"})
	v.SetDefault("server.http.firewall.block_path_substrings", []string{"..", "//", "\\"})
	v.SetDefault("server.http.firewall.reject_empty_ua", false)
	v.SetDefault("server.http.firewall.reject_empty_referer", false)
	v.SetDefault("server.http.firewall.block_ua_keywords", []string{"sqlmap", "nikto", "acunetix", "netsparker", "masscan", "nmap", "zgrab", "whatweb", "wpscan", "fuzz", "dirbuster", "gobuster"})
	v.SetDefault("server.http.firewall.json_path_prefixes", []string{"/v1", "/admin", "/auth"})
	v.SetDefault("server.http.firewall.ip_blocklist", []string{})
	v.SetDefault("server.http.firewall.ip_graylist", []string{})

	v.SetDefault("server.http.rate_limit.enabled", true)
	v.SetDefault("server.http.rate_limit.public_get_rpm", 120)
	v.SetDefault("server.http.rate_limit.write_rpm", 30)
	v.SetDefault("server.http.rate_limit.authed_rpm", 240)
	v.SetDefault("server.http.rate_limit.per_route", map[string]any{})

	v.SetDefault("server.http.anti_brush.enabled", true)
	v.SetDefault("server.http.anti_brush.rules", []any{})

	// security.jwt / token_version
	v.SetDefault("security.jwt.alg", "HS256")
	v.SetDefault("security.jwt.issuer", "storeready_ai")
	v.SetDefault("security.jwt.audience", "storeready_ai_client")
	v.SetDefault("security.jwt.leeway", "30s")
	v.SetDefault("security.jwt.hmac_secret", "")
	v.SetDefault("security.jwt.public_key_path", "")
	v.SetDefault("security.jwt.private_key_path", "")
	v.SetDefault("security.jwt.access_ttl", "15m")
	v.SetDefault("security.jwt.refresh_ttl", "720h")
	v.SetDefault("security.jwt.refresh_mode", "opaque")
	v.SetDefault("security.admin_jwt.alg", "HS256")
	v.SetDefault("security.admin_jwt.issuer", "storeready_ai_admin")
	v.SetDefault("security.admin_jwt.audience", "storeready_ai_admin")
	v.SetDefault("security.admin_jwt.leeway", "30s")
	v.SetDefault("security.admin_jwt.hmac_secret", "")
	v.SetDefault("security.admin_jwt.public_key_path", "")
	v.SetDefault("security.admin_jwt.private_key_path", "")
	v.SetDefault("security.admin_jwt.access_ttl", "2h")
	v.SetDefault("security.admin_jwt.refresh_ttl", "720h")
	v.SetDefault("security.admin_jwt.refresh_mode", "jwt")
	v.SetDefault("security.token_version.enabled", true)

	// db
	v.SetDefault("db.driver", "mysql")
	v.SetDefault("db.max_open_conns", 50)
	v.SetDefault("db.max_idle_conns", 10)
	v.SetDefault("db.conn_max_lifetime", "30m")
	v.SetDefault("db.conn_max_idle_time", "10m")
	v.SetDefault("db.slow_threshold", "200ms")

	// redis
	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 30)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.dial_timeout", "2s")
	v.SetDefault("redis.read_timeout", "1s")
	v.SetDefault("redis.write_timeout", "1s")

	// firebase
	v.SetDefault("firebase.project_id", "")
	v.SetDefault("firebase.credentials_file", "")
	v.SetDefault("firebase.proxy_url", "")
	v.SetDefault("firebase.auth.enabled", true)
	v.SetDefault("firebase.auth.providers", []string{"google", "apple"})
	v.SetDefault("firebase.auth.id_token.leeway", "60s")

	// billing
	v.SetDefault("billing.enabled", true)
	v.SetDefault("billing.default_entitlement_code", "vip")
	v.SetDefault("billing.google_play.enabled", true)
	v.SetDefault("billing.google_play.package_name", "")
	v.SetDefault("billing.google_play.credentials_file", "")
	v.SetDefault("billing.google_play.scopes", []string{})
	v.SetDefault("billing.app_store.enabled", false)
	v.SetDefault("billing.app_store.issuer_id", "")
	v.SetDefault("billing.app_store.key_id", "")
	v.SetDefault("billing.app_store.private_key_path", "")
	v.SetDefault("billing.app_store.bundle_id", "")
	v.SetDefault("billing.app_store.environment", "sandbox")

	// anti_abuse
	v.SetDefault("anti_abuse.primary", "rate_limit")

	// observability
	v.SetDefault("observability.log.level", "debug")
	v.SetDefault("observability.log.json", false)
	v.SetDefault("observability.file_log.enabled", false)
	v.SetDefault("observability.file_log.dir", "./logs")
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.path", "/metrics")
	v.SetDefault("observability.metrics.auth.enabled", false)
	v.SetDefault("observability.metrics.auth.username", "")
	v.SetDefault("observability.metrics.auth.password", "")
	v.SetDefault("observability.tracing.enabled", false)
	v.SetDefault("observability.tracing.exporter", "otlp")
	v.SetDefault("observability.tracing.insecure", true)
	v.SetDefault("observability.tracing.sample_ratio", 0.1)
}

func validate(c *Config) error {
	if c == nil {
		return errors.New("config: nil")
	}

	// server
	mode := strings.ToLower(strings.TrimSpace(c.Server.Mode))
	if mode != "debug" && mode != "release" {
		return fmt.Errorf("config: server.mode must be debug|release")
	}
	if strings.TrimSpace(c.Server.Listen) == "" {
		return errors.New("config: server.listen required")
	}
	if c.Server.ReadTimeout <= 0 {
		return errors.New("config: server.read_timeout must be > 0")
	}
	if c.Server.WriteTimeout <= 0 {
		return errors.New("config: server.write_timeout must be > 0")
	}
	if c.Server.IdleTimeout <= 0 {
		return errors.New("config: server.idle_timeout must be > 0")
	}
	if c.Server.GracefulTimeout <= 0 {
		return errors.New("config: server.graceful_timeout must be > 0")
	}

	// server.pprof
	if c.Server.Pprof.Enabled {
		if strings.TrimSpace(c.Server.Pprof.PathPrefix) == "" {
			return errors.New("config: server.pprof.path_prefix required when pprof enabled")
		}
	}

	// server.http（轻量校验，避免明显错误配置）
	if c.Server.HTTP.Timeout.Default <= 0 {
		return errors.New("config: server.http.timeout.default must be > 0")
	}
	if c.Server.HTTP.Limits.MaxHeaderBytes < 0 || c.Server.HTTP.Limits.MaxBodyBytes < 0 || c.Server.HTTP.Limits.MaxBodyBytesLarge < 0 {
		return errors.New("config: server.http.limits max_* must be >= 0")
	}
	if c.Server.HTTP.Limits.MaxBodyBytesLarge > 0 && c.Server.HTTP.Limits.MaxBodyBytes > 0 {
		if c.Server.HTTP.Limits.MaxBodyBytesLarge < c.Server.HTTP.Limits.MaxBodyBytes {
			return errors.New("config: server.http.limits.max_body_bytes_large must be >= max_body_bytes")
		}
	}
	if c.Server.HTTP.Limits.ReadHeaderTimeout < 0 {
		return errors.New("config: server.http.limits.read_header_timeout must be >= 0")
	}
	if c.Server.HTTP.Limits.MaxConns < 0 {
		return errors.New("config: server.http.limits.max_conns must be >= 0")
	}
	if c.Server.HTTP.CORS.Enabled {
		if len(c.Server.HTTP.CORS.AllowOrigins) == 0 {
			return errors.New("config: server.http.cors.allow_origins required when cors enabled")
		}
	}

	// security.jwt
	if c.Security.JWT.AccessTTL < 0 || c.Security.JWT.RefreshTTL < 0 {
		return errors.New("config: security.jwt ttl must be >= 0")
	}
	if c.Security.JWT.Leeway < 0 {
		return errors.New("config: security.jwt.leeway must be >= 0")
	}

	alg := strings.ToUpper(strings.TrimSpace(c.Security.JWT.Alg))
	if alg == "" {
		alg = "HS256"
	}
	if alg != "HS256" && alg != "RS256" {
		return errors.New("config: security.jwt.alg must be HS256|RS256")
	}

	// refresh_mode
	rm := strings.ToLower(strings.TrimSpace(c.Security.JWT.RefreshMode))
	if rm == "" {
		rm = "opaque"
	}
	if rm != "opaque" && rm != "jwt" {
		return errors.New("config: security.jwt.refresh_mode must be opaque|jwt")
	}

	// RS256 key paths required when Alg=RS256
	if alg == "RS256" {
		if strings.TrimSpace(c.Security.JWT.PublicKeyPath) == "" || strings.TrimSpace(c.Security.JWT.PrivateKeyPath) == "" {
			return errors.New("config: security.jwt public_key_path/private_key_path required when alg=RS256")
		}
	}

	// HS256 secret: allow empty in dev if legacy security.jwt_secret is set; bootstrap should decide final precedence
	if alg == "HS256" {
		if strings.TrimSpace(c.Security.JWT.HMACSecret) == "" && strings.TrimSpace(c.Security.JWTSecret) == "" {
			return errors.New("config: security.jwt.hmac_secret or security.jwt_secret required when alg=HS256")
		}
	}

	// security.admin_jwt
	if c.Security.AdminJWT.AccessTTL < 0 || c.Security.AdminJWT.RefreshTTL < 0 {
		return errors.New("config: security.admin_jwt ttl must be >= 0")
	}
	if c.Security.AdminJWT.Leeway < 0 {
		return errors.New("config: security.admin_jwt.leeway must be >= 0")
	}

	adminAlg := strings.ToUpper(strings.TrimSpace(c.Security.AdminJWT.Alg))
	if adminAlg == "" {
		adminAlg = "HS256"
	}
	if adminAlg != "HS256" && adminAlg != "RS256" {
		return errors.New("config: security.admin_jwt.alg must be HS256|RS256")
	}

	adminRM := strings.ToLower(strings.TrimSpace(c.Security.AdminJWT.RefreshMode))
	if adminRM == "" {
		adminRM = "jwt"
	}
	if adminRM != "opaque" && adminRM != "jwt" {
		return errors.New("config: security.admin_jwt.refresh_mode must be opaque|jwt")
	}

	if adminAlg == "RS256" {
		if strings.TrimSpace(c.Security.AdminJWT.PublicKeyPath) == "" || strings.TrimSpace(c.Security.AdminJWT.PrivateKeyPath) == "" {
			return errors.New("config: security.admin_jwt public_key_path/private_key_path required when alg=RS256")
		}
	}

	if adminAlg == "HS256" {
		if strings.TrimSpace(c.Security.AdminJWT.HMACSecret) == "" {
			return errors.New("config: security.admin_jwt.hmac_secret required when alg=HS256")
		}
	}

	// issuer/audience 不强制（开发环境可空）；生产环境可在后续加 stricter 校验

	// db
	if strings.TrimSpace(c.DB.Driver) == "" {
		return errors.New("config: db.driver required")
	}
	if strings.TrimSpace(c.DB.DSN) == "" {
		return errors.New("config: db.dsn required")
	}
	if c.DB.MaxOpenConns < 0 || c.DB.MaxIdleConns < 0 {
		return errors.New("config: db pool sizes must be >= 0")
	}

	// redis
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return errors.New("config: redis.addr required")
	}

	// firebase（仅做轻量校验；生产环境可加 stricter 规则）
	if c.Firebase.Auth.Enabled {
		if strings.TrimSpace(c.Firebase.ProjectID) == "" {
			return errors.New("config: firebase.project_id required when firebase.auth.enabled")
		}
		if strings.TrimSpace(c.Firebase.CredentialsFile) == "" {
			return errors.New("config: firebase.credentials_file required when firebase.auth.enabled")
		}
		// providers 允许为空（表示全开/不做限制），但建议至少配置一个
		if c.Firebase.Auth.IDToken.Leeway < 0 {
			return errors.New("config: firebase.auth.id_token.leeway must be >= 0")
		}
	}

	// billing（仅做轻量校验；允许未启用时留空）
	if c.Billing.Enabled {
		if strings.TrimSpace(c.Billing.DefaultEntitlementCode) == "" {
			return errors.New("config: billing.default_entitlement_code required when billing.enabled")
		}
		if c.Billing.GooglePlay.Enabled {
			if strings.TrimSpace(c.Billing.GooglePlay.PackageName) == "" {
				return errors.New("config: billing.google_play.package_name required when billing.google_play.enabled")
			}
			if strings.TrimSpace(c.Billing.GooglePlay.CredentialsFile) == "" {
				return errors.New("config: billing.google_play.credentials_file required when billing.google_play.enabled")
			}
		}
		if c.Billing.AppStore.Enabled {
			env := strings.ToLower(strings.TrimSpace(c.Billing.AppStore.Environment))
			if env != "sandbox" && env != "production" {
				return errors.New("config: billing.app_store.environment must be sandbox|production")
			}
		}
	}

	// anti_abuse.primary
	p := strings.TrimSpace(c.AntiAbuse.Primary)
	if p != "" && p != "rate_limit" && p != "anti_abuse" {
		return errors.New("config: anti_abuse.primary must be rate_limit|anti_abuse")
	}

	// observability.file_log
	if c.Observability.FileLog.Enabled {
		if strings.TrimSpace(c.Observability.FileLog.Dir) == "" {
			return errors.New("config: observability.file_log.dir required when file_log enabled")
		}
	}
	// observability.metrics.auth
	if c.Observability.Metrics.Auth.Enabled {
		if strings.TrimSpace(c.Observability.Metrics.Auth.Username) == "" || strings.TrimSpace(c.Observability.Metrics.Auth.Password) == "" {
			return errors.New("config: observability.metrics.auth username/password required when auth enabled")
		}
	}

	return nil
}
