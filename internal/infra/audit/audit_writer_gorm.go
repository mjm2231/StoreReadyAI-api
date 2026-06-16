package audit

import (
	"context"

	"storeready_ai/internal/client/middleware"

	"gorm.io/gorm"
)

// GormAuditWriter 使用 GORM 将审计日志落库。
// 表结构见 sqls/20_audit_logs.sql
type GormAuditWriter struct {
	DB *gorm.DB
}

// AuditLogModel 对应 audit_logs 表（字段以 SQL 迁移为准）。
type AuditLogModel struct {
	ID uint64 `gorm:"primaryKey;autoIncrement;column:id"`

	RID     string `gorm:"column:rid"`
	TraceID string `gorm:"column:trace_id"`

	CreatedAt int64 `gorm:"column:created_at"`

	UID      string `gorm:"column:uid"`
	TenantID string `gorm:"column:tenant_id"`
	Role     string `gorm:"column:role"`
	Scopes   string `gorm:"column:scopes"`

	Action       string `gorm:"column:action"`
	ResourceType string `gorm:"column:resource_type"`
	ResourceID   string `gorm:"column:resource_id"`

	IP     string `gorm:"column:ip"`
	UA     string `gorm:"column:ua"`
	Device string `gorm:"column:device"`
	Refer  string `gorm:"column:refer"`

	Success    bool   `gorm:"column:success"`
	HTTPStatus int    `gorm:"column:http_status"`
	ErrCode    string `gorm:"column:err_code"`
	LatencyMS  int64  `gorm:"column:latency_ms"`

	Method string `gorm:"column:method"`
	Path   string `gorm:"column:path"`

	QuerySummary string `gorm:"column:query_summary"`
	BodySummary  string `gorm:"column:body_summary"`
	RespSummary  string `gorm:"column:resp_summary"`

	RequestSizeB  int64 `gorm:"column:request_size_b"`
	ResponseSizeB int64 `gorm:"column:response_size_b"`

	RiskScore   int64  `gorm:"column:risk_score"`
	RiskAction  string `gorm:"column:risk_action"`
	RiskReasons string `gorm:"column:risk_reasons"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }

func NewGormAuditWriter(db *gorm.DB) *GormAuditWriter {
	return &GormAuditWriter{DB: db}
}

// Write 写入一条审计记录。
func (w *GormAuditWriter) Write(ctx context.Context, rec middleware.AuditRecord) error {
	if w == nil || w.DB == nil {
		return nil
	}
	m := AuditLogModel{
		RID:       rec.RID,
		TraceID:   rec.TraceID,
		CreatedAt: rec.CreatedAt,

		UID:      rec.UID,
		TenantID: rec.TenantID,
		Role:     rec.Role,
		Scopes:   rec.Scopes,

		Action:       rec.Action,
		ResourceType: rec.ResourceType,
		ResourceID:   rec.ResourceID,

		IP:     rec.IP,
		UA:     rec.UA,
		Device: rec.Device,
		Refer:  rec.Refer,

		Success:    rec.Success,
		HTTPStatus: rec.HTTPStatus,
		ErrCode:    rec.ErrCode,
		LatencyMS:  rec.LatencyMS,

		Method: rec.Method,
		Path:   rec.Path,

		QuerySummary: rec.QuerySummary,
		BodySummary:  rec.BodySummary,
		RespSummary:  rec.RespSummary,

		RequestSizeB:  rec.RequestSizeB,
		ResponseSizeB: rec.ResponseSizeB,

		RiskScore:   rec.RiskScore,
		RiskAction:  rec.RiskAction,
		RiskReasons: rec.RiskReasons,
	}
	return w.DB.WithContext(ctx).Create(&m).Error
}
