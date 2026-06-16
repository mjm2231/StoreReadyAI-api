package service

import (
	"context"
	"fmt"

	"storeready_ai/internal/admin/modules/me/dto"
	rbacsv "storeready_ai/internal/admin/modules/rbac/service"
)

type Service interface {
	GetCurrent(ctx context.Context, tenantID uint64, ID uint64) (*dto.MeResponse, error)
}

type service struct {
	rbac rbacsv.Service
}

func New(rbacsv rbacsv.Service) Service {
	return &service{
		rbac: rbacsv,
	}
}

func (s *service) GetCurrent(ctx context.Context, tenantID uint64, ID uint64) (*dto.MeResponse, error) {
	user, err := s.rbac.GetByID(ctx, tenantID, ID)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.rbac.GetSnapshot(ctx, tenantID, ID)
	if err != nil {
		return nil, err
	}

	roles := make([]dto.CurrentRole, 0, len(snapshot.Roles))
	for _, role := range snapshot.Roles {
		roles = append(roles, dto.CurrentRole{
			ID:   role.ID,
			Code: role.Code,
			Name: role.Name,
		})
	}

	resp := &dto.MeResponse{
		User: dto.CurrentUser{
			ID:       user.ID,
			Username: user.Username,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
			TenantID: user.TenantID,
			Status:   user.Status,
		},
		Roles:       roles,
		Permissions: snapshot.PermissionCodes,
	}
	fmt.Printf("GetCurrent MeResponse: %+v\n", resp)
	return resp, nil
}
