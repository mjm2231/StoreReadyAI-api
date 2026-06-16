package response

import "time"

type Response[T any] struct {
	ServerNow uint64         `json:"server_now,omitempty"`
	Code      int32          `json:"code"`
	Msg       string         `json:"msg"`
	MsgKey    string         `json:"msg_key,omitempty"`
	MsgParams map[string]any `json:"msg_params,omitempty"`
	Data      T              `json:"data,omitempty"`
	RID       string         `json:"rid,omitempty"`
}

const CodeOK int32 = 0

func OK[T any](data T, rid string) Response[T] {
	return Response[T]{Code: CodeOK, Msg: "OK", Data: data, RID: rid, ServerNow: uint64(time.Now().Unix())}
}

func Fail(code int32, msg, rid string) Response[any] {
	return Response[any]{Code: code, Msg: msg, RID: rid, ServerNow: uint64(time.Now().Unix())}
}

func FailWithKey(code int32, msg, msgKey string, msgParams map[string]any, rid string) Response[any] {
	return Response[any]{
		Code:      code,
		Msg:       msg,
		MsgKey:    msgKey,
		MsgParams: msgParams,
		RID:       rid,
		ServerNow: uint64(time.Now().Unix()),
	}
}

func FailWithData[T any](code int32, msg string, data T, rid string) Response[T] {
	return Response[T]{Code: code, Msg: msg, Data: data, RID: rid, ServerNow: uint64(time.Now().Unix())}
}

func FailWithKeyAndData[T any](code int32, msg, msgKey string, msgParams map[string]any, data T, rid string) Response[T] {
	return Response[T]{
		Code:      code,
		Msg:       msg,
		MsgKey:    msgKey,
		MsgParams: msgParams,
		Data:      data,
		RID:       rid,
		ServerNow: uint64(time.Now().Unix()),
	}
}

// ---- Swagger 文档专用（swag 不支持泛型/类型参数） ----
// 用法示例：
//   // @Success 200 {object} response.DocResponse{data=dto.FirebaseLoginResp} "成功"
//   // @Failure 400 {object} response.DocError "参数错误"

// DocResponse Swagger 文档专用：通用成功包裹（data 类型由注释里的 {data=xxx} 指定）。
// 注意：该结构体仅用于 swagger 文档生成，不影响运行时实际返回结构。
type DocResponse struct {
	Code      int32          `json:"code"`
	Msg       string         `json:"msg"`
	MsgKey    string         `json:"msg_key,omitempty"`
	MsgParams map[string]any `json:"msg_params,omitempty"`
	Data      interface{}    `json:"data,omitempty"`
	RID       string         `json:"rid,omitempty"`
	ServerNow uint64         `json:"server_now,omitempty"`
}

// DocError Swagger 文档专用：通用错误包裹（通常不返回 data）。
// 注意：该结构体仅用于 swagger 文档生成，不影响运行时实际返回结构。
type DocError struct {
	Code      int32          `json:"code"`
	MsgKey    string         `json:"msg_key,omitempty"`
	MsgParams map[string]any `json:"msg_params,omitempty"`
	Msg       string         `json:"msg"`
	RID       string         `json:"rid,omitempty"`
	ServerNow uint64         `json:"server_now,omitempty"`
}
