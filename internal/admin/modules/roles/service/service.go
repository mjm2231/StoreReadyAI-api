package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	roledto "storeready_ai/internal/admin/modules/roles/dto"
	rolemodel "storeready_ai/internal/admin/modules/roles/model"
	rolerepo "storeready_ai/internal/admin/modules/roles/repo"
)

var (
	ErrNilRepo            = errors.New("admin role service: repo is nil")
	ErrInvalidID          = errors.New("admin role service: invalid id")
	ErrInvalidName        = errors.New("admin role service: invalid name")
	ErrInvalidCode        = errors.New("admin role service: invalid code")
	ErrInvalidStatus      = errors.New("admin role service: invalid status")
	ErrRoleCodeExists     = errors.New("admin role service: role code already exists")
	ErrRoleNotFound       = errors.New("admin role service: role not found")
	ErrSystemRoleReadonly = errors.New("admin role service: system role is readonly")
)

// Service 是后台角色服务接口。
//
// 约定：
// 1. admin 模块统一采用“接口 + 实现”结构；
// 2. handler 依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 当前聚焦后台角色管理最小必要能力：列表、详情、创建、更新、状态更新、删除。
type Service interface {
	List(ctx context.Context, req roledto.RoleListRequest) (roledto.RoleListResponse, error)
	GetDetail(ctx context.Context, req roledto.GetRoleDetailRequest) (roledto.RoleDetail, error)
	Create(ctx context.Context, req roledto.CreateRoleRequest) (roledto.RoleDetail, error)
	Update(ctx context.Context, req roledto.UpdateRoleRequest) (roledto.RoleDetail, error)
	UpdateStatus(ctx context.Context, req roledto.UpdateRoleStatusRequest) error
	Delete(ctx context.Context, req roledto.DeleteRoleRequest) error

	ListRolesByIds(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]roledto.RoleItem, error)
}

// service 是 Service 的默认实现。
type service struct {
	repo rolerepo.Repository
	now  func() time.Time
}

func New(repo rolerepo.Repository) (Service, error) {
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

func (s *service) List(ctx context.Context, req roledto.RoleListRequest) (roledto.RoleListResponse, error) {
	if s == nil {
		return roledto.RoleListResponse{}, ErrNilRepo
	}
	req = req.Normalize()

	filter := rolerepo.ListFilter{
		Keyword: req.Keyword,
		Offset:  req.Offset,
		Limit:   req.Limit,
	}
	if req.Status != nil {
		status := *req.Status
		filter.Status = &status
	}
	if req.IsSystem != nil {
		isSystem := normalizeBoolFlag(*req.IsSystem)
		filter.IsSystem = &isSystem
	}
	if len(req.IDs) > 0 {
		filter.IDs = append([]uint64(nil), req.IDs...)
	}

	total, err := s.repo.Count(ctx, req.TenantID, filter)
	if err != nil {
		return roledto.RoleListResponse{}, err
	}
	items, err := s.repo.List(ctx, req.TenantID, filter)
	if err != nil {
		return roledto.RoleListResponse{}, err
	}
	return roledto.RoleListResponse{
		Total: total,
		Items: roledto.ToRoleItems(items),
	}, nil
}

func (s *service) GetDetail(ctx context.Context, req roledto.GetRoleDetailRequest) (roledto.RoleDetail, error) {
	if s == nil {
		return roledto.RoleDetail{}, ErrNilRepo
	}
	if req.ID == 0 {
		return roledto.RoleDetail{}, ErrInvalidID
	}
	role, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return roledto.RoleDetail{}, ErrRoleNotFound
		}
		return roledto.RoleDetail{}, err
	}
	return roledto.ToRoleDetail(role), nil
}

func (s *service) Create(ctx context.Context, req roledto.CreateRoleRequest) (roledto.RoleDetail, error) {
	if s == nil {
		return roledto.RoleDetail{}, ErrNilRepo
	}
	req = req.Normalize()
	if err := validateCreateRequest(req); err != nil {
		return roledto.RoleDetail{}, err
	}

	exists, err := s.repo.ExistsByCode(ctx, req.TenantID, req.Code, 0)
	if err != nil {
		return roledto.RoleDetail{}, err
	}
	if exists {
		return roledto.RoleDetail{}, ErrRoleCodeExists
	}

	nowUnix := s.nowUnix()
	role := &rolemodel.AdminRole{
		TenantID:  req.TenantID,
		Name:      req.Name,
		Code:      req.Code,
		Status:    req.Status,
		Sort:      req.Sort,
		IsSystem:  normalizeBoolFlag(req.IsSystem),
		Remark:    req.Remark,
		CreatedAt: nowUnix,
		UpdatedAt: nowUnix,
	}
	if err := s.repo.Create(ctx, role); err != nil {
		return roledto.RoleDetail{}, err
	}
	return roledto.ToRoleDetail(role), nil
}

