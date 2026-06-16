package userx

import (
	"context"
	"fmt"
	usermodel "storeready_ai/internal/client/modules/user/model"
	"storeready_ai/internal/pkg/errors"
)

// UserResolver 用于把 JWT / 上下文中的业务 uid 解析为 users 表内部主键。
type UserResolver interface {
	GetUserByUID(ctx context.Context, tenantID, uid uint64) (*usermodel.User, error)
}

func ResolveUserID(ctx context.Context, r UserResolver, tenantID, uid uint64) (uint64, error) {
	if r == nil {
		return 0, fmt.Errorf("user resolver is nil")
	}
	if uid == 0 {
		return 0, errors.New(errors.CodeInvalidParam, "uid 非法")
	}

	u, err := r.GetUserByUID(ctx, tenantID, uid)
	if err != nil {
		return 0, fmt.Errorf("get user by uid failed: %w", err)
	}
	if u == nil || u.ID == 0 {
		return 0, errors.New(errors.CodeUserNotFound, "用户不存在")
	}
	return u.ID, nil
}
