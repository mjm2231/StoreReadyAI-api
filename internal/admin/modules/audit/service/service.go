package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	auditdto "storeready_ai/internal/admin/modules/audit/dto"
	auditmodel "storeready_ai/internal/admin/modules/audit/model"
	auditrepo "storeready_ai/internal/admin/modules/audit/repo"
)

var (
	ErrNilRepo          = errors.New("admin audit service: repo is nil")
	ErrInvalidID        = errors.New("admin audit service: invalid id")
	ErrInvalidAction    = errors.New("admin audit service: invalid action")
	ErrAuditLogNotFound = errors.New("admin audit service: audit log not found")
)

// Service 是后台审计日志服务接口。
//
// 约定：
// 1. admin 模块统一采用“接口 + 实现”结构；
// 2. handler 或其它模块依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 审计日志默认只追加，不提供更新/删除能力。
type Service interface {
	Create(ctx context.Context, req auditdto.CreateAuditLogRequest) (auditdto.AuditLogDetail, error)
	BatchCreate(ctx context.Context, reqs []auditdto.CreateAuditLogRequest) error
	GetDetail(ctx context.Context, req auditdto.GetAuditLogDetailRequest) (auditdto.AuditLogDetail, error)
	List(ctx context.Context, req auditdto.AuditLogListRequest) (auditdto.AuditLogListResponse, error)
}

// service 是 Service 的默认实现。
type service struct {
	repo auditrepo.Repository
	now  func() time.Time
}

func New(repo auditrepo.Repository) (Service, error) {
	if repo == nil || repo.DB() == nil {
		return nil, ErrNilRepo
	}
	return &service{
		repo: repo,
		now:  time.Now,
	}, nil
}

func (s *service) SetNow(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = now
}

func (s *service) Create(ctx context.Context, req auditdto.CreateAuditLogRequest) (auditdto.AuditLogDetail, error) {
	if s == nil {
		return auditdto.AuditLogDetail{}, ErrNilRepo
	}
	req = req.Normalize()
	if req.Action == "" {
		return auditdto.AuditLogDetail{}, ErrInvalidAction
	}
	createdAt := req.CreatedAt
	if createdAt <= 0 {
		createdAt = s.nowUnix()
	}
	log := &auditmodel.AuditLog{
		RID:           req.RID,
		TraceID:       req.TraceID,
		CreatedAt:     createdAt,
		UID:           req.UID,
		TenantID:      req.TenantID,
		Role:          req.Role,
		Scopes:        req.Scopes,
		Action:        req.Action,
		ResourceType:  req.ResourceType,
		ResourceID:    req.ResourceID,
		IP:            req.IP,
		UA:            req.UA,
		Device:        req.Device,
		Refer:         req.Refer,
		Success:       normalizeBoolFlag(req.Success),
		HTTPStatus:    req.HTTPStatus,
		ErrCode:       req.ErrCode,
		LatencyMS:     req.LatencyMS,
		Method:        req.Method,
		Path:          req.Path,
		QuerySummary:  req.QuerySummary,
		BodySummary:   req.BodySummary,
		RespSummary:   req.RespSummary,
		RequestSizeB:  req.RequestSizeB,
		ResponseSizeB: req.ResponseSizeB,
		RiskScore:     req.RiskScore,
		RiskAction:    req.RiskAction,
		RiskReasons:   req.RiskReasons,
	}
	if err := s.repo.Create(ctx, log); err != nil {
		return auditdto.AuditLogDetail{}, err
	}
	return auditdto.ToAuditLogDetail(log), nil
}

