package validate

import (
	"fmt"
	"strings"
)

// FieldError 字段级错误（企业级建议：前端可直接定位字段 + 错误码）。
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"` // 例如：required / min_len / invalid_email
	Msg   string `json:"msg"`
}

func (e FieldError) Error() string {
	f := strings.TrimSpace(e.Field)
	c := strings.TrimSpace(e.Code)
	m := strings.TrimSpace(e.Msg)
	if m == "" {
		m = "参数错误"
	}
	if f == "" && c == "" {
		return m
	}
	if f == "" {
		return fmt.Sprintf("%s: %s", c, m)
	}
	if c == "" {
		return fmt.Sprintf("%s: %s", f, m)
	}
	return fmt.Sprintf("%s.%s: %s", f, c, m)
}

// Errors 多字段错误集合。
type Errors []FieldError

func (es Errors) Error() string {
	if len(es) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range es {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Error())
	}
	out := b.String()
	if len(out) > 1024 {
		return out[:1024] + "...(truncated)"
	}
	return out
}

// Rule 规则函数：返回 nil 表示通过；返回 *FieldError 表示失败。
// 只负责“单条规则”的判断，不关心字段名（字段名在 Validate 中填）。
type Rule func() *FieldError

// Field 为某个字段收集规则。
type Field struct {
	Name  string
	Rules []Rule
}

// F 创建字段规则集合。
func F(name string, rules ...Rule) Field {
	return Field{Name: strings.TrimSpace(name), Rules: rules}
}

// Validate 执行校验，返回字段错误集合（无错返回 nil）。
//
// 用法：
//
//	err := validate.Validate(
//	    validate.F("email",
//	        validate.Required(email, "邮箱必填"),
//	        validate.IsEmail(email, "邮箱格式错误"),
//	    ),
//	    validate.F("password",
//	        validate.MinLen(password, 6, "密码至少6位"),
//	    ),
//	)
func Validate(fields ...Field) error {
	var out Errors
	for _, f := range fields {
		for _, rule := range f.Rules {
			if rule == nil {
				continue
			}
			fe := rule()
			if fe == nil {
				continue
			}
			// 统一填字段名
			fe.Field = f.Name
			// code 必须有（兜底）
			if strings.TrimSpace(fe.Code) == "" {
				fe.Code = CodeInvalid
			}
			// msg 兜底
			if strings.TrimSpace(fe.Msg) == "" {
				fe.Msg = "参数错误"
			}
			out = append(out, *fe)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// FirstErrorMsg 取第一条错误信息（适合只需要一个 toast 的场景）。
func FirstErrorMsg(err error) string {
	es, ok := err.(Errors)
	if ok && len(es) > 0 {
		return es[0].Msg
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

// FirstErrorCode 取第一条错误码（适合前端统一处理）。
func FirstErrorCode(err error) string {
	es, ok := err.(Errors)
	if ok && len(es) > 0 {
		return es[0].Code
	}
	return ""
}
