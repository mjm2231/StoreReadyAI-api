package service

import (
	"context"
	"errors"
	"strings"

	"storeready_ai/internal/admin/modules/feedback/dto"
	"storeready_ai/internal/contracts/feedback/model"

	"storeready_ai/internal/admin/modules/feedback/repo"
)

var (
	ErrFeedbackContentRequired = errors.New("feedback content required")
	ErrFeedbackNotFound        = errors.New("feedback not found")
	ErrInvalidFeedbackStatus   = errors.New("invalid feedback status")
	ErrInvalidFeedbackPriority = errors.New("invalid feedback priority")
)

// ClientMeta 客户端环境信息。
type ClientMeta struct {
	AppVersion  string
	BuildNumber string
	Platform    string
	DeviceModel string
	OSVersion   string
	Locale      string
}

// Service 用户反馈业务接口。
type Service interface {
	Detail(ctx context.Context, tenantID uint64, req dto.FeedbackDetailReq) (*dto.Feedback, error)
	List(ctx context.Context, tenantID uint64, req dto.FeedbackListReq) (*dto.FeedbackListResp, error)
	UpdateStatus(ctx context.Context, tenantID uint64, handledBy uint64, req dto.UpdateFeedbackStatusReq) error
	Reply(ctx context.Context, tenantID uint64, handledBy uint64, req dto.ReplyFeedbackReq) error
	Delete(ctx context.Context, tenantID uint64, req dto.DeleteFeedbackReq) error
	Options(ctx context.Context) dto.FeedbackOptionsResp
}

type service struct {
	repo repo.Repository
}

// NewService 创建用户反馈服务。
func NewService(repo repo.Repository) Service {
	return &service{repo: repo}
}

// Detail 获取反馈详情。
func (s *service) Detail(ctx context.Context, tenantID uint64, req dto.FeedbackDetailReq) (*dto.Feedback, error) {
	feedback, err := s.repo.GetByID(ctx, tenantID, req.ID)
	if err != nil {
		return nil, err
	}
	if feedback == nil {
		return nil, ErrFeedbackNotFound
	}
	return ToFeedbackVO(*feedback), nil
}

