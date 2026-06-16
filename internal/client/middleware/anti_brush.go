package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"storeready_ai/internal/common"
	"storeready_ai/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AntiBrushAction 风控命中后的动作。
//
// 说明：
//   - 轻度：仅限流（返回 429）
//   - 中度：要求验证码/滑块（返回 429 + action=captcha_required，前端配合）
//   - 重度：短期封禁（返回 429 + action=blocked；可选写黑名单）
type AntiBrushAction string

const (
	AntiBrushActionAllow           AntiBrushAction = "allow"
	AntiBrushActionRateLimited     AntiBrushAction = "rate_limited"
	AntiBrushActionCaptchaRequired AntiBrushAction = "captcha_required"
	AntiBrushActionBlocked         AntiBrushAction = "blocked"
)

// AntiBrushEvent 命中风控时的安全事件（轻量结构，后续可接入落库/落队列/告警）。
type AntiBrushEvent struct {
	Type   string          `json:"type"`   // 固定：anti_brush_hit
	RID    string          `json:"rid"`    // request id
	IP     string          `json:"ip"`     // 客户端 IP
	UID    string          `json:"uid"`    // 登录用户 ID（可空）
	Route  string          `json:"route"`  // gin.FullPath 或 URL.Path
	Method string          `json:"method"` // HTTP 方法
	Action AntiBrushAction `json:"action"` // 采取的动作
	Score  int64           `json:"score"`  // 风险分
	// Reasons 风险原因列表（按贡献度排序，便于审计/安全事件）。
	Reasons []security.Reason `json:"reasons,omitempty"`
	Reason  string            `json:"reason"` // 命中原因（简述）
	Key     string            `json:"key"`    // 主要 Redis key（可空）
}

// AntiBrushIdentifiers 风控需要的标识。
type AntiBrushIdentifiers struct {
	IP       string
	UID      string
	DeviceID string
	Phone    string
	Account  string
}

type AntiBrushRuleItem struct {
	Dim    string
	Window time.Duration
	Limit  int64
	Score  int64
}

type AntiBrushTokenBucket struct {
	Dim   string
	RPM   int64
	Burst int64
	Score int64
}

type AntiBrushEscalation struct {
	LightScore  int64
	MediumScore int64
	HeavyScore  int64
	BlockTTL    time.Duration
}

type AntiBrushRule struct {
	Name       string
	PathPrefix string
	FullPath   string

	Windows []AntiBrushRuleItem
	Buckets []AntiBrushTokenBucket

	EnableFailureEscalation bool
	FailureDim              string
	FailureWindow           time.Duration
	FailureLimit            int64
	FailureScore            int64

	Escalation AntiBrushEscalation
}

type AntiBrushConfig struct {
	Redis redis.UniversalClient

	KeyPrefix string

	RespCode    string
	RespMessage string

	Rules []AntiBrushRule

	IdentifiersResolver func(c *gin.Context) AntiBrushIdentifiers

	OnHit   func(c *gin.Context, ev AntiBrushEvent)
	OnBlock func(c *gin.Context, ids AntiBrushIdentifiers, ttl time.Duration, reason string)

	// SecurityEmitter 安全事件发射器（可选）。
	// 用于把 AntiBrush 命中沉淀为安全事件，便于后续封禁/告警/BI。
	SecurityEmitter *security.SecurityEventEmitter

	WriteActionHeader bool
}

func (c *AntiBrushConfig) withDefaults() AntiBrushConfig {
	cfg := *c
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "ab:"
	}
	if cfg.RespCode == "" {
		cfg.RespCode = "anti_brush"
	}
	if cfg.RespMessage == "" {
		cfg.RespMessage = "Too Many Requests"
	}
	if cfg.IdentifiersResolver == nil {
		cfg.IdentifiersResolver = defaultAntiBrushIdentifiers
	}
	if cfg.OnHit == nil {
		cfg.OnHit = func(_ *gin.Context, _ AntiBrushEvent) {}
	}
	if cfg.OnBlock == nil {
		cfg.OnBlock = func(_ *gin.Context, _ AntiBrushIdentifiers, _ time.Duration, _ string) {}
	}
	if len(cfg.Rules) == 0 {
		cfg.Rules = defaultAntiBrushRules()
	}
	if !cfg.WriteActionHeader {
		cfg.WriteActionHeader = true
	}
	return cfg
}

