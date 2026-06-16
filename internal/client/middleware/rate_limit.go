package middleware

import (
	"context"
	"errors"
	"fmt"
	"storeready_ai/internal/security"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// 限流命中时的安全事件（轻量结构，后续你可以接到落库/落队列/告警）
type RateLimitSecurityEvent struct {
	Type      string `json:"type"`      // 固定：rate_limit_hit
	RID       string `json:"rid"`       // request id
	IP        string `json:"ip"`        // 客户端 IP
	UID       string `json:"uid"`       // 登录用户 ID（可空）
	Route     string `json:"route"`     // gin.FullPath 或 URL.Path
	Method    string `json:"method"`    // HTTP 方法
	Key       string `json:"key"`       // Redis key
	Capacity  int64  `json:"capacity"`  // 桶容量
	Rate      int64  `json:"rate"`      // 每分钟放行速率（rpm）
	Remaining int64  `json:"remaining"` // 估算剩余令牌
}

// RateLimitConfig 全站基础限流配置。
//
// 设计目标：
//   - key：未登录 route+method+ip；登录 route+uid
//   - 算法：Redis + Lua 的令牌桶（原子）
//   - 命中：429 + 业务码 + 可挂钩安全事件
//
// 重要提示：
//   - 令牌桶只约束“请求放行频率”，并不能阻止不尊重 ctx 的 CPU 密集逻辑；
//     真正抗压还要配合 Timeout、下游 ctx、队列化等。
type RateLimitConfig struct {
	// Redis 用于存储桶状态。
	// 推荐使用 redis.UniversalClient（支持单机/哨兵/集群）。
	Redis redis.UniversalClient

	// KeyPrefix Redis key 前缀，默认："rl:"。
	KeyPrefix string

	// 公共 GET：每 IP 每分钟限额（rpm）。默认：120。
	PublicGetRPM int64

	// 写接口：每 IP 每分钟限额（rpm）。默认：30。
	WriteRPM int64

	// 登录态：每 UID 每分钟限额（rpm）。默认：240。
	AuthedRPM int64

	// UserIDResolver 用于从 gin.Context 取出 uid。
	// 返回 (uid, true) 表示已登录。
	// 默认：从 ctx.Get("uid") 读取（支持 string / int64 / uint64 / int）。
	UserIDResolver func(c *gin.Context) (string, bool)

	// RouteKeyResolver 用于生成 route 字段。
	// 默认优先 c.FullPath()，为空则用 c.Request.URL.Path。
	RouteKeyResolver func(c *gin.Context) string

	// PerRouteRPM 针对特定 route（FullPath）覆盖限额。
	// key 建议写成："GET /v1/xxx" 或 "POST /v1/xxx" 或 "* /v1/xxx"。
	// 注意：登录态 key 规则不含 method，但这里为了好用仍允许按 method 覆盖。
	PerRouteRPM map[string]int64

	// OnLimitHit 命中限流时的回调（可在这里发 security event）。
	// 默认：空。
	OnLimitHit func(c *gin.Context, ev RateLimitSecurityEvent)

	// 响应业务码/提示。
	// 默认：code="rate_limited" msg="Too Many Requests"
	RespCode    string
	RespMessage string
	// SecurityEmitter 安全事件发射器（可选）。
	// 用于把 RateLimit 命中沉淀为安全事件，便于后续封禁/告警/BI。
	SecurityEmitter *security.SecurityEventEmitter
}

func (c *RateLimitConfig) withDefaults() RateLimitConfig {
	cfg := *c
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "rl:"
	}
	if cfg.PublicGetRPM <= 0 {
		cfg.PublicGetRPM = 120
	}
	if cfg.WriteRPM <= 0 {
		cfg.WriteRPM = 30
	}
	if cfg.AuthedRPM <= 0 {
		cfg.AuthedRPM = 240
	}
	if cfg.UserIDResolver == nil {
		cfg.UserIDResolver = defaultUIDResolver
	}
	if cfg.RouteKeyResolver == nil {
		cfg.RouteKeyResolver = defaultRouteResolver
	}
	if cfg.OnLimitHit == nil {
		cfg.OnLimitHit = func(_ *gin.Context, _ RateLimitSecurityEvent) {}
	}
	if cfg.RespCode == "" {
		cfg.RespCode = "rate_limited"
	}
	if cfg.RespMessage == "" {
		cfg.RespMessage = "Too Many Requests"
	}
	return cfg
}

