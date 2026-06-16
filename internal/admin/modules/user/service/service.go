package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"storeready_ai/internal/admin/modules/user/dto"
	userdto "storeready_ai/internal/admin/modules/user/dto"
	usermodel "storeready_ai/internal/admin/modules/user/model"
	userrepo "storeready_ai/internal/admin/modules/user/repo"
	utils "storeready_ai/internal/pkg/uitls"
)

var (
	ErrNilRepo               = errors.New("admin user service: repo is nil")
	ErrNilPasswordHasher     = errors.New("admin user service: password hasher is nil")
	ErrInvalidID             = errors.New("admin user service: invalid id")
	ErrInvalidUsername       = errors.New("admin user service: invalid username")
	ErrInvalidPassword       = errors.New("admin user service: invalid password")
	ErrPasswordNotMatch      = errors.New("admin user service: password and confirm password do not match")
	ErrUsernameAlreadyExists = errors.New("admin user service: username already exists")
	ErrEmailAlreadyExists    = errors.New("admin user service: email already exists")
	ErrInvalidStatus         = errors.New("admin user service: invalid status")
	ErrAdminUserNotFound     = errors.New("admin user service: admin user not found")
)

// PasswordHasher 负责管理员密码哈希。
//
// 说明：
// 1. service 不直接依赖具体加密算法；
// 2. 具体实现可在更上层接 bcrypt/argon2；
// 3. 当前只要求创建/改密能产出 hash，校验密码可后续补 Compare。
type PasswordHasher interface {
	HashPassword(password string, cost int) (string, error)
}

// Service 是后台管理员用户服务。
//
// 当前职责：
// 1. 封装 admin_users 的基础 CRUD；
// 2. 处理 DTO 标准化、唯一性校验、状态校验、密码哈希；
// 3. 输出 DTO，避免 handler 直接操作 model。
type Service struct {
	repo   userrepo.Repository
	hasher PasswordHasher
	now    func() time.Time
}

func New(repo userrepo.Repository, hasher PasswordHasher) (*Service, error) {
	if repo == nil || repo.DB() == nil {
		return nil, ErrNilRepo
	}

	if hasher == nil {
		return nil, ErrNilPasswordHasher
	}
	return &Service{
		repo:   repo,
		hasher: hasher,
		now:    time.Now,
	}, nil
}

func (s *Service) SetNow(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = now
}

func (s *Service) List(ctx context.Context, req userdto.AdminUserListRequest) (userdto.AdminUserListResponse, error) {
	if s == nil {
		return userdto.AdminUserListResponse{}, ErrNilRepo
	}
	req = req.Normalize()
	offset := (req.Page.Page - 1) * req.Page.PageSize

	filter := userrepo.ListFilter{
		Keyword: req.Keyword,
		Offset:  offset,
		Limit:   req.Page.PageSize,
	}
	if req.Status != 0 {
		status := req.Status
		filter.Status = &status
	}
	if req.IsSuperAdmin != nil {
		v := normalizeFlag(*req.IsSuperAdmin)
		filter.IsSuperAdmin = &v
	}
	tenantId, err := utils.ToUint64(req.TenantID)
	if err != nil {
		return userdto.AdminUserListResponse{}, err
	}
	total, err := s.repo.Count(ctx, tenantId, filter)
	if err != nil {
		return userdto.AdminUserListResponse{}, err
	}
	items, err := s.repo.List(ctx, tenantId, filter)
	if err != nil {
		return userdto.AdminUserListResponse{}, err
	}
	return userdto.AdminUserListResponse{
		Total: total,
		Items: userdto.ToAdminUserItems(items),
	}, nil
}

func (s *Service) Create(ctx context.Context, req userdto.CreateAdminUserRequest) (userdto.AdminUserDetail, error) {
	if s == nil {
		return userdto.AdminUserDetail{}, ErrNilRepo
	}
	req = req.Normalize()
	if err := validateCreateRequest(req); err != nil {
		return userdto.AdminUserDetail{}, err
	}

	exists, err := s.repo.ExistsByUsername(ctx, req.TenantID, req.Username, 0)
	if err != nil {
		return userdto.AdminUserDetail{}, err
	}
	if exists {
		return userdto.AdminUserDetail{}, ErrUsernameAlreadyExists
	}
	if req.Email != "" {
		exists, err = s.repo.ExistsByEmail(ctx, req.TenantID, req.Email, 0)
		if err != nil {
			return userdto.AdminUserDetail{}, err
		}
		if exists {
			return userdto.AdminUserDetail{}, ErrEmailAlreadyExists
		}
	}

	passwordHash, err := s.hasher.HashPassword(req.Password, 12)
	if err != nil {
		return userdto.AdminUserDetail{}, err
	}

	nowUnix := s.nowUnix()
	m := &usermodel.AdminUser{
		TenantID:     req.TenantID,
		Username:     req.Username,
		PasswordHash: strings.TrimSpace(passwordHash),
		Nickname:     req.Nickname,
		Email:        req.Email,
		Mobile:       req.Mobile,
		Avatar:       req.Avatar,
		Status:       req.Status,
		IsSuperAdmin: normalizeFlag(req.IsSuperAdmin),
		Remark:       req.Remark,
		CreatedAt:    nowUnix,
		UpdatedAt:    nowUnix,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return userdto.AdminUserDetail{}, err
	}
	return userdto.ToAdminUserDetail(m), nil
}

