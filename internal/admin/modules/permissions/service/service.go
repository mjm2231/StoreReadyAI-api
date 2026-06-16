package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	permissiondto "storeready_ai/internal/admin/modules/permissions/dto"
	permissionmodel "storeready_ai/internal/admin/modules/permissions/model"
	permissionrepo "storeready_ai/internal/admin/modules/permissions/repo"
)

var (
	ErrNilRepo                  = errors.New("admin permission service: repo is nil")
	ErrInvalidID                = errors.New("admin permission service: invalid id")
	ErrInvalidName              = errors.New("admin permission service: invalid name")
	ErrInvalidCode              = errors.New("admin permission service: invalid code")
	ErrInvalidType              = errors.New("admin permission service: invalid type")
	ErrInvalidStatus            = errors.New("admin permission service: invalid status")
	ErrPermissionCodeExists     = errors.New("admin permission service: permission code already exists")
	ErrPermissionNotFound       = errors.New("admin permission service: permission not found")
	ErrSystemPermissionReadonly = errors.New("admin permission service: system permission is readonly")
)

// Service 是后台权限服务接口。
//
// 约定：
// 1. admin 模块统一采用“接口 + 实现”结构；
// 2. handler 依赖接口而不是具体实现，便于测试与后续扩展；
// 3. 当前聚焦后台权限管理最小必要能力：列表、详情、创建、更新、状态更新、删除。
type Service interface {
	List(ctx context.Context, req permissiondto.PermissionListRequest) (permissiondto.PermissionListResponse, error)
	GetDetail(ctx context.Context, req permissiondto.GetPermissionDetailRequest) (permissiondto.PermissionDetail, error)
	Create(ctx context.Context, req permissiondto.CreatePermissionRequest) (permissiondto.PermissionDetail, error)
	Update(ctx context.Context, req permissiondto.UpdatePermissionRequest) (permissiondto.PermissionDetail, error)
	UpdateStatus(ctx context.Context, req permissiondto.UpdatePermissionStatusRequest) error
	Delete(ctx context.Context, req permissiondto.DeletePermissionRequest) error
	ListPermissionCodesByIds(ctx context.Context, tenantID uint64, perIds []uint64) ([]string, error)
}

// service 是 Service 的默认实现。
type service struct {
	repo permissionrepo.Repository
	now  func() time.Time
}

func New(repo permissionrepo.Repository) (Service, error) {
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

func (s *service) List(ctx context.Context, req permissiondto.PermissionListRequest) (permissiondto.PermissionListResponse, error) {
	if s == nil {
		return permissiondto.PermissionListResponse{}, ErrNilRepo
	}
	req = req.Normalize()

	filter := permissionrepo.ListFilter{
		Keyword: req.Keyword,
		Module:  req.Module,
		Offset:  req.Offset,
		Limit:   req.Limit,
	}
	if req.Type != nil {
		t := *req.Type
		filter.Type = &t
	}
	if req.Status != nil {
		status := *req.Status
		filter.Status = &status
	}
	if req.ParentID != nil {
		parentID := *req.ParentID
		filter.ParentID = &parentID
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
		return permissiondto.PermissionListResponse{}, err
	}
	items, err := s.repo.List(ctx, req.TenantID, filter)
	if err != nil {
		return permissiondto.PermissionListResponse{}, err
	}
	return permissiondto.PermissionListResponse{
		Total: total,
		Items: permissiondto.ToPermissionItems(items),
	}, nil
}

func (s *service) GetDetail(ctx context.Context, req permissiondto.GetPermissionDetailRequest) (permissiondto.PermissionDetail, error) {
	if s == nil {
		return permissiondto.PermissionDetail{}, ErrNilRepo
	}
	if req.ID == 0 {
		return permissiondto.PermissionDetail{}, ErrInvalidID
	}
	permission, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return permissiondto.PermissionDetail{}, ErrPermissionNotFound
		}
		return permissiondto.PermissionDetail{}, err
	}
	return permissiondto.ToPermissionDetail(permission), nil
}

func (s *service) Create(ctx context.Context, req permissiondto.CreatePermissionRequest) (permissiondto.PermissionDetail, error) {
	if s == nil {
		return permissiondto.PermissionDetail{}, ErrNilRepo
	}
	req = req.Normalize()
	if err := validateCreateRequest(req); err != nil {
		return permissiondto.PermissionDetail{}, err
	}

	exists, err := s.repo.ExistsByCode(ctx, req.TenantID, req.Code, 0)
	if err != nil {
		return permissiondto.PermissionDetail{}, err
	}
	if exists {
		return permissiondto.PermissionDetail{}, ErrPermissionCodeExists
	}

	nowUnix := s.nowUnix()
	permission := &permissionmodel.AdminPermission{
		TenantID:  req.TenantID,
		Name:      req.Name,
		Code:      req.Code,
		Module:    req.Module,
		Type:      req.Type,
		ParentID:  req.ParentID,
		Path:      req.Path,
		Icon:      req.Icon,
		Sort:      req.Sort,
		Status:    req.Status,
		IsSystem:  normalizeBoolFlag(req.IsSystem),
		Remark:    req.Remark,
		CreatedAt: nowUnix,
		UpdatedAt: nowUnix,
	}
	if err := s.repo.Create(ctx, permission); err != nil {
		return permissiondto.PermissionDetail{}, err
	}
	return permissiondto.ToPermissionDetail(permission), nil
}

