package service

import (
	"context"
	"log"
	"strings"

	"storeready_ai/internal/client/modules/project/dto"
	"storeready_ai/internal/client/modules/project/model"
	"storeready_ai/internal/client/modules/project/repo"
	"storeready_ai/internal/pkg/ai"
	errx "storeready_ai/internal/pkg/errors"
)

// ProjectService 项目领域 Service 层接口。
type ProjectService interface {
	// CreateProject 创建项目。
	CreateProject(ctx context.Context, tenantID, userID uint64, req dto.CreateProjectReq) (*dto.CreateProjectResp, error)
	// ListProjects 查询当前用户项目列表。
	ListProjects(ctx context.Context, tenantID, userID uint64, req dto.ListProjectsReq) (*dto.ListProjectsResp, error)
	// GetProjectDetail 查询项目详情和上架资料。
	GetProjectDetail(ctx context.Context, tenantID, userID uint64, req dto.GetProjectDetailReq) (*dto.ProjectDetailResp, error)
	// SaveProjectStoreInfo 保存项目上架资料。
	SaveProjectStoreInfo(ctx context.Context, tenantID, userID uint64, req dto.SaveProjectStoreInfoReq) (*dto.SaveProjectStoreInfoResp, error)
	// GenerateProjectStoreInfo AI 生成上架资料。
	GenerateProjectStoreInfo(ctx context.Context, tenantID, userID uint64, req dto.GenerateProjectStoreInfoReq) (*dto.GenerateProjectStoreInfoResp, error)
}

// serviceImpl ProjectService 的默认实现。
type serviceImpl struct {
	projects       repo.ProjectRepo
	storeInfoAIGen ai.StoreInfoGenerator
}

// New 创建 ProjectService。
func New(projects repo.ProjectRepo, storeInfoAIGen ai.StoreInfoGenerator) ProjectService {
	return &serviceImpl{
		projects:       projects,
		storeInfoAIGen: storeInfoAIGen,
	}
}

// CreateProject 创建项目。
func (s *serviceImpl) CreateProject(ctx context.Context, tenantID, userID uint64, req dto.CreateProjectReq) (*dto.CreateProjectResp, error) {
	if s == nil || s.projects == nil {
		return nil, errx.New(errx.CodeInternal, "project repo not configured")
	}
	if tenantID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	if userID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "user_id required")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errx.New(errx.CodeInvalidParam, "project name required")
	}
	if len([]rune(name)) > 100 {
		return nil, errx.New(errx.CodeInvalidParam, "project name too long")
	}

	platform := strings.TrimSpace(req.Platform)
	if platform != "" && platform != model.ProjectPlatformIOS && platform != model.ProjectPlatformAndroid {
		return nil, errx.New(errx.CodeInvalidParam, "invalid project platform")
	}

	description := strings.TrimSpace(req.Description)

	project := &model.Project{
		TenantID:    tenantID,
		UserID:      userID,
		Name:        name,
		Description: description,
		Platform:    platform,
		Status:      model.ProjectStatusDraft,
	}
	if err := s.projects.Create(ctx, project); err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "create project failed")
	}

	return &dto.CreateProjectResp{
		Project: toProjectItem(project),
	}, nil
}

// ListProjects 查询当前用户项目列表。
func (s *serviceImpl) ListProjects(ctx context.Context, tenantID, userID uint64, req dto.ListProjectsReq) (*dto.ListProjectsResp, error) {
	if s == nil || s.projects == nil {
		return nil, errx.New(errx.CodeInternal, "project repo not configured")
	}
	if tenantID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	if userID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "user_id required")
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

	projects, total, err := s.projects.ListByUser(ctx, tenantID, userID, page, pageSize)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "list projects failed")
	}

	items := make([]dto.ProjectItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, toProjectItem(project))
	}

	return &dto.ListProjectsResp{
		List:     items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

func toProjectItem(project *model.Project) dto.ProjectItem {
	if project == nil {
		return dto.ProjectItem{}
	}
	return dto.ProjectItem{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		Platform:    project.Platform,
		Status:      project.Status,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}

// GetProjectDetail 查询项目详情和上架资料。
func (s *serviceImpl) GetProjectDetail(ctx context.Context, tenantID, userID uint64, req dto.GetProjectDetailReq) (*dto.ProjectDetailResp, error) {
	if s == nil || s.projects == nil {
		return nil, errx.New(errx.CodeInternal, "project repo not configured")
	}
	if tenantID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	if userID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "user_id required")
	}
	if req.ProjectID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "project_id required")
	}

	project, err := s.projects.GetProjectByID(ctx, tenantID, userID, req.ProjectID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "get project failed")
	}

	resp := &dto.ProjectDetailResp{
		Project: toProjectItem(project),
	}
	info, err := s.projects.GetStoreInfoByProjectID(ctx, tenantID, userID, req.ProjectID)
	if err == nil {
		item := toProjectStoreInfoItem(info)
		resp.StoreInfo = &item
	}

	return resp, nil
}