func AntiBrush(cfg AntiBrushConfig) gin.HandlerFunc {
	cfg = cfg.withDefaults()
	if cfg.Redis == nil {
		panic("AntiBrush: Redis 不能为空")
	}

	return func(c *gin.Context) {
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		rule, ok := matchAntiBrushRule(c, cfg.Rules)
		if !ok {
			c.Next()
			return
		}

		ids := cfg.IdentifiersResolver(c)
		if ids.IP == "" {
			ids.IP = GetClientIP(c)
		}
		if ids.UID == "" {
			if uid, ok := defaultUIDResolver(c); ok {
				ids.UID = uid
			}
		}

		rid := GetRequestID(c)
		method := c.Request.Method

		rs, reason, key := cfg.evaluateRule(c.Request.Context(), rule, ids)
		score := rs.Total()
		ruleName := ruleKey(rule)
		action := decideAction(rule.Escalation, score)

		if action == AntiBrushActionBlocked {
			reason = reasonOr(reason, "触发重度风控")
			cfg.emitAndAbort(c, AntiBrushEvent{
				Type:    "anti_brush_hit",
				RID:     rid,
				IP:      ids.IP,
				UID:     ids.UID,
				Route:   route,
				Method:  method,
				Action:  action,
				Score:   score,
				Reasons: rs.Reasons(5),
				Reason:  reason,
				Key:     key,
			}, action, ids, rule.Escalation.BlockTTL, ruleName)
			return
		}

		if action == AntiBrushActionRateLimited || action == AntiBrushActionCaptchaRequired {
			reason = reasonOr(reason, "触发防刷")
			cfg.emitAndAbort(c, AntiBrushEvent{
				Type:    "anti_brush_hit",
				RID:     rid,
				IP:      ids.IP,
				UID:     ids.UID,
				Route:   route,
				Method:  method,
				Action:  action,
				Score:   score,
				Reasons: rs.Reasons(5),
				Reason:  reason,
				Key:     key,
			}, action, ids, 0, ruleName)
			return
		}

		if rule.EnableFailureEscalation {
			w := &statusCaptureWriter{ResponseWriter: c.Writer, status: 200}
			c.Writer = w
			c.Next()
			if isFailureStatus(w.status) {
				_ = cfg.recordFailure(c.Request.Context(), rule, ids)
			}
			return
		}

		c.Next()
	}
}

// emitAndAbort 处理风控命中后的响应和事件发射。
func (cfg AntiBrushConfig) emitAndAbort(c *gin.Context, ev AntiBrushEvent, action AntiBrushAction, ids AntiBrushIdentifiers, blockTTL time.Duration, ruleName string) {
	cfg.OnHit(c, ev)

	// 统一沉淀安全事件（可选）：失败不影响主链路
	emitAntiBrushSecurityEvent(c, cfg.SecurityEmitter, ev, action, ruleName)

	if action == AntiBrushActionBlocked {
		if blockTTL <= 0 {
			blockTTL = 10 * time.Minute
		}
		cfg.OnBlock(c, ids, blockTTL, ev.Reason)
	}

	if cfg.WriteActionHeader {
		c.Writer.Header().Set("X-AntiBrush-Action", string(action))
	}

	if !c.Writer.Written() {
		rid := GetRequestID(c)
		c.AbortWithStatusJSON(429, gin.H{
			"code":   cfg.RespCode,
			"msg":    cfg.RespMessage,
			"rid":    rid,
			"action": action,
		})
		return
	}
	c.Abort()
}

func matchAntiBrushRule(c *gin.Context, rules []AntiBrushRule) (AntiBrushRule, bool) {
	path := ""
	if c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	full := c.FullPath()

	for _, r := range rules {
		if r.FullPath != "" && full != "" && r.FullPath == full {
			return r, true
		}
		if r.PathPrefix != "" && path != "" && strings.HasPrefix(path, r.PathPrefix) {
			return r, true
		}
	}
	return AntiBrushRule{}, false
}

