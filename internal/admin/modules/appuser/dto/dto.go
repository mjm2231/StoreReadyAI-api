package dto

import "storeready_ai/internal/contracts/user"

type ListUsersResp struct {
	Users []*user.UserVO `json:"list"`
	Total uint64         `json:"total"`
}

type GetUserByIDReq struct {
	ID string `json:"id" binding:"required"`
}

type GetUserByIDResp struct {
	User *user.UserVO `json:"user"`
}

type UpdateUserReq struct {
	ID uint64 `json:"id" binding:"required"`
	user.UpdateUserReq
}