// SaveProjectStoreInfo 保存项目上架资料。
func (s *serviceImpl) SaveProjectStoreInfo(ctx context.Context, tenantID, userID uint64, req dto.SaveProjectStoreInfoReq) (*dto.SaveProjectStoreInfoResp, error) {
	if s == nil || s.projects == nil {
		return nil, errx.New(errx.CodeInternal, "project repo not configured")
	}
	if tenantID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	if userID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "user_id required")
	}
	if req.ProjectID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "project_id required")
	}

	if _, err := s.projects.GetProjectByID(ctx, tenantID, userID, req.ProjectID); err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "get project failed")
	}

	info := &model.ProjectStoreInfo{
		TenantID:         tenantID,
		UserID:           userID,
		ProjectID:        req.ProjectID,
		AppName:          strings.TrimSpace(req.AppName),
		Subtitle:         strings.TrimSpace(req.Subtitle),
		Keywords:         strings.TrimSpace(req.Keywords),
		ShortDescription: strings.TrimSpace(req.ShortDescription),
		FullDescription:  strings.TrimSpace(req.FullDescription),
		Category:         strings.TrimSpace(req.Category),
		ContentRating:    strings.TrimSpace(req.ContentRating),
		PrivacyPolicyURL: strings.TrimSpace(req.PrivacyPolicyURL),
		SupportURL:       strings.TrimSpace(req.SupportURL),
		MarketingURL:     strings.TrimSpace(req.MarketingURL),
		Copyright:        strings.TrimSpace(req.Copyright),
		ContactEmail:     strings.TrimSpace(req.ContactEmail),
		Status:           strings.TrimSpace(req.Status),
	}
	if info.Status == "" {
		info.Status = model.ProjectStoreInfoStatusDraft
	}
	if info.Status != model.ProjectStoreInfoStatusDraft && info.Status != model.ProjectStoreInfoStatusReady {
		return nil, errx.New(errx.CodeInvalidParam, "invalid store info status")
	}

	if err := s.projects.SaveStoreInfo(ctx, info); err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "save project store info failed")
	}

	return &dto.SaveProjectStoreInfoResp{
		StoreInfo: toProjectStoreInfoItem(info),
	}, nil
}

func toProjectStoreInfoItem(info *model.ProjectStoreInfo) dto.ProjectStoreInfoItem {
	if info == nil {
		return dto.ProjectStoreInfoItem{}
	}
	return dto.ProjectStoreInfoItem{
		ID:               info.ID,
		ProjectID:        info.ProjectID,
		AppName:          info.AppName,
		Subtitle:         info.Subtitle,
		Keywords:         info.Keywords,
		ShortDescription: info.ShortDescription,
		FullDescription:  info.FullDescription,
		Category:         info.Category,
		ContentRating:    info.ContentRating,
		PrivacyPolicyURL: info.PrivacyPolicyURL,
		SupportURL:       info.SupportURL,
		MarketingURL:     info.MarketingURL,
		Copyright:        info.Copyright,
		ContactEmail:     info.ContactEmail,
		Status:           info.Status,
		CreatedAt:        info.CreatedAt,
		UpdatedAt:        info.UpdatedAt,
	}
}

// GenerateProjectStoreInfo AI 生成上架资料。
func (s *serviceImpl) GenerateProjectStoreInfo(ctx context.Context, tenantID, userID uint64, req dto.GenerateProjectStoreInfoReq) (*dto.GenerateProjectStoreInfoResp, error) {
	if s == nil || s.projects == nil {
		return nil, errx.New(errx.CodeInternal, "project repo not configured")
	}
	if s.storeInfoAIGen == nil {
		return nil, errx.New(errx.CodeInternal, "store info ai generator not configured")
	}
	if tenantID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "tenant_id required")
	}
	if userID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "user_id required")
	}
	if req.ProjectID == 0 {
		return nil, errx.New(errx.CodeInvalidParam, "project_id required")
	}

	project, err := s.projects.GetProjectByID(ctx, tenantID, userID, req.ProjectID)
	if err != nil {
		return nil, errx.Wrap(err, errx.CodeInternal, "get project failed")
	}

	input := ai.StoreInfoInput{
		ProjectID:   req.ProjectID,
		Name:        project.Name,
		Description: project.Description,
		Platform:    project.Platform,
		Status:      project.Status,
	}

	info, err := s.projects.GetStoreInfoByProjectID(ctx, tenantID, userID, req.ProjectID)
	if err == nil && info != nil {
		input.ExistingAppName = info.AppName
		input.ExistingSubtitle = info.Subtitle
		input.ExistingKeywords = info.Keywords
		input.ExistingShortDescription = info.ShortDescription
		input.ExistingFullDescription = info.FullDescription
		input.ExistingCategory = info.Category
		input.ExistingContentRating = info.ContentRating
	}

	generated, err := s.storeInfoAIGen.GenerateStoreInfo(ctx, input)
	if err != nil {
		log.Printf("[project_ai_generate_failed] tenant_id=%d user_id=%d project_id=%d err=%v", tenantID, userID, req.ProjectID, err)
		return nil, errx.Wrap(err, errx.CodeInternal, "generate project store info failed")
	}
	if generated == nil {
		return nil, errx.New(errx.CodeInternal, "generate project store info empty")
	}

	return &dto.GenerateProjectStoreInfoResp{
		Subtitle:         generated.Subtitle,
		ShortDescription: generated.ShortDescription,
		FullDescription:  generated.FullDescription,
		Keywords:         generated.Keywords,
	}, nil
}