func (cfg AntiBrushConfig) evaluateRule(ctx context.Context, rule AntiBrushRule, ids AntiBrushIdentifiers) (rs *security.RiskScore, reason string, key string) {
	rs = security.NewRiskScore()
	if rule.Escalation.BlockTTL > 0 {
		bk := cfg.blockKey(rule, ids)
		if bk != "" {
			exists, _ := cfg.Redis.Exists(ctx, bk).Result()
			if exists > 0 {
				rs.AddN("blocked_already", rule.Escalation.HeavyScore, security.SeverityHigh, map[string]string{"rule": ruleKey(rule)}, map[string]any{"key": bk})
				return rs, "已被封禁", bk
			}
		}
	}

	for _, w := range rule.Windows {
		val := pickDimValue(w.Dim, ids)
		if val == "" {
			continue
		}
		k := cfg.windowKey(rule, w.Dim, val)
		cnt, err := slidingWindowHit(ctx, cfg.Redis, k, w.Window)
		if err != nil {
			continue
		}
		if cnt > w.Limit {
			rs.AddN("sliding_window_over_limit", w.Score, security.SeverityMedium, map[string]string{"dim": w.Dim, "rule": ruleKey(rule)}, map[string]any{"window": w.Window.String(), "limit": w.Limit})
			reason = appendReason(reason, fmt.Sprintf("滑动窗口超限:%s", w.Dim))
			key = k
		}
	}

	for _, b := range rule.Buckets {
		val := pickDimValue(b.Dim, ids)
		if val == "" {
			continue
		}
		k := cfg.bucketKey(rule, b.Dim, val)
		burst := b.Burst
		if burst <= 0 {
			burst = b.RPM
		}
		allowed, _, err := tokenBucketAllowAB(ctx, cfg.Redis, k, b.RPM, burst)
		if err != nil {
			continue
		}
		if !allowed {
			rs.AddN("token_bucket_over_limit", b.Score, security.SeverityLow, map[string]string{"dim": b.Dim, "rule": ruleKey(rule)}, map[string]any{"rpm": b.RPM, "burst": burst})
			reason = appendReason(reason, fmt.Sprintf("令牌桶超限:%s", b.Dim))
			key = k
		}
	}

	if rule.EnableFailureEscalation && rule.FailureLimit > 0 {
		val := pickDimValue(rule.FailureDim, ids)
		if val != "" {
			fk := cfg.failureKey(rule, rule.FailureDim, val)
			cnt, _ := slidingWindowCount(ctx, cfg.Redis, fk, rule.FailureWindow)
			if cnt >= rule.FailureLimit {
				rs.AddN("failure_burst", rule.FailureScore, security.SeverityMedium, map[string]string{"dim": rule.FailureDim, "rule": ruleKey(rule)}, map[string]any{"window": rule.FailureWindow.String(), "limit": rule.FailureLimit, "count": cnt})
				reason = appendReason(reason, "失败次数过多")
				key = fk
			}
		}
	}

	action := decideAction(rule.Escalation, rs.Total())
	if action == AntiBrushActionBlocked && rule.Escalation.BlockTTL > 0 {
		bk := cfg.blockKey(rule, ids)
		if bk != "" {
			_ = cfg.Redis.Set(ctx, bk, "1", rule.Escalation.BlockTTL).Err()
			key = bk
			reason = appendReason(reason, "写入短期封禁")
		}
	}

	return rs, reason, key
}

func decideAction(es AntiBrushEscalation, score int64) AntiBrushAction {
	if es.HeavyScore > 0 && score >= es.HeavyScore {
		return AntiBrushActionBlocked
	}
	if es.MediumScore > 0 && score >= es.MediumScore {
		return AntiBrushActionCaptchaRequired
	}
	if es.LightScore > 0 && score >= es.LightScore {
		return AntiBrushActionRateLimited
	}
	return AntiBrushActionAllow
}