// RateLimit 全站基础限流中间件。
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	if cfg.Redis == nil {
		panic("RateLimit: Redis 不能为空")
	}

	return func(c *gin.Context) {
		route := cfg.RouteKeyResolver(c)
		method := c.Request.Method
		ip := GetClientIP(c)
		rid := GetRequestID(c)

		uid, authed := cfg.UserIDResolver(c)

		// 计算本次请求 rpm（支持 per-route 覆盖）
		rpm := cfg.pickRPM(authed, method, route)
		if rpm <= 0 {
			// rpm<=0 表示不限制
			c.Next()
			return
		}

		// 组装 key
		var key string
		if authed {
			// 登录态：route + uid
			key = cfg.KeyPrefix + "u:" + safeKey(route) + ":" + safeKey(uid)
		} else {
			// 未登录：route + method + ip
			key = cfg.KeyPrefix + "i:" + safeKey(route) + ":" + safeKey(method) + ":" + safeKey(ip)
		}

		allowed, remaining, err := tokenBucketAllow(c.Request.Context(), cfg.Redis, key, rpm)
		if err != nil {
			// Redis 异常：为了可用性默认放行（也可改成拒绝/降级）。
			c.Next()
			return
		}
		if allowed {
			c.Next()
			return
		}

		// 命中限流：429 + security event
		ev := RateLimitSecurityEvent{
			Type:      "rate_limit_hit",
			RID:       rid,
			IP:        ip,
			UID:       uid,
			Route:     route,
			Method:    method,
			Key:       key,
			Capacity:  rpm,
			Rate:      rpm,
			Remaining: remaining,
		}
		cfg.OnLimitHit(c, ev)

		// 命中限流时，若配置了 SecurityEmitter，则发出统一安全事件
		emitRateLimitSecurityEvent(c, cfg.SecurityEmitter, key, rpm, remaining, authed, route, method)

		if !c.Writer.Written() {
			c.AbortWithStatusJSON(429, gin.H{
				"code": cfg.RespCode,
				"msg":  cfg.RespMessage,
				"rid":  rid,
			})
			return
		}
		c.Abort()
	}
}

func (cfg RateLimitConfig) pickRPM(authed bool, method, route string) int64 {
	// 先看 per-route 覆盖
	if cfg.PerRouteRPM != nil {
		if v, ok := cfg.PerRouteRPM[strings.ToUpper(method)+" "+route]; ok {
			return v
		}
		if v, ok := cfg.PerRouteRPM["* "+route]; ok {
			return v
		}
	}

	if authed {
		return cfg.AuthedRPM
	}
	// 未登录：按 GET/写接口区分
	if strings.EqualFold(method, httpMethodGET) {
		return cfg.PublicGetRPM
	}
	return cfg.WriteRPM
}

const httpMethodGET = "GET"

func defaultRouteResolver(c *gin.Context) string {
	p := c.FullPath()
	if p != "" {
		return p
	}
	if c.Request != nil && c.Request.URL != nil {
		if c.Request.URL.Path != "" {
			return c.Request.URL.Path
		}
	}
	return "/"
}

func defaultUIDResolver(c *gin.Context) (string, bool) {
	v, ok := c.Get("uid")
	if !ok || v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return "", false
		}
		return t, true
	case int:
		return strconv.FormatInt(int64(t), 10), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case uint64:
		return strconv.FormatUint(t, 10), true
	case uint:
		return strconv.FormatUint(uint64(t), 10), true
	default:
		return fmt.Sprint(t), true
	}
}

func safeKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	// 简单替换，避免出现空格/换行导致 key 混乱
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "\n", "_")
	s = strings.ReplaceAll(s, "\r", "_")
	return s
}

// ----------------------
// Redis + Lua：令牌桶
// ----------------------

// Lua 脚本说明（令牌桶）：
//   - 使用 Redis TIME 获取服务器时间（秒+微秒），避免客户端时钟漂移
//   - state 存在 hash：tokens、ts
//   - 每次请求：按时间差补充 tokens（rate=rpm/60），最大不超过 capacity（=rpm）
//   - tokens >= 1 则扣减并放行，否则拒绝
//   - 设置过期（2 分钟），避免 key 永久增长
var luaTokenBucket = redis.NewScript(`
-- KEYS[1] = key
-- ARGV[1] = capacity (number)
-- ARGV[2] = rate_per_sec (number)
-- ARGV[3] = ttl_ms

local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

local now = redis.call('TIME')
local now_sec = tonumber(now[1])
local now_usec = tonumber(now[2])
local now_ms = now_sec * 1000 + math.floor(now_usec / 1000)

local data = redis.call('HMGET', key, 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts = tonumber(data[2])

if tokens == nil then
  tokens = capacity
end
if ts == nil then
  ts = now_ms
end

local delta_ms = now_ms - ts
if delta_ms < 0 then
  delta_ms = 0
end

-- 补充令牌
local add = (delta_ms / 1000.0) * rate
if add > 0 then
  tokens = math.min(capacity, tokens + add)
  ts = now_ms
end

local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end

redis.call('HMSET', key, 'tokens', tokens, 'ts', ts)
redis.call('PEXPIRE', key, ttl)

-- remaining 取 floor，便于展示
return {allowed, math.floor(tokens)}
`)

func tokenBucketAllow(ctx context.Context, rdb redis.UniversalClient, key string, rpm int64) (bool, int64, error) {
	if rpm <= 0 {
		return true, 0, nil
	}
	// capacity 直接用 rpm，符合“每分钟最大突发≈rpm”
	capacity := float64(rpm)
	// 每秒补充 rate
	ratePerSec := float64(rpm) / 60.0
	// 过期 2 分钟，避免 key 膨胀
	ttlMs := int64(2 * time.Minute / time.Millisecond)

	res, err := luaTokenBucket.Run(ctx, rdb, []string{key}, capacity, ratePerSec, ttlMs).Result()
	if err != nil {
		return true, 0, err
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return true, 0, errors.New("rate limit lua 返回格式错误")
	}
	allowed := toInt64(arr[0]) == 1
	remaining := toInt64(arr[1])
	return allowed, remaining, nil
}

func toInt64(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		if t == "" {
			return 0
		}
		i, _ := strconv.ParseInt(t, 10, 64)
		return i
	case []byte:
		if len(t) == 0 {
			return 0
		}
		i, _ := strconv.ParseInt(string(t), 10, 64)
		return i
	default:
		return 0
	}
}

// emitRateLimitSecurityEvent 统一发射限流安全事件，便于后续封禁/告警/BI。
// 只有当 emitter 非空且（Writer 或 Queue 非空）时才真正发事件。
// details 包含 reason/key/rpm/remaining/authed/route/method/ip 等关键字段。
func emitRateLimitSecurityEvent(c *gin.Context, emitter *security.SecurityEventEmitter, key string, rpm int64, remaining int64, authed bool, route, method string) {
	if emitter == nil || (emitter.Writer == nil && emitter.Queue == nil) {
		return
	}
	details := map[string]any{
		"reason":    "rate_limited",
		"key":       key,
		"rpm":       rpm,
		"remaining": remaining,
		"authed":    authed,
		"route":     route,
		"method":    method,
	}
	// 兜底补齐常用字段
	if _, ok := details["ip"]; !ok {
		details["ip"] = GetClientIP(c)
	}
	_ = security.EmitSecurityEvent(c, emitter, "rate_limit_hit", security.SecuritySeverityLow, details)
}
