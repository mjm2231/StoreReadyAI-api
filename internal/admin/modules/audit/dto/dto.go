package dto

import (
	"strings"

	auditmodel "storeready_ai/internal/admin/modules/audit/model"
	"storeready_ai/internal/common/page"
)

// CreateAuditLogRequest 创建审计日志请求。
//
// 说明：
// 1. 审计日志通常由服务端内部写入，这个 DTO 主要用于 service 层统一接收参数；
// 2. query/body/resp/risk_reasons 为摘要或 JSON 字符串，进入 service 前应已完成脱敏与截断；
// 3. success 使用 0/1 表达，便于与表结构保持一致。
type CreateAuditLogRequest struct {
	RID           string `json:"rid"`
	TraceID       string `json:"trace_id"`
	CreatedAt     int64  `json:"created_at"`
	UID           string `json:"uid"`
	TenantID      string `json:"tenant_id"`
	Role          string `json:"role"`
	Scopes        string `json:"scopes"`
	Action        string `json:"action"`
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	IP            string `json:"ip"`
	UA            string `json:"ua"`
	Device        string `json:"device"`
	Refer         string `json:"refer"`
	Success       uint8  `json:"success"`
	HTTPStatus    int32  `json:"http_status"`
	ErrCode       string `json:"err_code"`
	LatencyMS     int64  `json:"latency_ms"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	QuerySummary  string `json:"query_summary"`
	BodySummary   string `json:"body_summary"`
	RespSummary   string `json:"resp_summary"`
	RequestSizeB  int64  `json:"request_size_b"`
	ResponseSizeB int64  `json:"response_size_b"`
	RiskScore     int64  `json:"risk_score"`
	RiskAction    string `json:"risk_action"`
	RiskReasons   string `json:"risk_reasons"`
}

func (r CreateAuditLogRequest) Normalize() CreateAuditLogRequest {
	r.RID = strings.TrimSpace(r.RID)
	r.TraceID = strings.TrimSpace(r.TraceID)
	r.UID = strings.TrimSpace(r.UID)
	r.TenantID = strings.TrimSpace(r.TenantID)
	r.Role = strings.TrimSpace(r.Role)
	r.Scopes = strings.TrimSpace(r.Scopes)
	r.Action = strings.TrimSpace(r.Action)
	r.ResourceType = strings.TrimSpace(r.ResourceType)
	r.ResourceID = strings.TrimSpace(r.ResourceID)
	r.IP = strings.TrimSpace(r.IP)
	r.UA = strings.TrimSpace(r.UA)
	r.Device = strings.TrimSpace(r.Device)
	r.Refer = strings.TrimSpace(r.Refer)
	r.ErrCode = strings.TrimSpace(r.ErrCode)
	r.Method = strings.TrimSpace(strings.ToUpper(r.Method))
	r.Path = strings.TrimSpace(r.Path)
	r.QuerySummary = strings.TrimSpace(r.QuerySummary)
	r.BodySummary = strings.TrimSpace(r.BodySummary)
	r.RespSummary = strings.TrimSpace(r.RespSummary)
	r.RiskAction = strings.TrimSpace(r.RiskAction)
	r.RiskReasons = strings.TrimSpace(r.RiskReasons)
	if r.Success > 0 {
		r.Success = 1
	}
	return r
}

// GetAuditLogDetailRequest 获取审计日志详情请求。
type GetAuditLogDetailRequest struct {
	ID uint64 `json:"id" binding:"required"`
}

// AuditLogListRequest 后台审计日志列表请求。
type AuditLogListRequest struct {
	RID          string       `json:"rid"`
	TraceID      string       `json:"trace_id"`
	UID          string       `json:"uid"`
	TenantID     string       `json:"tenant_id"`
	Role         string       `json:"role"`
	Action       string       `json:"action"`
	ResourceType string       `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	Success      *uint8       `json:"success"`
	HTTPStatus   *int32       `json:"http_status"`
	ErrCode      string       `json:"err_code"`
	Method       string       `json:"method"`
	Path         string       `json:"path"`
	Keyword      string       `json:"keyword"`
	CreatedFrom  *int64       `json:"created_from"`
	CreatedTo    *int64       `json:"created_to"`
	IDs          []uint64     `json:"ids"`
	Page         page.PageReq `json:"page_req"`
}

func (r AuditLogListRequest) Normalize() AuditLogListRequest {
	r.RID = strings.TrimSpace(r.RID)
	r.TraceID = strings.TrimSpace(r.TraceID)
	r.UID = strings.TrimSpace(r.UID)
	r.TenantID = strings.TrimSpace(r.TenantID)
	r.Role = strings.TrimSpace(r.Role)
	r.Action = strings.TrimSpace(r.Action)
	r.ResourceType = strings.TrimSpace(r.ResourceType)
	r.ResourceID = strings.TrimSpace(r.ResourceID)
	r.ErrCode = strings.TrimSpace(r.ErrCode)
	r.Method = strings.TrimSpace(strings.ToUpper(r.Method))
	r.Path = strings.TrimSpace(r.Path)
	r.Keyword = strings.TrimSpace(r.Keyword)
	if r.Page.Page < 0 {
		r.Page.Page = 0
	}
	if r.Page.PageSize <= 0 {
		r.Page.PageSize = 20
	}
	if r.Page.PageSize > 200 {
		r.Page.PageSize = 200
	}
	if len(r.IDs) > 0 {
		ids := make([]uint64, 0, len(r.IDs))
		seen := make(map[uint64]struct{}, len(r.IDs))
		for _, id := range r.IDs {
			if id == 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			r.IDs = nil
		} else {
			r.IDs = ids
		}
	}
	if r.Success != nil && *r.Success > 0 {
		v := uint8(1)
		r.Success = &v
	}
	return r
}