func (s *service) Update(ctx context.Context, req roledto.UpdateRoleRequest) (roledto.RoleDetail, error) {
	if s == nil {
		return roledto.RoleDetail{}, ErrNilRepo
	}
	req = req.Normalize()
	if err := validateUpdateRequest(req); err != nil {
		return roledto.RoleDetail{}, err
	}

	role, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return roledto.RoleDetail{}, ErrRoleNotFound
		}
		return roledto.RoleDetail{}, err
	}
	if role.IsSystemBuiltin() {
		return roledto.RoleDetail{}, ErrSystemRoleReadonly
	}

	if req.Code != role.Code {
		exists, err := s.repo.ExistsByCode(ctx, req.TenantID, req.Code, req.ID)
		if err != nil {
			return roledto.RoleDetail{}, err
		}
		if exists {
			return roledto.RoleDetail{}, ErrRoleCodeExists
		}
	}

	role.Name = req.Name
	role.Code = req.Code
	role.Status = req.Status
	role.Sort = req.Sort
	role.IsSystem = normalizeBoolFlag(req.IsSystem)
	role.Remark = req.Remark
	role.UpdatedAt = s.nowUnix()

	if err := s.repo.Update(ctx, role); err != nil {
		return roledto.RoleDetail{}, err
	}
	return roledto.ToRoleDetail(role), nil
}

func (s *service) UpdateStatus(ctx context.Context, req roledto.UpdateRoleStatusRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	if req.ID == 0 {
		return ErrInvalidID
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}

	role, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if role.IsSystemBuiltin() {
		return ErrSystemRoleReadonly
	}
	return s.repo.UpdateStatus(ctx, req.TenantID, req.ID, req.Status, s.nowUnix())
}

func (s *service) Delete(ctx context.Context, req roledto.DeleteRoleRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	if req.ID == 0 {
		return ErrInvalidID
	}

	role, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if role.IsSystemBuiltin() {
		return ErrSystemRoleReadonly
	}
	nowUnix := s.nowUnix()
	return s.repo.SoftDelete(ctx, req.TenantID, req.ID, nowUnix, nowUnix)
}

func (s *service) ListRolesByIds(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]roledto.RoleItem, error) {
	if s == nil {
		return nil, ErrNilRepo
	}
	if tenantID == 0 {
		return nil, ErrInvalidID
	}
	if len(roleIDs) == 0 {
		return []roledto.RoleItem{}, nil
	}

	modelroles, err := s.repo.GetByIDs(ctx, tenantID, roleIDs)
	if err != nil {
		return nil, err
	}
	roles := make([]roledto.RoleItem, 0, len(modelroles))
	for _, r := range modelroles {
		if r == nil {
			continue
		}
		roles = append(roles, roledto.RoleItem{
			ID:       r.ID,
			Name:     r.Name,
			Code:     r.Code,
			Status:   r.Status,
			IsSystem: r.IsSystem,
		})
	}
	return roles, nil
}

func (s *service) nowUnix() uint64 {
	if s == nil || s.now == nil {
		return uint64(time.Now().Unix())
	}
	return uint64(s.now().Unix())
}

func validateCreateRequest(req roledto.CreateRoleRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrInvalidName
	}
	if strings.TrimSpace(req.Code) == "" {
		return ErrInvalidCode
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}
	return nil
}

func validateUpdateRequest(req roledto.UpdateRoleRequest) error {
	if req.ID == 0 {
		return ErrInvalidID
	}
	if strings.TrimSpace(req.Name) == "" {
		return ErrInvalidName
	}
	if strings.TrimSpace(req.Code) == "" {
		return ErrInvalidCode
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}
	return nil
}

func isValidStatus(v uint8) bool {
	switch v {
	case rolemodel.AdminRoleStatusActive, rolemodel.AdminRoleStatusDisabled:
		return true
	default:
		return false
	}
}

func normalizeBoolFlag(v uint8) uint8 {
	if v > 0 {
		return 1
	}
	return 0
}
