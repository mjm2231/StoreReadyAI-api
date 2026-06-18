package log

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ============ Context keys (request scoped) ============

type ctxKey string

const (
	ctxKeyRequestID ctxKey = "request_id"
	ctxKeyTraceID   ctxKey = "trace_id"
	ctxKeySpanID    ctxKey = "span_id"
	ctxKeyUserID    ctxKey = "user_id"
	ctxKeyTenantID  ctxKey = "tenant_id"
	ctxKeyIP        ctxKey = "ip"
)

// WithRequestMeta attaches common request-scoped fields into context.
// Use it in middleware to enrich logs everywhere.
func WithRequestMeta(ctx context.Context, requestID, traceID, spanID, ip, tenantID, userID string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyRequestID, requestID)
	ctx = context.WithValue(ctx, ctxKeyTraceID, traceID)
	ctx = context.WithValue(ctx, ctxKeySpanID, spanID)
	ctx = context.WithValue(ctx, ctxKeyIP, ip)
	ctx = context.WithValue(ctx, ctxKeyTenantID, tenantID)
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	return ctx
}

func getStr(ctx context.Context, k ctxKey) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(k).(string)
	return v
}

// ============ Logger ============

// Config is an enterprise-grade logger configuration.
// JSON should be enabled in prod; console is more readable in dev.
//
// TimeKey is usually "ts"; LevelKey often "level".
// CallerKey includes file:line.
//
// Sampling is recommended for very high QPS services; keep disabled by default.
//
// OutputPaths examples: []string{"stdout"} or []string{"./.data/logs/app.log"}
// ErrorOutputPaths examples: []string{"stderr"}

// Note: This logger is designed for local-dev without containers. For production,
// consider rotating file output with lumberjack (optional) or external log collector.

type Config struct {
	Service    string
	Env        string // dev|staging|prod
	Level      string // debug|info|warn|error
	JSON       bool
	Caller     bool
	Stacktrace bool
	Sampling   bool

	OutputPaths      []string
	ErrorOutputPaths []string

	TimeKey   string
	LevelKey  string
	NameKey   string
	CallerKey string
	MsgKey    string

	// RedactKeys: keys that will be redacted in fields (e.g. password, token)
	RedactKeys []string
}

var (
	global     *zap.Logger
	globalSug  *zap.SugaredLogger
	redactKeys = map[string]struct{}{}
)

// Init initializes global logger. Should be called once at startup.
func Init(cfg Config) (*zap.Logger, error) {
	if cfg.TimeKey == "" {
		cfg.TimeKey = "ts"
	}
	if cfg.LevelKey == "" {
		cfg.LevelKey = "level"
	}
	if cfg.MsgKey == "" {
		cfg.MsgKey = "msg"
	}
	if cfg.NameKey == "" {
		cfg.NameKey = "logger"
	}
	if cfg.CallerKey == "" {
		cfg.CallerKey = "caller"
	}
	if len(cfg.OutputPaths) == 0 {
		cfg.OutputPaths = []string{"stdout"}
	}
	if len(cfg.ErrorOutputPaths) == 0 {
		cfg.ErrorOutputPaths = []string{"stderr"}
	}
	for _, k := range cfg.RedactKeys {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" {
			redactKeys[k] = struct{}{}
		}
	}

	lvl := parseLevel(cfg.Level)
	encCfg := zapcore.EncoderConfig{
		TimeKey:        cfg.TimeKey,
		LevelKey:       cfg.LevelKey,
		NameKey:        cfg.NameKey,
		CallerKey:      cfg.CallerKey,
		MessageKey:     cfg.MsgKey,
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     epochMillisTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var enc zapcore.Encoder
	if cfg.JSON {
		enc = zapcore.NewJSONEncoder(encCfg)
	} else {
		encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		enc = zapcore.NewConsoleEncoder(encCfg)
	}

	ws, err := buildWriteSyncers(cfg.OutputPaths)
	if err != nil {
		return nil, err
	}
	ews, err := buildWriteSyncers(cfg.ErrorOutputPaths)
	if err != nil {
		return nil, err
	}

	core := zapcore.NewCore(enc, ws, lvl)

	// Send error-level logs to error outputs as well.
	errorCore := zapcore.NewCore(enc, ews, zapcore.ErrorLevel)
	core = zapcore.NewTee(core, errorCore)

	// Optional sampling (for extremely high QPS). Disabled by default.
	if cfg.Sampling {
		core = zapcore.NewSamplerWithOptions(core, time.Second, 100, 100)
	}

	z := zap.New(core)
	if cfg.Caller {
		z = z.WithOptions(zap.AddCaller())
	}
	if cfg.Stacktrace {
		z = z.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))
	}

	// static fields
	if cfg.Service != "" {
		z = z.With(zap.String("service", cfg.Service))
	}
	if cfg.Env != "" {
		z = z.With(zap.String("env", cfg.Env))
	}

	global = z
	globalSug = z.Sugar()
	return z, nil
}