// List 查询反馈列表。
func (s *service) List(ctx context.Context, tenantID uint64, req dto.FeedbackListReq) (*dto.FeedbackListResp, error) {
	list, total, err := s.repo.List(ctx, tenantID, req)
	if err != nil {
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	vos := make([]dto.Feedback, 0, len(list))
	for _, item := range list {
		vo := ToFeedbackVO(item)
		if vo != nil {
			vos = append(vos, *vo)
		}
	}

	return &dto.FeedbackListResp{
		List:  vos,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// UpdateStatus 更新反馈处理状态。
func (s *service) UpdateStatus(ctx context.Context, tenantID uint64, handledBy uint64, req dto.UpdateFeedbackStatusReq) error {
	if !IsValidFeedbackStatus(req.Status) {
		return ErrInvalidFeedbackStatus
	}
	if req.Priority > 0 && !IsValidFeedbackPriority(req.Priority) {
		return ErrInvalidFeedbackPriority
	}

	return s.repo.UpdateStatus(ctx, tenantID, req.ID, req.Status, req.Priority, handledBy)
}

// Reply 回复反馈。
func (s *service) Reply(ctx context.Context, tenantID uint64, handledBy uint64, req dto.ReplyFeedbackReq) error {
	status := req.Status
	if status == 0 {
		status = model.FeedbackStatusResolved
	}
	if !IsValidFeedbackStatus(status) {
		return ErrInvalidFeedbackStatus
	}

	return s.repo.Reply(ctx, tenantID, req.ID, strings.TrimSpace(req.ReplyContent), status, handledBy)
}

// Delete 软删除反馈。
func (s *service) Delete(ctx context.Context, tenantID uint64, req dto.DeleteFeedbackReq) error {
	return s.repo.SoftDelete(ctx, tenantID, req.ID)
}

// Options 获取反馈相关枚举选项。
func (s *service) Options(ctx context.Context) dto.FeedbackOptionsResp {
	return dto.FeedbackOptionsResp{
		Categories: FeedbackCategoryOptions(),
		Statuses:   FeedbackStatusOptions(),
		Priorities: FeedbackPriorityOptions(),
	}
}

// ToFeedbackVO 将 model 转换为展示对象。
func ToFeedbackVO(feedback model.UserFeedback) *dto.Feedback {
	return &dto.Feedback{
		ID:       feedback.ID,
		TenantID: feedback.TenantID,
		UID:      feedback.UID,

		Category:         feedback.Category,
		CategoryLabel:    FeedbackCategoryLabel(feedback.Category),
		CategoryLabelKey: FeedbackCategoryLabelKey(feedback.Category),

		Title:   feedback.Title,
		Content: feedback.Content,
		Contact: feedback.Contact,

		Status:         feedback.Status,
		StatusLabel:    FeedbackStatusLabel(feedback.Status),
		StatusLabelKey: FeedbackStatusLabelKey(feedback.Status),

		Priority:         feedback.Priority,
		PriorityLabel:    FeedbackPriorityLabel(feedback.Priority),
		PriorityLabelKey: FeedbackPriorityLabelKey(feedback.Priority),

		ReplyContent: feedback.ReplyContent,
		HandledBy:    feedback.HandledBy,
		HandledAt:    feedback.HandledAt,

		AppVersion:  feedback.AppVersion,
		BuildNumber: feedback.BuildNumber,
		Platform:    feedback.Platform,
		DeviceModel: feedback.DeviceModel,
		OSVersion:   feedback.OSVersion,
		Locale:      feedback.Locale,

		Extra: feedback.Extra,

		CreatedAt: feedback.CreatedAt,
		UpdatedAt: feedback.UpdatedAt,
	}
}

// FeedbackCategoryOptions 反馈分类选项。
func FeedbackCategoryOptions() []dto.FeedbackOption {
	return []dto.FeedbackOption{
		{Value: model.FeedbackCategoryGeneral, Label: "普通反馈", LabelKey: "feedback.category.general"},
		{Value: model.FeedbackCategoryBug, Label: "问题报错", LabelKey: "feedback.category.bug"},
		{Value: model.FeedbackCategorySuggestion, Label: "功能建议", LabelKey: "feedback.category.suggestion"},
		{Value: model.FeedbackCategoryBilling, Label: "支付订阅", LabelKey: "feedback.category.billing"},
		{Value: model.FeedbackCategoryAccount, Label: "账号登录", LabelKey: "feedback.category.account"},
		{Value: model.FeedbackCategoryOther, Label: "其他", LabelKey: "feedback.category.other"},
	}
}

// FeedbackCategoryLabel 返回反馈分类中文展示文案。
func FeedbackCategoryLabel(category uint8) string {
	switch model.NormalizeFeedbackCategory(category) {
	case model.FeedbackCategoryGeneral:
		return "普通反馈"
	case model.FeedbackCategoryBug:
		return "问题报错"
	case model.FeedbackCategorySuggestion:
		return "功能建议"
	case model.FeedbackCategoryBilling:
		return "支付订阅"
	case model.FeedbackCategoryAccount:
		return "账号登录"
	case model.FeedbackCategoryOther:
		return "其他"
	default:
		return "其他"
	}
}

// FeedbackCategoryLabelKey 返回反馈分类 i18n key。
func FeedbackCategoryLabelKey(category uint8) string {
	switch model.NormalizeFeedbackCategory(category) {
	case model.FeedbackCategoryGeneral:
		return "feedback.category.general"
	case model.FeedbackCategoryBug:
		return "feedback.category.bug"
	case model.FeedbackCategorySuggestion:
		return "feedback.category.suggestion"
	case model.FeedbackCategoryBilling:
		return "feedback.category.billing"
	case model.FeedbackCategoryAccount:
		return "feedback.category.account"
	case model.FeedbackCategoryOther:
		return "feedback.category.other"
	default:
		return "feedback.category.other"
	}
}

// FeedbackStatusOptions 反馈状态选项。
func FeedbackStatusOptions() []dto.FeedbackOption {
	return []dto.FeedbackOption{
		{Value: model.FeedbackStatusPending, Label: "待处理", LabelKey: "feedback.status.pending"},
		{Value: model.FeedbackStatusProcessing, Label: "处理中", LabelKey: "feedback.status.processing"},
		{Value: model.FeedbackStatusResolved, Label: "已处理", LabelKey: "feedback.status.resolved"},
		{Value: model.FeedbackStatusClosed, Label: "已关闭", LabelKey: "feedback.status.closed"},
	}
}

// IsValidFeedbackStatus 判断反馈状态是否合法。
func IsValidFeedbackStatus(status uint8) bool {
	switch status {
	case model.FeedbackStatusPending,
		model.FeedbackStatusProcessing,
		model.FeedbackStatusResolved,
		model.FeedbackStatusClosed:
		return true
	default:
		return false
	}
}

// FeedbackStatusLabel 返回反馈状态中文展示文案。
func FeedbackStatusLabel(status uint8) string {
	switch status {
	case model.FeedbackStatusPending:
		return "待处理"
	case model.FeedbackStatusProcessing:
		return "处理中"
	case model.FeedbackStatusResolved:
		return "已处理"
	case model.FeedbackStatusClosed:
		return "已关闭"
	default:
		return "未知"
	}
}

// FeedbackStatusLabelKey 返回反馈状态 i18n key。
func FeedbackStatusLabelKey(status uint8) string {
	switch status {
	case model.FeedbackStatusPending:
		return "feedback.status.pending"
	case model.FeedbackStatusProcessing:
		return "feedback.status.processing"
	case model.FeedbackStatusResolved:
		return "feedback.status.resolved"
	case model.FeedbackStatusClosed:
		return "feedback.status.closed"
	default:
		return "feedback.status.unknown"
	}
}

// FeedbackPriorityOptions 反馈优先级选项。
func FeedbackPriorityOptions() []dto.FeedbackOption {
	return []dto.FeedbackOption{
		{Value: model.FeedbackPriorityLow, Label: "低", LabelKey: "feedback.priority.low"},
		{Value: model.FeedbackPriorityNormal, Label: "普通", LabelKey: "feedback.priority.normal"},
		{Value: model.FeedbackPriorityHigh, Label: "高", LabelKey: "feedback.priority.high"},
		{Value: model.FeedbackPriorityUrgent, Label: "紧急", LabelKey: "feedback.priority.urgent"},
	}
}

// IsValidFeedbackPriority 判断反馈优先级是否合法。
func IsValidFeedbackPriority(priority uint8) bool {
	switch priority {
	case model.FeedbackPriorityLow,
		model.FeedbackPriorityNormal,
		model.FeedbackPriorityHigh,
		model.FeedbackPriorityUrgent:
		return true
	default:
		return false
	}
}

// FeedbackPriorityLabel 返回反馈优先级中文展示文案。
func FeedbackPriorityLabel(priority uint8) string {
	switch priority {
	case model.FeedbackPriorityLow:
		return "低"
	case model.FeedbackPriorityNormal:
		return "普通"
	case model.FeedbackPriorityHigh:
		return "高"
	case model.FeedbackPriorityUrgent:
		return "紧急"
	default:
		return "未知"
	}
}

// FeedbackPriorityLabelKey 返回反馈优先级 i18n key。
func FeedbackPriorityLabelKey(priority uint8) string {
	switch priority {
	case model.FeedbackPriorityLow:
		return "feedback.priority.low"
	case model.FeedbackPriorityNormal:
		return "feedback.priority.normal"
	case model.FeedbackPriorityHigh:
		return "feedback.priority.high"
	case model.FeedbackPriorityUrgent:
		return "feedback.priority.urgent"
	default:
		return "feedback.priority.unknown"
	}
}
