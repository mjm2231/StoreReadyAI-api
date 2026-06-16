package middleware

import (
	"storeready_ai/internal/infra/hander"
	"storeready_ai/internal/security"

	"github.com/gin-gonic/gin"
)

func boolPtr(v bool) *bool { return &v }

func boolVal(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// BootstrapConfig 用于一次性组装全套企业级中间件。
//
// 设计目标：
//   - 依赖集中注入：SecurityEventEmitter、AuditWriter 等不在业务代码里散落
//   - 顺序固定可控：确保 AccessLog/ Audit 能拿到完整信息
//   - 行为可配置：每个中间件都有独立 Config，可以按需覆盖
//
// 推荐顺序：
//  0. RealIP
//  1. RequestID
//     1.5 Tracing（链路追踪，注入 trace_id/span_id）
//  2. Recovery
//  3. Timeout
//  4. CORS
//  5. Firewall
//  6. RateLimit
//  7. AntiBrush
//  8. Auth
//  9. Audit
//  10. Metrics
//  11. AccessLog
//
// 注意：
//   - AccessLog 建议最后，确保拿到 status/cost/uid 等最终信息
//   - Metrics 通常放在 Audit 之后或之前都可；这里按你之前的顺序放在 Audit 之后
//
// 你可以按环境拆分：例如 admin 与 api 使用不同的 CORS/RateLimit/AntiBrush 规则。
type BootstrapConfig struct {
	// 统一依赖
	SecurityEmitter *security.SecurityEventEmitter

	// SecurityWriter 安全事件写入器（可选）。
	// 如果未显式提供 SecurityEmitter，或 emitter 中 Writer 为空，则会使用这里的默认 Writer。
	SecurityWriter security.SecurityEventWriter

	// SecurityQueue 安全事件队列发布器（可选）。
	// 如果未显式提供 SecurityEmitter，或 emitter 中 Queue 为空，则会使用这里的默认 Queue。
	SecurityQueue security.SecurityEventQueuePublisher

	// AuditWriter 审计日志写入器（可选）。
	// 如果未在 AuditConfig 中显式指定 Writer，则会使用这里的默认 Writer。
	AuditWriter AuditWriter

	// AuditQueue 审计日志队列发布器（可选）。
	// 如果未在 AuditConfig 中显式指定 Queue，则会使用这里的默认 Queue。
	AuditQueue AuditQueuePublisher

	// 逐个中间件配置（可覆盖）
	RealIP    RealIPConfig
	RequestID RequestIDConfig
	Tracing   TracingConfig
	Recovery  RecoveryConfig
	Timeout   TimeoutConfig
	CORS      CORSConfig
	Firewall  FirewallConfig
	RateLimit RateLimitConfig
	AntiBrush AntiBrushConfig
	// Auth 鉴权配置。
	//
	// 说明：
	// 1. 当前 BootstrapMiddlewares 默认挂载的是 AuthOptional(cfg.Auth)；
	// 2. 若在 AuthConfig 中配置了 UserIDResolver，则鉴权成功后会额外解析 user_id；
	// 3. 解析成功后，uid / tenant_id / user_id 会统一写入 gin.Context，供后续 handler / service 使用。
	Auth      AuthConfig
	Audit     AuditConfig
	Metrics   MetricsConfig
	AccessLog AccessLogConfig

	// 是否启用某个中间件（默认都启用；置为 false 表示跳过）
	EnableRealIP    *bool
	EnableRequestID *bool
	EnableTracing   *bool
	EnableRecovery  *bool
	EnableTimeout   *bool
	EnableCORS      *bool
	EnableFirewall  *bool
	EnableRateLimit *bool
	EnableAntiBrush *bool
	EnableAuth      *bool
	EnableAudit     *bool
	EnableMetrics   *bool
	EnableAccessLog *bool
}

func (c *BootstrapConfig) withDefaults() BootstrapConfig {
	cfg := *c

	// 默认全开（除非显式关闭）
	// 说明：EnableXXX 使用 *bool 三态：
	//   - nil  表示未配置 -> 默认 true
	//   - true 表示显式开启
	//   - false 表示显式关闭
	if cfg.EnableRealIP == nil {
		cfg.EnableRealIP = boolPtr(true)
	}
	if cfg.EnableRequestID == nil {
		cfg.EnableRequestID = boolPtr(true)
	}
	if cfg.EnableTracing == nil {
		cfg.EnableTracing = boolPtr(true)
	}
	if cfg.EnableRecovery == nil {
		cfg.EnableRecovery = boolPtr(true)
	}
	if cfg.EnableTimeout == nil {
		cfg.EnableTimeout = boolPtr(true)
	}
	if cfg.EnableCORS == nil {
		cfg.EnableCORS = boolPtr(true)
	}
	if cfg.EnableFirewall == nil {
		cfg.EnableFirewall = boolPtr(true)
	}
	if cfg.EnableRateLimit == nil {
		cfg.EnableRateLimit = boolPtr(true)
	}
	if cfg.EnableAntiBrush == nil {
		cfg.EnableAntiBrush = boolPtr(true)
	}
	if cfg.EnableAuth == nil {
		cfg.EnableAuth = boolPtr(true)
	}
	if cfg.EnableAudit == nil {
		cfg.EnableAudit = boolPtr(true)
	}
	if cfg.EnableMetrics == nil {
		cfg.EnableMetrics = boolPtr(true)
	}
	if cfg.EnableAccessLog == nil {
		cfg.EnableAccessLog = boolPtr(true)
	}

	// 统一组装/补齐 SecurityEmitter（可选）
	// 规则：
	//   - 如果显式提供了 cfg.SecurityEmitter：仅在其 Writer/Queue 为空时补齐默认值
	//   - 如果未提供 cfg.SecurityEmitter：当提供了 SecurityWriter 或 SecurityQueue 任一时，自动创建 emitter
	if cfg.SecurityEmitter != nil {
		if cfg.SecurityEmitter.Writer == nil && cfg.SecurityWriter != nil {
			cfg.SecurityEmitter.Writer = cfg.SecurityWriter
		}
		if cfg.SecurityEmitter.Queue == nil && cfg.SecurityQueue != nil {
			cfg.SecurityEmitter.Queue = cfg.SecurityQueue
		}
	} else {
		if cfg.SecurityWriter != nil || cfg.SecurityQueue != nil {
			cfg.SecurityEmitter = &security.SecurityEventEmitter{Writer: cfg.SecurityWriter, Queue: cfg.SecurityQueue}
		}
	}

	// 把统一的 SecurityEmitter 注入到需要的中间件中
	if cfg.SecurityEmitter != nil {
		// Firewall / RateLimit / AntiBrush / Recovery
		cfg.Firewall.SecurityEmitter = cfg.SecurityEmitter
		cfg.RateLimit.SecurityEmitter = cfg.SecurityEmitter
		cfg.AntiBrush.SecurityEmitter = cfg.SecurityEmitter
		cfg.Recovery.SecurityEmitter = cfg.SecurityEmitter
	}

	// 把统一的 AuditWriter 注入到 AuditConfig 中（仅在未显式配置时）
	if cfg.Audit.Writer == nil && cfg.AuditWriter != nil {
		cfg.Audit.Writer = cfg.AuditWriter
	}

	// 把统一的 AuditQueue 注入到 AuditConfig 中（仅在未显式配置时）
	if cfg.Audit.Queue == nil && cfg.AuditQueue != nil {
		cfg.Audit.Queue = cfg.AuditQueue
	}

	return cfg
}

// BootstrapMiddlewares 组装并返回中间件列表。
//
// 用法：
//
//	r.Use(middleware.BootstrapMiddlewares(middleware.BootstrapConfig{ ... })...)
func BootstrapMiddlewares(c BootstrapConfig) []gin.HandlerFunc {
	cfg := c.withDefaults()

	mws := make([]gin.HandlerFunc, 0, 13)

	if boolVal(cfg.EnableRealIP, true) {
		mws = append(mws, RealIP(cfg.RealIP))
	}
	if boolVal(cfg.EnableRequestID, true) {
		mws = append(mws, RequestID(cfg.RequestID))
	}
	if boolVal(cfg.EnableTracing, true) {
		mws = append(mws, Tracing(cfg.Tracing))
	}
	mws = append(mws, hander.HeaderInfoMiddleware()) // 统一注入 header 信息到 ctx，供后续中间件使用
	if boolVal(cfg.EnableRecovery, true) {
		mws = append(mws, Recovery(cfg.Recovery))
	}
	if boolVal(cfg.EnableTimeout, true) {
		mws = append(mws, Timeout(cfg.Timeout))
	}
	if boolVal(cfg.EnableCORS, true) {
		mws = append(mws, CORS(cfg.CORS))
	}
	if boolVal(cfg.EnableFirewall, true) {
		mws = append(mws, Firewall(cfg.Firewall))
	}
	if boolVal(cfg.EnableRateLimit, true) {
		mws = append(mws, RateLimit(cfg.RateLimit))
	}
	if boolVal(cfg.EnableAntiBrush, true) {
		mws = append(mws, AntiBrush(cfg.AntiBrush))
	}

	if boolVal(cfg.EnableAudit, true) {
		mws = append(mws, Audit(cfg.Audit))
	}
	if boolVal(cfg.EnableMetrics, true) {
		mws = append(mws, Metrics(cfg.Metrics))
	}
	if boolVal(cfg.EnableAccessLog, true) {
		mws = append(mws, AccessLog(cfg.AccessLog))
	}

	return mws
}