func (s *service) Update(ctx context.Context, req permissiondto.UpdatePermissionRequest) (permissiondto.PermissionDetail, error) {
	if s == nil {
		return permissiondto.PermissionDetail{}, ErrNilRepo
	}
	req = req.Normalize()
	if err := validateUpdateRequest(req); err != nil {
		return permissiondto.PermissionDetail{}, err
	}

	permission, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return permissiondto.PermissionDetail{}, ErrPermissionNotFound
		}
		return permissiondto.PermissionDetail{}, err
	}
	if permission.IsSystemBuiltin() {
		return permissiondto.PermissionDetail{}, ErrSystemPermissionReadonly
	}

	if req.Code != permission.Code {
		exists, err := s.repo.ExistsByCode(ctx, req.TenantID, req.Code, req.ID)
		if err != nil {
			return permissiondto.PermissionDetail{}, err
		}
		if exists {
			return permissiondto.PermissionDetail{}, ErrPermissionCodeExists
		}
	}

	permission.Name = req.Name
	permission.Code = req.Code
	permission.Module = req.Module
	permission.Type = req.Type
	permission.ParentID = req.ParentID
	permission.Path = req.Path
	permission.Icon = req.Icon
	permission.Sort = req.Sort
	permission.Status = req.Status
	permission.IsSystem = normalizeBoolFlag(req.IsSystem)
	permission.Remark = req.Remark
	permission.UpdatedAt = s.nowUnix()

	if err := s.repo.Update(ctx, permission); err != nil {
		return permissiondto.PermissionDetail{}, err
	}
	return permissiondto.ToPermissionDetail(permission), nil
}

func (s *service) UpdateStatus(ctx context.Context, req permissiondto.UpdatePermissionStatusRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	if req.ID == 0 {
		return ErrInvalidID
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}

	permission, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPermissionNotFound
		}
		return err
	}
	if permission.IsSystemBuiltin() {
		return ErrSystemPermissionReadonly
	}
	return s.repo.UpdateStatus(ctx, req.TenantID, req.ID, req.Status, s.nowUnix())
}

func (s *service) Delete(ctx context.Context, req permissiondto.DeletePermissionRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	if req.ID == 0 {
		return ErrInvalidID
	}

	permission, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPermissionNotFound
		}
		return err
	}
	if permission.IsSystemBuiltin() {
		return ErrSystemPermissionReadonly
	}
	nowUnix := s.nowUnix()
	return s.repo.SoftDelete(ctx, req.TenantID, req.ID, nowUnix, nowUnix)
}

func (s *service) ListPermissionCodesByIds(ctx context.Context, tenantID uint64, perIds []uint64) ([]string, error) {
	if s == nil {
		return nil, ErrNilRepo
	}
	if tenantID == 0 {
		return nil, errors.New("tenant_id is required")
	}
	if len(perIds) == 0 {
		return []string{}, nil
	}

	permissions, err := s.repo.GetByIDs(ctx, tenantID, perIds)
	if err != nil {
		return nil, err
	}

	codes := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		codes = append(codes, permission.Code)
	}
	return codes, nil
}

// func (s *service) ListPermissionCodesByUser(ctx context.Context, tenantID uint64, userID uint64) ([]string, error) {
// 	if s == nil {
// 		return nil, ErrNilRepo
// 	}
// 	if tenantID == 0 {
// 		return nil, errors.New("tenant_id is required")
// 	}
// 	if userID == 0 {
// 		return nil, errors.New("user_id is required")
// 	}
// 	roleIDs, err := s.userRole.ListRoleIDsByAdminUserID(ctx, tenantID, userID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(roleIDs) == 0 {
// 		return []string{}, nil
// 	}
// 	permissionIDs, err := s.rolePermission.ListPermissionIDsByRoleIDs(ctx, tenantID, roleIDs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if len(permissionIDs) == 0 {
// 		return []string{}, nil
// 	}
// 	permissionmodels, err := s.repo.GetByRoles(ctx, tenantID, permissionIDs)
// 	if err != nil {
// 		return nil, err
// 	}

// 	codes := make([]string, 0, len(permissionmodels))
// 	for _, model := range permissionmodels {
// 		codes = append(codes, model.Code)
// 	}
// 	return codes, nil
// }

func (s *service) nowUnix() uint64 {
	if s == nil || s.now == nil {
		return uint64(time.Now().Unix())
	}
	return uint64(s.now().Unix())
}

func validateCreateRequest(req permissiondto.CreatePermissionRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrInvalidName
	}
	if strings.TrimSpace(req.Code) == "" {
		return ErrInvalidCode
	}
	if !isValidType(req.Type) {
		return ErrInvalidType
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}
	return nil
}

func validateUpdateRequest(req permissiondto.UpdatePermissionRequest) error {
	if req.ID == 0 {
		return ErrInvalidID
	}
	if strings.TrimSpace(req.Name) == "" {
		return ErrInvalidName
	}
	if strings.TrimSpace(req.Code) == "" {
		return ErrInvalidCode
	}
	if !isValidType(req.Type) {
		return ErrInvalidType
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}
	return nil
}

func isValidType(v uint8) bool {
	switch v {
	case permissionmodel.AdminPermissionTypeMenu, permissionmodel.AdminPermissionTypePage, permissionmodel.AdminPermissionTypeAction:
		return true
	default:
		return false
	}
}

func isValidStatus(v uint8) bool {
	switch v {
	case permissionmodel.AdminPermissionStatusActive, permissionmodel.AdminPermissionStatusDisabled:
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
