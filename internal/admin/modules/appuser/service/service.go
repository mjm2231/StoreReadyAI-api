package service

import (
	"context"
	"storeready_ai/internal/contracts/user"
)

type Service interface {
	// ListUsers 列表查询用户（分页、过滤）。
	ListUsers(ctx context.Context, tenantID uint64, filter user.QueryUserFilter) ([]*user.UserVO, uint64, error)
	// UpdateUser 更新用户信息（可选：后续用到再加）。
	UpdateUser(ctx context.Context, tenantID, userID uint64, req user.UpdateUserReq) error
	// GetUserByID returns a user by externally exposed id.
	GetUserByID(ctx context.Context, tenantID, id uint64) (*user.UserVO, error)
}

type service struct {
	usersv Service
}

func New(usersv Service) *service {
	return &service{usersv: usersv}
}

func (s *service) ListUsers(ctx context.Context, tenantID uint64, filter user.QueryUserFilter) ([]*user.UserVO, uint64, error) {
	return s.usersv.ListUsers(ctx, tenantID, filter)
}

func (s *service) UpdateUser(ctx context.Context, tenantID, userID uint64, req user.UpdateUserReq) error {
	return s.usersv.UpdateUser(ctx, tenantID, userID, req)
}

func (s *service) GetUserByID(ctx context.Context, tenantID, id uint64) (*user.UserVO, error) {
	return s.usersv.GetUserByID(ctx, tenantID, id)
}