func (cfg AntiBrushConfig) recordFailure(ctx context.Context, rule AntiBrushRule, ids AntiBrushIdentifiers) error {
	if !rule.EnableFailureEscalation || rule.FailureLimit <= 0 {
		return nil
	}
	val := pickDimValue(rule.FailureDim, ids)
	if val == "" {
		return nil
	}
	k := cfg.failureKey(rule, rule.FailureDim, val)
	_, err := slidingWindowHit(ctx, cfg.Redis, k, rule.FailureWindow)
	return err
}

func pickDimValue(dim string, ids AntiBrushIdentifiers) string {
	switch strings.ToLower(strings.TrimSpace(dim)) {
	case "ip":
		return strings.TrimSpace(ids.IP)
	case "phone":
		return strings.TrimSpace(ids.Phone)
	case "device":
		return strings.TrimSpace(ids.DeviceID)
	case "account":
		return strings.TrimSpace(ids.Account)
	default:
		return ""
	}
}

func (cfg AntiBrushConfig) windowKey(rule AntiBrushRule, dim, val string) string {
	return cfg.KeyPrefix + "w:" + safeKey(ruleKey(rule)) + ":" + safeKey(dim) + ":" + safeKey(val)
}

func (cfg AntiBrushConfig) bucketKey(rule AntiBrushRule, dim, val string) string {
	return cfg.KeyPrefix + "b:" + safeKey(ruleKey(rule)) + ":" + safeKey(dim) + ":" + safeKey(val)
}

func (cfg AntiBrushConfig) failureKey(rule AntiBrushRule, dim, val string) string {
	return cfg.KeyPrefix + "f:" + safeKey(ruleKey(rule)) + ":" + safeKey(dim) + ":" + safeKey(val)
}

func (cfg AntiBrushConfig) blockKey(rule AntiBrushRule, ids AntiBrushIdentifiers) string {
	ip := strings.TrimSpace(ids.IP)
	dev := strings.TrimSpace(ids.DeviceID)
	if ip == "" && dev == "" {
		return ""
	}
	return cfg.KeyPrefix + "blk:" + safeKey(ruleKey(rule)) + ":" + safeKey(ip) + ":" + safeKey(dev)
}

func ruleKey(rule AntiBrushRule) string {
	if rule.Name != "" {
		return rule.Name
	}
	if rule.FullPath != "" {
		return rule.FullPath
	}
	if rule.PathPrefix != "" {
		return rule.PathPrefix
	}
	return "rule"
}

func appendReason(old, add string) string {
	if add == "" {
		return old
	}
	if old == "" {
		return add
	}
	return old + ";" + add
}

func reasonOr(old, fallback string) string {
	if strings.TrimSpace(old) != "" {
		return old
	}
	return fallback
}

func isFailureStatus(code int) bool {
	switch code {
	case 400, 401, 403, 429, 500:
		return true
	default:
		return false
	}
}

func defaultAntiBrushIdentifiers(c *gin.Context) AntiBrushIdentifiers {
	ids := AntiBrushIdentifiers{}
	ids.IP = GetClientIP(c)

	if uid, ok := defaultUIDResolver(c); ok {
		ids.UID = uid
	}

	if v, ok := c.Get("ab_device"); ok {
		if s, ok2 := v.(string); ok2 {
			ids.DeviceID = strings.TrimSpace(s)
		}
	}
	if ids.DeviceID == "" {
		ids.DeviceID = common.FirstNonEmpty(
			c.GetHeader("X-Device-Id"),
			c.GetHeader("X-Device-ID"),
			c.GetHeader("X-Client-Id"),
			c.GetHeader("X-Client-ID"),
		)
		ids.DeviceID = strings.TrimSpace(ids.DeviceID)
	}

	if v, ok := c.Get("ab_phone"); ok {
		if s, ok2 := v.(string); ok2 {
			ids.Phone = strings.TrimSpace(s)
		}
	}
	if ids.Phone == "" {
		ids.Phone = strings.TrimSpace(common.FirstNonEmpty(
			c.Query("phone"),
			c.Query("mobile"),
			c.PostForm("phone"),
			c.PostForm("mobile"),
		))
	}
	if ids.Phone == "" {
		ids.Phone = strings.TrimSpace(getSmallJSONField(c, 4096, "phone", "mobile"))
	}

	if v, ok := c.Get("ab_account"); ok {
		if s, ok2 := v.(string); ok2 {
			ids.Account = strings.TrimSpace(s)
		}
	}
	if ids.Account == "" {
		ids.Account = strings.TrimSpace(common.FirstNonEmpty(
			c.Query("account"),
			c.PostForm("account"),
			c.Query("email"),
			c.PostForm("email"),
		))
	}
	if ids.Account == "" {
		ids.Account = strings.TrimSpace(getSmallJSONField(c, 4096, "account", "email"))
	}

	return ids
}

