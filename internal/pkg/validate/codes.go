package validate

// 校验错误码（建议前端/网关统一识别）。
// 说明：
//   - Code 是字段级错误码，不等同于业务错误码（pkg/errors.Code）。
//   - 这里的 code 主要用于表单校验/参数校验的细粒度定位。
const (
	CodeInvalid      = "invalid"
	CodeRequired     = "required"
	CodeMinLen       = "min_len"
	CodeMaxLen       = "max_len"
	CodeInvalidEmail = "invalid_email"
	CodeInvalidPhone = "invalid_phone"
)
