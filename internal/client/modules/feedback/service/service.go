package service

import (
	"context"
	"errors"
	"strings"

	"storeready_ai/internal/client/modules/feedback/dto"
	"storeready_ai/internal/contracts/feedback/model"

	"storeready_ai/internal/client/modules/feedback/repo"
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
	Create(ctx context.Context, tenantID uint64, uid uint64, req dto.CreateFeedbackReq, meta ClientMeta) (*dto.FeedbackVO, error)
}

type service struct {
	repo repo.Repository
}

// NewService 创建用户反馈服务。
func NewService(repo repo.Repository) Service {
	return &service{repo: repo}
}

// Create 创建用户反馈。
func (s *service) Create(ctx context.Context, tenantID uint64, uid uint64, req dto.CreateFeedbackReq, meta ClientMeta) (*dto.FeedbackVO, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrFeedbackContentRequired
	}

	feedback := &model.UserFeedback{
		TenantID:    tenantID,
		UID:         uid,
		Category:    model.NormalizeFeedbackCategory(req.Category),
		Title:       strings.TrimSpace(req.Title),
		Content:     content,
		Contact:     strings.TrimSpace(req.Contact),
		Status:      model.FeedbackStatusPending,
		Priority:    model.FeedbackPriorityNormal,
		AppVersion:  strings.TrimSpace(meta.AppVersion),
		BuildNumber: strings.TrimSpace(meta.BuildNumber),
		Platform:    strings.TrimSpace(meta.Platform),
		DeviceModel: strings.TrimSpace(meta.DeviceModel),
		OSVersion:   strings.TrimSpace(meta.OSVersion),
		Locale:      strings.TrimSpace(meta.Locale),
		Extra:       strings.TrimSpace(req.Extra),
	}

	if err := s.repo.Create(ctx, feedback); err != nil {
		return nil, err
	}

	return ToFeedbackVO(*feedback), nil
}

// ToFeedbackVO 将 model 转换为展示对象。
func ToFeedbackVO(feedback model.UserFeedback) *dto.FeedbackVO {
	return &dto.FeedbackVO{
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
