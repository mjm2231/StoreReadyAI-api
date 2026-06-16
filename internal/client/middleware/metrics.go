package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsConfig Prometheus 指标配置。
//
// 说明：
//   - Namespace/SubSystem 用于统一管理指标前缀（可选）
//   - Buckets 用于请求耗时直方图的桶（可选）
//
// 默认指标：
//   - http_requests_total{route,method,code}
//   - http_request_duration_seconds_bucket{route,method}
//   - rate_limit_hit_total{route,dimension}
//   - anti_brush_hit_total{route,reason}
type MetricsConfig struct {
	Namespace string
	Subsystem string

	// Buckets 请求耗时直方图 buckets（单位：秒）。
	// 为空时使用 Prometheus 默认 buckets。
	Buckets []float64
}

var (
	metricsOnce sync.Once

	httpRequestsTotal          *prometheus.CounterVec
	httpRequestDurationSeconds *prometheus.HistogramVec
	rateLimitHitTotal          *prometheus.CounterVec
	antiBrushHitTotal          *prometheus.CounterVec
)

func initMetrics(cfg MetricsConfig) {
	metricsOnce.Do(func() {
		ns := strings.TrimSpace(cfg.Namespace)
		sub := strings.TrimSpace(cfg.Subsystem)
		buckets := cfg.Buckets
		if len(buckets) == 0 {
			buckets = prometheus.DefBuckets
		}

		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "http_requests_total",
				Help:      "HTTP 请求总数（按 route/method/code 统计）",
			},
			[]string{"route", "method", "code"},
		)

		httpRequestDurationSeconds = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP 请求耗时分布（秒，按 route/method 统计）",
				Buckets:   buckets,
			},
			[]string{"route", "method"},
		)

		rateLimitHitTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "rate_limit_hit_total",
				Help:      "RateLimit 命中次数（按 route/dimension 统计，dimension=ip|uid）",
			},
			[]string{"route", "dimension"},
		)

		antiBrushHitTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: ns,
				Subsystem: sub,
				Name:      "anti_brush_hit_total",
				Help:      "AntiBrush 命中次数（按 route/reason 统计）",
			},
			[]string{"route", "reason"},
		)

		// 注册到默认 registry
		prometheus.MustRegister(httpRequestsTotal)
		prometheus.MustRegister(httpRequestDurationSeconds)
		prometheus.MustRegister(rateLimitHitTotal)
		prometheus.MustRegister(antiBrushHitTotal)
	})
}

// Metrics Prometheus 统计中间件。
//
// 建议挂载顺序：
//   - 放在 Auth/Audit 之后、AccessLog 之前或之后都可以
//   - route 建议使用 gin.FullPath（需要路由匹配后才有值）
func Metrics(cfg MetricsConfig) gin.HandlerFunc {
	initMetrics(cfg)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := strings.TrimSpace(c.FullPath())
		if route == "" {
			if c.Request != nil && c.Request.URL != nil {
				route = strings.TrimSpace(c.Request.URL.Path)
			}
			if route == "" {
				route = "unknown"
			}
		}

		method := ""
		if c.Request != nil {
			method = strings.ToUpper(strings.TrimSpace(c.Request.Method))
		}
		if method == "" {
			method = "UNKNOWN"
		}

		status := c.Writer.Status()
		code := strconv.Itoa(status)

		dur := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(route, method, code).Inc()
		httpRequestDurationSeconds.WithLabelValues(route, method).Observe(dur)
	}
}

// MetricsHandler 返回 /metrics HTTP handler（promhttp）。
//
// 使用方式：
//
//	r.GET("/metrics", middleware.MetricsHandler())
func MetricsHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// IncRateLimitHit 记录 RateLimit 命中次数。
//
// 说明：
//   - dimension 建议传 "ip" 或 "uid"
//   - route 建议传 gin.FullPath；为空会被归一为 "unknown"
func IncRateLimitHit(route, dimension string) {
	initMetrics(MetricsConfig{})
	r := strings.TrimSpace(route)
	if r == "" {
		r = "unknown"
	}
	d := strings.TrimSpace(dimension)
	if d == "" {
		d = "unknown"
	}
	rateLimitHitTotal.WithLabelValues(r, d).Inc()
}

// IncAntiBrushHit 记录 AntiBrush 命中次数。
//
// 说明：
//   - reason 建议传可枚举的短字符串（例如 action 或命中原因）
//   - route 建议传 gin.FullPath；为空会被归一为 "unknown"
func IncAntiBrushHit(route, reason string) {
	initMetrics(MetricsConfig{})
	r := strings.TrimSpace(route)
	if r == "" {
		r = "unknown"
	}
	rs := strings.TrimSpace(reason)
	if rs == "" {
		rs = "unknown"
	}
	antiBrushHitTotal.WithLabelValues(r, rs).Inc()
}

// 兼容：如果你们有自定义的 health handler，避免被误用。
func noCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Status(http.StatusNoContent)
}