func (s *service) BatchCreate(ctx context.Context, reqs []auditdto.CreateAuditLogRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	if len(reqs) == 0 {
		return nil
	}
	logs := make([]*auditmodel.AuditLog, 0, len(reqs))
	for _, req := range reqs {
		req = req.Normalize()
		if req.Action == "" {
			continue
		}
		createdAt := req.CreatedAt
		if createdAt <= 0 {
			createdAt = s.nowUnix()
		}
		logs = append(logs, &auditmodel.AuditLog{
			RID:           req.RID,
			TraceID:       req.TraceID,
			CreatedAt:     createdAt,
			UID:           req.UID,
			TenantID:      req.TenantID,
			Role:          req.Role,
			Scopes:        req.Scopes,
			Action:        req.Action,
			ResourceType:  req.ResourceType,
			ResourceID:    req.ResourceID,
			IP:            req.IP,
			UA:            req.UA,
			Device:        req.Device,
			Refer:         req.Refer,
			Success:       normalizeBoolFlag(req.Success),
			HTTPStatus:    req.HTTPStatus,
			ErrCode:       req.ErrCode,
			LatencyMS:     req.LatencyMS,
			Method:        req.Method,
			Path:          req.Path,
			QuerySummary:  req.QuerySummary,
			BodySummary:   req.BodySummary,
			RespSummary:   req.RespSummary,
			RequestSizeB:  req.RequestSizeB,
			ResponseSizeB: req.ResponseSizeB,
			RiskScore:     req.RiskScore,
			RiskAction:    req.RiskAction,
			RiskReasons:   req.RiskReasons,
		})
	}
	if len(logs) == 0 {
		return nil
	}
	return s.repo.BatchCreate(ctx, logs)
}

func (s *service) GetDetail(ctx context.Context, req auditdto.GetAuditLogDetailRequest) (auditdto.AuditLogDetail, error) {
	if s == nil {
		return auditdto.AuditLogDetail{}, ErrNilRepo
	}
	if req.ID == 0 {
		return auditdto.AuditLogDetail{}, ErrInvalidID
	}
	log, err := s.repo.GetByID(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return auditdto.AuditLogDetail{}, ErrAuditLogNotFound
		}
		return auditdto.AuditLogDetail{}, err
	}
	return auditdto.ToAuditLogDetail(log), nil
}

func (s *service) List(ctx context.Context, req auditdto.AuditLogListRequest) (auditdto.AuditLogListResponse, error) {
	if s == nil {
		return auditdto.AuditLogListResponse{}, ErrNilRepo
	}
	req = req.Normalize()
	offset := (req.Page.PageSize - 1) * req.Page.Page
	limit := req.Page.PageSize
	filter := auditrepo.ListFilter{
		RID:          req.RID,
		TraceID:      req.TraceID,
		UID:          req.UID,
		TenantID:     req.TenantID,
		Role:         req.Role,
		Action:       req.Action,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		ErrCode:      req.ErrCode,
		Method:       req.Method,
		Path:         req.Path,
		Keyword:      req.Keyword,
		Offset:       offset,
		Limit:        limit,
	}
	if req.Success != nil {
		v := normalizeBoolFlag(*req.Success)
		filter.Success = &v
	}
	if req.HTTPStatus != nil {
		v := *req.HTTPStatus
		filter.HTTPStatus = &v
	}
	if req.CreatedFrom != nil {
		v := *req.CreatedFrom
		filter.CreatedFrom = &v
	}
	if req.CreatedTo != nil {
		v := *req.CreatedTo
		filter.CreatedTo = &v
	}
	if len(req.IDs) > 0 {
		filter.IDs = append([]uint64(nil), req.IDs...)
	}
	count, err := s.repo.Count(ctx, filter)
	if err != nil {
		return auditdto.AuditLogListResponse{}, err
	}
	items, err := s.repo.List(ctx, filter)
	if err != nil {
		return auditdto.AuditLogListResponse{}, err
	}
	return auditdto.AuditLogListResponse{
		Total: count,
		Items: auditdto.ToAuditLogItems(items),
	}, nil
}

func (s *service) nowUnix() int64 {
	if s == nil || s.now == nil {
		return time.Now().Unix()
	}
	return s.now().Unix()
}

func normalizeBoolFlag(v uint8) uint8 {
	if v > 0 {
		return 1
	}
	return 0
}