func getSmallJSONField(c *gin.Context, maxBytes int64, keys ...string) string {
	if c.Request == nil {
		return ""
	}
	ct := strings.ToLower(strings.TrimSpace(c.GetHeader("Content-Type")))
	if ct == "" || !strings.HasPrefix(ct, "application/json") {
		return ""
	}
	if c.Request.ContentLength <= 0 || c.Request.ContentLength > maxBytes {
		return ""
	}
	if c.Request.Body == nil {
		return ""
	}

	b, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBytes))
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(b))
	b = bytes.TrimSpace(b)
	if len(b) == 0 {
		return ""
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return fmt.Sprint(v)
		}
	}
	return ""
}

// ----------------------
// Redis + Lua：滑动窗口
// ----------------------

var luaSlidingWindowHit = redis.NewScript(`
-- KEYS[1] = key
-- ARGV[1] = now_ms
-- ARGV[2] = window_ms
-- ARGV[3] = ttl_ms

local key = KEYS[1]
local now_ms = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local ttl_ms = tonumber(ARGV[3])

local min_ms = now_ms - window_ms
redis.call('ZREMRANGEBYSCORE', key, 0, min_ms)

local member = tostring(now_ms) .. '-' .. tostring(math.random(100000, 999999))
redis.call('ZADD', key, now_ms, member)

local cnt = redis.call('ZCARD', key)
redis.call('PEXPIRE', key, ttl_ms)
return cnt
`)

var luaSlidingWindowCount = redis.NewScript(`
-- KEYS[1] = key
-- ARGV[1] = now_ms
-- ARGV[2] = window_ms
-- ARGV[3] = ttl_ms

local key = KEYS[1]
local now_ms = tonumber(ARGV[1])
local window_ms = tonumber(ARGV[2])
local ttl_ms = tonumber(ARGV[3])

local min_ms = now_ms - window_ms
redis.call('ZREMRANGEBYSCORE', key, 0, min_ms)
local cnt = redis.call('ZCARD', key)
redis.call('PEXPIRE', key, ttl_ms)
return cnt
`)

func slidingWindowHit(ctx context.Context, rdb redis.UniversalClient, key string, window time.Duration) (int64, error) {
	if window <= 0 {
		return 0, errors.New("window 不能为空")
	}
	nowMs := nowUnixMs()
	windowMs := int64(window / time.Millisecond)
	ttlMs := windowMs * 2
	res, err := luaSlidingWindowHit.Run(ctx, rdb, []string{key}, nowMs, windowMs, ttlMs).Result()
	if err != nil {
		return 0, err
	}
	return toInt64(res), nil
}

func slidingWindowCount(ctx context.Context, rdb redis.UniversalClient, key string, window time.Duration) (int64, error) {
	if window <= 0 {
		return 0, errors.New("window 不能为空")
	}
	nowMs := nowUnixMs()
	windowMs := int64(window / time.Millisecond)
	ttlMs := windowMs * 2
	res, err := luaSlidingWindowCount.Run(ctx, rdb, []string{key}, nowMs, windowMs, ttlMs).Result()
	if err != nil {
		return 0, err
	}
	return toInt64(res), nil
}

func nowUnixMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// ----------------------
// Redis + Lua：令牌桶（AntiBrush 专用）
// ----------------------

