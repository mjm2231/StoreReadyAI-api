package validate

import (
	"regexp"
	"strings"
)

var (
	reEmail = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	// 简单手机号校验（国际化业务建议按国家码更严格）
	rePhone = regexp.MustCompile(`^\+?[0-9]{7,20}$`)
)

// Required 必填。
func Required(v string, msg string) Rule {
	return func() *FieldError {
		if strings.TrimSpace(v) == "" {
			return &FieldError{Code: CodeRequired, Msg: msg}
		}
		return nil
	}
}

// MinLen 最小长度。
func MinLen(v string, n int, msg string) Rule {
	return func() *FieldError {
		if len(v) < n {
			return &FieldError{Code: CodeMinLen, Msg: msg}
		}
		return nil
	}
}

// MaxLen 最大长度。
func MaxLen(v string, n int, msg string) Rule {
	return func() *FieldError {
		if len(v) > n {
			return &FieldError{Code: CodeMaxLen, Msg: msg}
		}
		return nil
	}
}

// IsEmail 邮箱格式。
func IsEmail(v string, msg string) Rule {
	return func() *FieldError {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if !reEmail.MatchString(v) {
			return &FieldError{Code: CodeInvalidEmail, Msg: msg}
		}
		return nil
	}
}

// IsPhone 手机号格式（简版）。
func IsPhone(v string, msg string) Rule {
	return func() *FieldError {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if !rePhone.MatchString(v) {
			return &FieldError{Code: CodeInvalidPhone, Msg: msg}
		}
		return nil
	}
}
