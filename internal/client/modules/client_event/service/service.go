package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	clienteventdto "storeready_ai/internal/client/modules/client_event/dto"
	clienteventmodel "storeready_ai/internal/client/modules/client_event/model"
	clienteventrepo "storeready_ai/internal/client/modules/client_event/repo"
	"storeready_ai/internal/pkg/errors"
)

// Service 客户端埋点服务。
//
// 职责：
// 1. 收口客户端埋点上报与查询；
// 2. 统一补齐 tenant_id / uid / created_at；
// 3. 统一做最小必要参数校验与字符串清洗；
// 4. 避免 handler 直接依赖数据库模型。
type Service interface {
	// Report 上报单条客户端埋点事件。
	Report(ctx context.Context, tenantID, uid uint64, req clienteventdto.ReportClientEventReq) (*clienteventrepo.CreateBatchResult, error)

	// ReportBatch 批量上报客户端埋点事件。
	ReportBatch(ctx context.Context, tenantID, uid uint64, req clienteventdto.ReportClientEventsBatchReq) (*clienteventrepo.CreateBatchResult, error)

	// List 查询客户端埋点事件列表。
	List(ctx context.Context, tenantID uint64, req clienteventdto.ListClientEventsReq) (*clienteventdto.ListClientEventsResp, error)
}

type service struct {
	repo clienteventrepo.ClientEventRepo
}

// New 创建客户端埋点服务。
func New(repo clienteventrepo.ClientEventRepo) Service {
	return &service{repo: repo}
}

// Report 上报单条客户端埋点事件。
func (s *service) Report(ctx context.Context, tenantID, uid uint64, req clienteventdto.ReportClientEventReq) (*clienteventrepo.CreateBatchResult, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New(errors.CodeInternal, "client event service not configured")
	}

	event, err := s.buildEvent(tenantID, uid, &req.ClientEventItemReq)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, event)
}

// ReportBatch 批量上报客户端埋点事件。
func (s *service) ReportBatch(ctx context.Context, tenantID, uid uint64, req clienteventdto.ReportClientEventsBatchReq) (*clienteventrepo.CreateBatchResult, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New(errors.CodeInternal, "client event service not configured")
	}
	if len(req.Items) == 0 {
		return nil, errors.New(errors.CodeInvalidParam, "items required")
	}

	events := make([]*clienteventmodel.ClientEvent, 0, len(req.Items))
	for _, item := range req.Items {
		if item == nil {
			continue
		}
		event, err := s.buildEvent(tenantID, uid, item)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if len(events) == 0 {
		return nil, errors.New(errors.CodeInvalidParam, "items required")
	}

	return s.repo.CreateBatch(ctx, events)
}

func (s *service) buildEvent(tenantID, uid uint64, req *clienteventdto.ClientEventItemReq) (*clienteventmodel.ClientEvent, error) {
	if req == nil {
		return nil, errors.New(errors.CodeInvalidParam, "event item required")
	}

	eventGroup := strings.TrimSpace(req.EventGroup)
	if eventGroup == "" {
		return nil, errors.New(errors.CodeInvalidParam, "event_group required")
	}
	eventName := strings.TrimSpace(req.EventName)
	if eventName == "" {
		return nil, errors.New(errors.CodeInvalidParam, "event_name required")
	}
	platform := strings.TrimSpace(strings.ToLower(req.Platform))
	if platform == "" {
		return nil, errors.New(errors.CodeInvalidParam, "platform required")
	}

	payload := strings.TrimSpace(req.Payload)
	if payload != "" && !json.Valid([]byte(payload)) {
		return nil, errors.New(errors.CodeInvalidParam, "payload must be valid json")
	}

	now := uint64(time.Now().Unix())
	return &clienteventmodel.ClientEvent{
		TenantID:   tenantID,
		UID:        uid,
		EventID:    strings.TrimSpace(req.EventID),
		ReceivedAt: now,

		EventGroup:  eventGroup,
		EventName:   eventName,
		EventSource: strings.TrimSpace(strings.ToLower(req.EventSource)),
		Platform:    platform,

		AppVersion:  strings.TrimSpace(req.AppVersion),
		BuildNumber: strings.TrimSpace(req.BuildNumber),
		PackageName: strings.TrimSpace(req.PackageName),

		DeviceID:    strings.TrimSpace(req.DeviceID),
		DeviceModel: strings.TrimSpace(req.DeviceModel),
		OSVersion:   strings.TrimSpace(req.OSVersion),

		NetworkType:    strings.TrimSpace(strings.ToLower(req.NetworkType)),
		StoreAvailable: req.StoreAvailable,

		EventCode:    strings.TrimSpace(req.EventCode),
		EventMessage: strings.TrimSpace(req.EventMessage),
		Payload:      payload,

		CreatedAt: now,
	}, nil
}

// List 查询客户端埋点事件列表。
func (s *service) List(ctx context.Context, tenantID uint64, req clienteventdto.ListClientEventsReq) (*clienteventdto.ListClientEventsResp, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New(errors.CodeInternal, "client event service not configured")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	items, err := s.repo.List(ctx, clienteventrepo.ListClientEventsOption{
		TenantID:   tenantID,
		UID:        req.UID,
		EventGroup: strings.TrimSpace(req.EventGroup),
		EventName:  strings.TrimSpace(req.EventName),
		Platform:   strings.TrimSpace(strings.ToLower(req.Platform)),
		StartAt:    req.StartAt,
		EndAt:      req.EndAt,
		Offset:     offset,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	resp := &clienteventdto.ListClientEventsResp{
		Items: make([]*clienteventdto.ClientEventResp, 0, len(items)),
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		resp.Items = append(resp.Items, &clienteventdto.ClientEventResp{
			ID: item.ID,

			TenantID:   item.TenantID,
			UID:        item.UID,
			EventID:    item.EventID,
			ReceivedAt: item.ReceivedAt,

			EventGroup:  item.EventGroup,
			EventName:   item.EventName,
			EventSource: item.EventSource,
			Platform:    item.Platform,

			AppVersion:  item.AppVersion,
			BuildNumber: item.BuildNumber,
			PackageName: item.PackageName,

			DeviceID:    item.DeviceID,
			DeviceModel: item.DeviceModel,
			OSVersion:   item.OSVersion,

			NetworkType:    item.NetworkType,
			StoreAvailable: item.StoreAvailable,

			EventCode:    item.EventCode,
			EventMessage: item.EventMessage,
			Payload:      item.Payload,

			CreatedAt: item.CreatedAt,
		})
	}
	return resp, nil
}