// AuditLogItem 后台审计日志列表项。
type AuditLogItem struct {
	ID            uint64 `json:"id"`
	RID           string `json:"rid"`
	TraceID       string `json:"trace_id"`
	CreatedAt     int64  `json:"created_at"`
	UID           string `json:"uid"`
	TenantID      string `json:"tenant_id"`
	Role          string `json:"role"`
	Scopes        string `json:"scopes"`
	Action        string `json:"action"`
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	IP            string `json:"ip"`
	UA            string `json:"ua"`
	Device        string `json:"device"`
	Refer         string `json:"refer"`
	Success       uint8  `json:"success"`
	HTTPStatus    int32  `json:"http_status"`
	ErrCode       string `json:"err_code"`
	LatencyMS     int64  `json:"latency_ms"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	QuerySummary  string `json:"query_summary"`
	BodySummary   string `json:"body_summary"`
	RespSummary   string `json:"resp_summary"`
	RequestSizeB  int64  `json:"request_size_b"`
	ResponseSizeB int64  `json:"response_size_b"`
	RiskScore     int64  `json:"risk_score"`
	RiskAction    string `json:"risk_action"`
	RiskReasons   string `json:"risk_reasons"`
}

// AuditLogDetail 后台审计日志详情。
type AuditLogDetail struct {
	ID            uint64 `json:"id"`
	RID           string `json:"rid"`
	TraceID       string `json:"trace_id"`
	CreatedAt     int64  `json:"created_at"`
	UID           string `json:"uid"`
	TenantID      string `json:"tenant_id"`
	Role          string `json:"role"`
	Scopes        string `json:"scopes"`
	Action        string `json:"action"`
	ResourceType  string `json:"resource_type"`
	ResourceID    string `json:"resource_id"`
	IP            string `json:"ip"`
	UA            string `json:"ua"`
	Device        string `json:"device"`
	Refer         string `json:"refer"`
	Success       uint8  `json:"success"`
	HTTPStatus    int32  `json:"http_status"`
	ErrCode       string `json:"err_code"`
	LatencyMS     int64  `json:"latency_ms"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	QuerySummary  string `json:"query_summary"`
	BodySummary   string `json:"body_summary"`
	RespSummary   string `json:"resp_summary"`
	RequestSizeB  int64  `json:"request_size_b"`
	ResponseSizeB int64  `json:"response_size_b"`
	RiskScore     int64  `json:"risk_score"`
	RiskAction    string `json:"risk_action"`
	RiskReasons   string `json:"risk_reasons"`
}

// AuditLogListResponse 后台审计日志列表响应。
type AuditLogListResponse struct {
	Total int64          `json:"total"`
	Items []AuditLogItem `json:"items"`
}

func ToAuditLogItem(m *auditmodel.AuditLog) AuditLogItem {
	if m == nil {
		return AuditLogItem{}
	}
	return AuditLogItem{
		ID:            m.ID,
		RID:           m.RID,
		TraceID:       m.TraceID,
		CreatedAt:     m.CreatedAt,
		UID:           m.UID,
		TenantID:      m.TenantID,
		Role:          m.Role,
		Scopes:        m.Scopes,
		Action:        m.Action,
		ResourceType:  m.ResourceType,
		ResourceID:    m.ResourceID,
		IP:            m.IP,
		UA:            m.UA,
		Device:        m.Device,
		Refer:         m.Refer,
		Success:       m.Success,
		HTTPStatus:    m.HTTPStatus,
		ErrCode:       m.ErrCode,
		LatencyMS:     m.LatencyMS,
		Method:        m.Method,
		Path:          m.Path,
		QuerySummary:  m.QuerySummary,
		BodySummary:   m.BodySummary,
		RespSummary:   m.RespSummary,
		RequestSizeB:  m.RequestSizeB,
		ResponseSizeB: m.ResponseSizeB,
		RiskScore:     m.RiskScore,
		RiskAction:    m.RiskAction,
		RiskReasons:   m.RiskReasons,
	}
}

func ToAuditLogDetail(m *auditmodel.AuditLog) AuditLogDetail {
	if m == nil {
		return AuditLogDetail{}
	}
	return AuditLogDetail{
		ID:            m.ID,
		RID:           m.RID,
		TraceID:       m.TraceID,
		CreatedAt:     m.CreatedAt,
		UID:           m.UID,
		TenantID:      m.TenantID,
		Role:          m.Role,
		Scopes:        m.Scopes,
		Action:        m.Action,
		ResourceType:  m.ResourceType,
		ResourceID:    m.ResourceID,
		IP:            m.IP,
		UA:            m.UA,
		Device:        m.Device,
		Refer:         m.Refer,
		Success:       m.Success,
		HTTPStatus:    m.HTTPStatus,
		ErrCode:       m.ErrCode,
		LatencyMS:     m.LatencyMS,
		Method:        m.Method,
		Path:          m.Path,
		QuerySummary:  m.QuerySummary,
		BodySummary:   m.BodySummary,
		RespSummary:   m.RespSummary,
		RequestSizeB:  m.RequestSizeB,
		ResponseSizeB: m.ResponseSizeB,
		RiskScore:     m.RiskScore,
		RiskAction:    m.RiskAction,
		RiskReasons:   m.RiskReasons,
	}
}

func ToAuditLogItems(list []*auditmodel.AuditLog) []AuditLogItem {
	if len(list) == 0 {
		return nil
	}
	items := make([]AuditLogItem, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		items = append(items, ToAuditLogItem(item))
	}
	if len(items) == 0 {
		return nil
	}
	return items
}