var luaTokenBucketAB = redis.NewScript(`
-- KEYS[1] = key
-- ARGV[1] = capacity
-- ARGV[2] = rate_per_sec
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
return {allowed, math.floor(tokens)}
`)

func tokenBucketAllowAB(ctx context.Context, rdb redis.UniversalClient, key string, rpm int64, burst int64) (bool, int64, error) {
	if rpm <= 0 {
		return true, 0, nil
	}
	if burst <= 0 {
		burst = rpm
	}
	capacity := float64(burst)
	ratePerSec := float64(rpm) / 60.0
	ttlMs := int64(2 * time.Minute / time.Millisecond)
	res, err := luaTokenBucketAB.Run(ctx, rdb, []string{key}, capacity, ratePerSec, ttlMs).Result()
	if err != nil {
		return true, 0, err
	}
	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return true, 0, errors.New("anti_brush token bucket lua 返回格式错误")
	}
	allowed := toInt64(arr[0]) == 1
	remaining := toInt64(arr[1])
	return allowed, remaining, nil
}

type statusCaptureWriter struct {
	gin.ResponseWriter
	status int
}

func (w *statusCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func defaultAntiBrushRules() []AntiBrushRule {
	return []AntiBrushRule{
		{
			Name:       "auth_sms_send",
			PathPrefix: "/auth/sms/send",
			Windows: []AntiBrushRuleItem{
				{Dim: "ip", Window: 60 * time.Second, Limit: 3, Score: 50},
				{Dim: "phone", Window: 10 * time.Minute, Limit: 3, Score: 60},
				{Dim: "device", Window: 10 * time.Minute, Limit: 5, Score: 40},
			},
			Buckets: []AntiBrushTokenBucket{
				{Dim: "ip", RPM: 60, Burst: 30, Score: 30},
			},
			Escalation: AntiBrushEscalation{
				LightScore:  50,
				MediumScore: 90,
				HeavyScore:  140,
				BlockTTL:    10 * time.Minute,
			},
		},
		{
			Name:       "auth_login",
			PathPrefix: "/auth/login",
			Windows: []AntiBrushRuleItem{
				{Dim: "ip", Window: 10 * time.Minute, Limit: 20, Score: 60},
				{Dim: "account", Window: 10 * time.Minute, Limit: 10, Score: 70},
			},
			Buckets: []AntiBrushTokenBucket{
				{Dim: "ip", RPM: 120, Burst: 60, Score: 20},
			},
			EnableFailureEscalation: true,
			FailureDim:              "account",
			FailureWindow:           10 * time.Minute,
			FailureLimit:            5,
			FailureScore:            80,
			Escalation: AntiBrushEscalation{
				LightScore:  60,
				MediumScore: 120,
				HeavyScore:  180,
				BlockTTL:    15 * time.Minute,
			},
		},
	}
}

// emitAntiBrushSecurityEvent 统一上报 AntiBrush 命中到安全事件。
//
// 说明：
//   - 仅在配置了 SecurityEmitter 且 Writer/Queue 至少一个存在时生效
//   - 不影响主链路
func emitAntiBrushSecurityEvent(c *gin.Context, emitter *security.SecurityEventEmitter, ev AntiBrushEvent, action AntiBrushAction, ruleName string) {
	if emitter == nil || (emitter.Writer == nil && emitter.Queue == nil) {
		return
	}

	sev := security.SecuritySeverityLow
	switch action {
	case AntiBrushActionCaptchaRequired:
		sev = security.SecuritySeverityMedium
	case AntiBrushActionBlocked:
		sev = security.SecuritySeverityHigh
	case AntiBrushActionRateLimited:
		sev = security.SecuritySeverityLow
	default:
		sev = security.SecuritySeverityLow
	}

	details := map[string]any{
		"reason":  ev.Reason,
		"action":  string(action),
		"score":   ev.Score,
		"key":     ev.Key,
		"rule":    ruleName,
		"route":   ev.Route,
		"method":  ev.Method,
		"ip":      ev.IP,
		"reasons": ev.Reasons,
	}
	_ = security.EmitSecurityEvent(c, emitter, "anti_brush_hit", sev, details)
}
