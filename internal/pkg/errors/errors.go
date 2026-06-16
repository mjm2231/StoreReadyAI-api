package errors

import (
	stderrors "errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// AppError 企业级业务错误。
//
// 增强点：
//   - MsgKey：给客户端做本地化（前端拿 key 去翻译）
//   - Details：结构化详情（可序列化，适合上报/BI/调试展示）
//
// 注意：
//   - Msg 是兜底提示（可为空）；对外不要直接输出 Cause/Stack
//   - Details 不要放敏感数据（token/密码/身份证等）
type AppError struct {
	Code          Code           `json:"code"`
	Msg           string         `json:"msg"`
	MsgKeyText    string         `json:"msg_key,omitempty"`
	MsgParamsData map[string]any `json:"msg_params,omitempty"`
	Details       map[string]any `json:"details,omitempty"`

	Cause error  `json:"-"`
	Stack string `json:"-"`
}

func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
	msg := strings.TrimSpace(e.Msg)
	if msg == "" {
		msg = strconv.FormatInt(int64(e.Code), 10)
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// New 创建一个业务错误（无 cause）。
func New(code Code, msg string) *AppError {
	return &AppError{Code: code, Msg: msg, Stack: captureStack(3)}
}

// NewI18n 创建带 msg key 的业务错误。
func NewI18n(code Code, msgKey string, msg string) *AppError {
	return &AppError{Code: code, MsgKeyText: strings.TrimSpace(msgKey), Msg: msg, Stack: captureStack(3)}
}

// NewWithKey 创建带 msg key 和参数的业务错误。
func NewWithKey(code Code, msgKey string, msg string, msgParams map[string]any) *AppError {
	return &AppError{
		Code:          code,
		Msg:           msg,
		MsgKeyText:    strings.TrimSpace(msgKey),
		MsgParamsData: msgParams,
		Stack:         captureStack(3),
	}
}

// WithDetails 设置结构化详情（链式）。
func (e *AppError) WithDetails(details map[string]any) *AppError {
	if e == nil {
		return nil
	}
	// 允许 nil，表示不设置
	e.Details = details
	return e
}

// AddDetail 增量写入一个 detail（链式）。
func (e *AppError) AddDetail(k string, v any) *AppError {
	if e == nil {
		return nil
	}
	k = strings.TrimSpace(k)
	if k == "" {
		return e
	}
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	e.Details[k] = v
	return e
}

// WithMsgKey 设置 msg key（链式）。
func (e *AppError) WithMsgKey(msgKey string) *AppError {
	if e == nil {
		return nil
	}
	e.MsgKeyText = strings.TrimSpace(msgKey)
	return e
}

// WithMsgParams 设置 msg 参数（链式）。
func (e *AppError) WithMsgParams(msgParams map[string]any) *AppError {
	if e == nil {
		return nil
	}
	e.MsgParamsData = msgParams
	return e
}

// Wrap 将底层 err 包装为 AppError。
//
// 规则：
//   - 若 err 已是 *AppError：保留 Code/MsgKey/Details；msg 非空则覆盖 Msg
//   - 若 err 不是 *AppError：使用传入 code/msg
func Wrap(err error, code Code, msg string) *AppError {
	if err == nil {
		return nil
	}
	var ae *AppError
	if stderrors.As(err, &ae) && ae != nil {
		out := *ae
		if strings.TrimSpace(msg) != "" {
			out.Msg = msg
		}
		out.Cause = err
		if strings.TrimSpace(out.Stack) == "" {
			out.Stack = captureStack(3)
		}
		return &out
	}
	return &AppError{Code: code, Msg: msg, Cause: err, Stack: captureStack(3)}
}

// WithCause 将业务错误附带底层 cause。
func WithCause(code Code, msg string, cause error) *AppError {
	if cause == nil {
		return New(code, msg)
	}
	return &AppError{Code: code, Msg: msg, Cause: cause, Stack: captureStack(3)}
}

// CodeOf 提取错误码。
func CodeOf(err error) Code {
	if err == nil {
		return CodeOK
	}
	var ae *AppError
	if stderrors.As(err, &ae) && ae != nil && ae.Code != 0 {
		return ae.Code
	}
	return CodeInternal
}

// MsgKeyOf 提取 msg key（无则返回空）。
func MsgKeyOf(err error) string {
	var ae *AppError
	if stderrors.As(err, &ae) && ae != nil {
		return strings.TrimSpace(ae.MsgKeyText)
	}
	return ""
}

// MsgParamsOf 提取 msg 参数（无则返回 nil）。
func MsgParamsOf(err error) map[string]any {
	var ae *AppError
	if stderrors.As(err, &ae) && ae != nil {
		return ae.MsgParamsData
	}
	return nil
}

// DetailsOf 提取结构化详情（无则返回 nil）。
func DetailsOf(err error) map[string]any {
	var ae *AppError
	if stderrors.As(err, &ae) && ae != nil {
		return ae.Details
	}
	return nil
}

func (e *AppError) MsgKeyValue() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.MsgKeyText)
}

func (e *AppError) MsgParamsValue() map[string]any {
	if e == nil {
		return nil
	}
	return e.MsgParamsData
}

func (e *AppError) MsgKey() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.MsgKeyText)
}

func (e *AppError) MsgParams() map[string]any {
	if e == nil {
		return nil
	}
	return e.MsgParamsData
}

// MessageOf 提取对外提示（不暴露内部 cause）。
func MessageOf(err error, fallback string) string {
	if err == nil {
		return ""
	}
	var ae *AppError
	if stderrors.As(err, &ae) && ae != nil {
		if strings.TrimSpace(ae.Msg) != "" {
			return ae.Msg
		}
		if ae.Code != 0 {
			return strconv.FormatInt(int64(ae.Code), 10)
		}
	}
	msg := strings.TrimSpace(err.Error())
	if msg != "" {
		return msg
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "系统错误"
}

// StackOf 获取栈信息（仅用于日志）。
func StackOf(err error) string {
	var ae *AppError
	if stderrors.As(err, &ae) && ae != nil {
		return ae.Stack
	}
	return ""
}

// IsCode 判断 err 是否为指定业务码（支持 Wrapped）。
func IsCode(err error, code Code) bool {
	return CodeOf(err) == code
}

func captureStack(skip int) string {
	const maxFrames = 32
	const maxLen = 8192

	pcs := make([]uintptr, maxFrames)
	n := runtime.Callers(skip, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	var b strings.Builder
	for {
		f, more := frames.Next()
		if !strings.Contains(f.Function, "runtime.") {
			b.WriteString(f.Function)
			b.WriteString("\n\t")
			b.WriteString(f.File)
			b.WriteString(":")
			b.WriteString(fmt.Sprintf("%d", f.Line))
			b.WriteString("\n")
		}
		if !more {
			break
		}
	}
	out := b.String()
	if len(out) > maxLen {
		return out[:maxLen] + "...(truncated)"
	}
	return out
}
