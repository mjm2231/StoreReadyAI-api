package tracer

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"storeready_ai/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Init 初始化 OpenTelemetry Tracer。
//
// 说明：
//   - 若 tracing.enabled=false，则什么都不做，返回空的 shutdown。
//   - exporter 支持：otlp_grpc（默认） / otlp_http（推荐与 Jaeger/Tempo/OTel Collector 兼容）
//   - endpoint/insecure/sample_ratio 由配置控制。
//   - 会设置全局：TracerProvider + TextMapPropagator（W3C TraceContext + Baggage）。
//
// 返回值：
//   - shutdown：用于优雅退出时 flush & close exporter
func Init(cfg *config.Config) (shutdown func(ctx context.Context) error, err error) {
	if cfg == nil {
		return func(context.Context) error { return nil }, errors.New("tracer: cfg 为空")
	}
	if !cfg.Observability.Tracing.Enabled {
		// 未开启 tracing：返回空 shutdown
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return func(context.Context) error { return nil }, nil
	}

	serviceName := strings.TrimSpace(cfg.Server.Name)
	if serviceName == "" {
		serviceName = "storeready_ai"
	}

	// 1) 构建 exporter
	exp, expShutdown, err := buildExporter(cfg.Observability.Tracing)
	if err != nil {
		return func(context.Context) error { return nil }, err
	}

	// 2) Resource（服务元信息）
	res, err := resource.New(
		context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("env", strings.ToLower(strings.TrimSpace(cfg.Server.Mode))),
		),
	)
	if err != nil {
		_ = expShutdown(context.Background())
		return func(context.Context) error { return nil }, fmt.Errorf("tracer: create resource failed: %w", err)
	}

	// 3) 采样策略（ratio 取值建议 0~1）
	ratio := cfg.Observability.Tracing.SampleRatio
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	// 4) TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		// 批量导出：企业默认推荐
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
	)

	// 5) 设置全局 provider + 传播器
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown = func(ctx context.Context) error {
		// 先 flush provider
		err1 := tp.Shutdown(ctx)
		// 再关闭 exporter（若 exporter 有独立 close）
		err2 := expShutdown(ctx)
		if err1 != nil {
			return err1
		}
		return err2
	}
	return shutdown, nil
}

// Tracer 获取 tracer（封装 otel.Tracer）。
func Tracer(name string) trace.Tracer {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "storeready_ai"
	}
	return otel.Tracer(name)
}

// buildExporter 根据配置创建 exporter。
//
// 注意：
//   - OpenTelemetry 已移除对 Jaeger exporter 的官方支持（2023 年后）。
//   - Jaeger 官方也推荐使用 OTLP（通常经由 OTel Collector 或 Jaeger 的 OTLP 接收端）。
//
// 支持：
//   - otlp_grpc（默认）
//   - otlp_http
//   - otlp（等同 otlp_grpc）
//
// 兼容：
//   - exporter=jaeger 会自动降级为 otlp_grpc（避免编译期 deprecated 依赖）。
func buildExporter(cfg config.ObservabilityTracingConfig) (sdktrace.SpanExporter, func(ctx context.Context) error, error) {
	exporter := strings.ToLower(strings.TrimSpace(cfg.Exporter))
	if exporter == "" {
		exporter = "otlp_grpc"
	}
	if exporter == "otlp" {
		exporter = "otlp_grpc"
	}
	if exporter == "grpc" {
		exporter = "otlp_grpc"
	}
	if exporter == "http" {
		exporter = "otlp_http"
	}
	if exporter == "jaeger" {
		// 兼容旧配置：自动降级为 otlp_grpc
		exporter = "otlp_grpc"
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, func(context.Context) error { return nil }, errors.New("tracer: otlp endpoint 不能为空")
	}

	switch exporter {
	case "otlp_grpc":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(endpoint),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		client := otlptracegrpc.NewClient(opts...)
		exp, err := otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, func(context.Context) error { return nil }, fmt.Errorf("tracer: create otlp grpc exporter failed: %w", err)
		}
		return exp, func(ctx context.Context) error { return exp.Shutdown(ctx) }, nil

	case "otlp_http":
		// otlptracehttp 的 endpoint 推荐写 host:port。
		// 为了更友好，这里也支持写成 http(s)://host:port/path 的形式。
		endpointOpt := endpoint
		pathOpt := ""
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			u, err := url.Parse(endpoint)
			if err == nil {
				if strings.TrimSpace(u.Host) != "" {
					endpointOpt = u.Host
				}
				if strings.TrimSpace(u.Path) != "" && u.Path != "/" {
					pathOpt = u.Path
				}
			}
		}

		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(endpointOpt),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if pathOpt != "" {
			opts = append(opts, otlptracehttp.WithURLPath(pathOpt))
		}

		client := otlptracehttp.NewClient(opts...)
		exp, err := otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, func(context.Context) error { return nil }, fmt.Errorf("tracer: create otlp http exporter failed: %w", err)
		}
		return exp, func(ctx context.Context) error { return exp.Shutdown(ctx) }, nil

	default:
		return nil, func(context.Context) error { return nil }, fmt.Errorf("tracer: unsupported exporter: %s (supported: otlp_grpc|otlp_http)", exporter)
	}
}