func (s *Service) Update(ctx context.Context, req userdto.UpdateAdminUserRequest) (userdto.AdminUserDetail, error) {
	if s == nil {
		return userdto.AdminUserDetail{}, ErrNilRepo
	}
	req = req.Normalize()
	if err := validateUpdateRequest(req); err != nil {
		return userdto.AdminUserDetail{}, err
	}

	m, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userdto.AdminUserDetail{}, ErrAdminUserNotFound
		}
		return userdto.AdminUserDetail{}, err
	}

	if req.Email != "" && req.Email != m.Email {
		exists, err := s.repo.ExistsByEmail(ctx, req.TenantID, req.Email, req.ID)
		if err != nil {
			return userdto.AdminUserDetail{}, err
		}
		if exists {
			return userdto.AdminUserDetail{}, ErrEmailAlreadyExists
		}
	}

	m.Nickname = req.Nickname
	m.Email = req.Email
	m.Mobile = req.Mobile
	m.Avatar = req.Avatar
	if req.Status != 0 {
		m.Status = req.Status
	}
	m.IsSuperAdmin = normalizeFlag(req.IsSuperAdmin)
	m.Remark = req.Remark
	m.UpdatedAt = s.nowUnix()

	if err := s.repo.Update(ctx, m); err != nil {
		return userdto.AdminUserDetail{}, err
	}
	return userdto.ToAdminUserDetail(m), nil
}

func (s *Service) UpdatePassword(ctx context.Context, req userdto.UpdateAdminUserPasswordRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	req = req.Normalize()
	if err := validateUpdatePasswordRequest(req); err != nil {
		return err
	}
	_, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}
	passwordHash, err := s.hasher.HashPassword(req.Password, 12)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, req.TenantID, req.ID, passwordHash, s.nowUnix())
}

func (s *Service) UpdateStatus(ctx context.Context, req userdto.UpdateAdminUserStatusRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	if req.ID == 0 {
		return ErrInvalidID
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}
	_, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}
	return s.repo.UpdateStatus(ctx, req.TenantID, req.ID, req.Status, s.nowUnix())
}

func (s *Service) Delete(ctx context.Context, req userdto.DeleteAdminUserRequest) error {
	if s == nil {
		return ErrNilRepo
	}
	if req.ID == 0 {
		return ErrInvalidID
	}
	_, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}
	nowUnix := s.nowUnix()
	return s.repo.SoftDelete(ctx, req.TenantID, req.ID, nowUnix, nowUnix)
}

func (s *Service) GetDetail(ctx context.Context, req userdto.GetAdminUserDetailRequest) (userdto.AdminUserDetail, error) {
	if s == nil {
		return userdto.AdminUserDetail{}, ErrNilRepo
	}
	if req.ID == 0 {
		return userdto.AdminUserDetail{}, ErrInvalidID
	}
	m, err := s.repo.GetByID(ctx, req.TenantID, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return userdto.AdminUserDetail{}, ErrAdminUserNotFound
		}
		return userdto.AdminUserDetail{}, err
	}
	return userdto.ToAdminUserDetail(m), nil
}

func (s *Service) UpdateLoginInfo(ctx context.Context, tenantID uint64, adminUserID uint64, loginIP string) error {
	if s == nil {
		return ErrNilRepo
	}
	if adminUserID == 0 {
		return ErrInvalidID
	}
	if tenantID == 0 {
		return ErrInvalidID
	}
	_, err := s.repo.GetByID(ctx, tenantID, adminUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminUserNotFound
		}
		return err
	}
	return s.repo.UpdateLoginInfo(ctx, tenantID, adminUserID, s.nowUnix(), strings.TrimSpace(loginIP), s.nowUnix())
}

func (s *Service) GetByID(ctx context.Context, tenantID uint64, id uint64) (*dto.AdminUserItem, error) {
	if s == nil {
		return nil, ErrNilRepo
	}
	if tenantID == 0 {
		return nil, ErrInvalidID
	}
	if id == 0 {
		return nil, ErrInvalidID
	}
	m, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminUserNotFound
		}
		return nil, err
	}
	return &dto.AdminUserItem{
		TenantID:     m.TenantID,
		ID:           m.ID,
		Username:     m.Username,
		Nickname:     m.Nickname,
		Email:        m.Email,
		Mobile:       m.Mobile,
		Avatar:       m.Avatar,
		Status:       m.Status,
		IsSuperAdmin: m.IsSuperAdmin,
		LastLoginAt:  m.LastLoginAt,
		LastLoginIP:  m.LastLoginIP,
		Remark:       m.Remark,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}, nil
}

func (s *Service) nowUnix() uint64 {
	if s == nil || s.now == nil {
		return uint64(time.Now().Unix())
	}
	return uint64(s.now().Unix())
}

func validateCreateRequest(req userdto.CreateAdminUserRequest) error {
	if strings.TrimSpace(req.Username) == "" {
		return ErrInvalidUsername
	}
	if strings.TrimSpace(req.Password) == "" {
		return ErrInvalidPassword
	}
	if !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}
	return nil
}

func validateUpdateRequest(req userdto.UpdateAdminUserRequest) error {
	if req.ID == 0 {
		return ErrInvalidID
	}
	if req.Status != 0 && !isValidStatus(req.Status) {
		return ErrInvalidStatus
	}
	return nil
}

func validateUpdatePasswordRequest(req userdto.UpdateAdminUserPasswordRequest) error {
	if req.ID == 0 {
		return ErrInvalidID
	}
	if strings.TrimSpace(req.Password) == "" {
		return ErrInvalidPassword
	}
	if req.Password != req.ConfirmPassword {
		return ErrPasswordNotMatch
	}
	return nil
}

func isValidStatus(status uint8) bool {
	switch status {
	case usermodel.AdminUserStatusActive,
		usermodel.AdminUserStatusDisabled,
		usermodel.AdminUserStatusDeleted:
		return true
	default:
		return false
	}
}

func normalizeFlag(v uint8) uint8 {
	if v > 0 {
		return 1
	}
	return 0
}
