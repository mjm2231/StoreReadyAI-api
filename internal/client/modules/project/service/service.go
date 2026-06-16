package service

import (
	"context"
	"strings"

	"storeready_ai/internal/client/modules/project/dto"
	"storeready_ai/internal/client/modules/project/model"
	"storeready_ai/internal/client/modules/project/repo"
	errx "storeready_ai/internal/pkg/errors"
)

// ProjectService 项目领域 Service 层接口。
type ProjectService interface {
	// CreateProject 创建项目。
	CreateProject(ctx context.Context, tenantID, userID uint64, req dto.CreateProjectReq) (*dto.CreateProjectResp, error)
	// ListProjects 查询当前用户项目列表。
	ListProjects(ctx context.Context, tenantID, userID uint64, req dto.ListProjectsReq) (*dto.ListProjectsResp, error)
}

// serviceImpl ProjectService 的默认实现。
type serviceImpl struct {
	projects repo.ProjectRepo
}

// New 创建 ProjectService。
func New(projects repo.ProjectRepo) ProjectService {
	return &serviceImpl{projects: projects}
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

	project := &model.Project{
		TenantID: tenantID,
		UserID:   userID,
		Name:     name,
		Platform: platform,
		Status:   model.ProjectStatusDraft,
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
		ID:        project.ID,
		Name:      project.Name,
		Platform:  project.Platform,
		Status:    project.Status,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}
}