func L() *zap.Logger {
	if global == nil {
		// fallback minimal logger
		z, _ := Init(Config{JSON: false, Level: "info", OutputPaths: []string{"stdout"}, ErrorOutputPaths: []string{"stderr"}})
		return z
	}
	return global
}

func S() *zap.SugaredLogger { return L().Sugar() }

// C returns a logger enriched with context fields.
func C(ctx context.Context) *zap.Logger {
	z := L()
	if ctx == nil {
		return z
	}

	fields := make([]zap.Field, 0, 6)
	if v := getStr(ctx, ctxKeyRequestID); v != "" {
		fields = append(fields, zap.String("request_id", v))
	}
	if v := getStr(ctx, ctxKeyTraceID); v != "" {
		fields = append(fields, zap.String("trace_id", v))
	}
	if v := getStr(ctx, ctxKeySpanID); v != "" {
		fields = append(fields, zap.String("span_id", v))
	}
	if v := getStr(ctx, ctxKeyIP); v != "" {
		fields = append(fields, zap.String("ip", v))
	}
	if v := getStr(ctx, ctxKeyTenantID); v != "" {
		fields = append(fields, zap.String("tenant_id", v))
	}
	if v := getStr(ctx, ctxKeyUserID); v != "" {
		fields = append(fields, zap.String("user_id", v))
	}

	if len(fields) == 0 {
		return z
	}
	return z.With(fields...)
}

// Sync flushes buffered logs.
func Sync() { _ = L().Sync() }

// ============ Helpers ============

func parseLevel(s string) zapcore.Level {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func epochMillisTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendInt64(t.UnixMilli())
}

func buildWriteSyncers(paths []string) (zapcore.WriteSyncer, error) {
	writers := make([]zapcore.WriteSyncer, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		switch p {
		case "", "stdout":
			writers = append(writers, zapcore.AddSync(os.Stdout))
		case "stderr":
			writers = append(writers, zapcore.AddSync(os.Stderr))
		default:
			f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return nil, err
			}
			writers = append(writers, zapcore.AddSync(f))
		}
	}
	if len(writers) == 1 {
		return writers[0], nil
	}
	return zapcore.NewMultiWriteSyncer(writers...), nil
}

// ============ Field helpers with redaction ============

// String returns a zap.Field with optional redaction.
func String(key, val string) zap.Field {
	if shouldRedact(key) {
		return zap.String(key, "***")
	}
	return zap.String(key, val)
}

func Any(key string, val any) zap.Field {
	if shouldRedact(key) {
		return zap.String(key, "***")
	}
	return zap.Any(key, val)
}

func shouldRedact(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	_, ok := redactKeys[k]
	return ok
}

// Writer exposes an io.Writer that writes logs at a fixed level.
// Useful to integrate third-party libs (e.g., gorm logger) into zap.
func Writer(level zapcore.Level) io.Writer {
	return &levelWriter{level: level}
}

type levelWriter struct {
	level zapcore.Level
}

func (lw *levelWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}
	switch lw.level {
	case zapcore.DebugLevel:
		L().Debug(msg)
	case zapcore.InfoLevel:
		L().Info(msg)
	case zapcore.WarnLevel:
		L().Warn(msg)
	default:
		L().Error(msg)
	}
	return len(p), nil
}
