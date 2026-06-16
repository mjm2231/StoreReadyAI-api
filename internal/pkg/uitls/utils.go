package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ToString 将任意基础类型转换为 string。
func ToString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case []byte:
		return string(val)
	case fmt.Stringer:
		return val.String()
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToInt64 将常见基础类型转换为 int64。
func ToInt64(v any) (int64, error) {
	switch val := v.(type) {
	case nil:
		return 0, fmt.Errorf("value is nil")
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		if val > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("uint64 value out of int64 range: %d", val)
		}
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		return strconv.ParseInt(strings.TrimSpace(val), 10, 64)
	case []byte:
		return strconv.ParseInt(strings.TrimSpace(string(val)), 10, 64)
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type for ToInt64: %T", v)
	}
}

// ToInt 将常见基础类型转换为 int。
func ToInt(v any) (int, error) {
	n, err := ToInt64(v)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// MustToInt 将值转换为 int，失败时返回默认值。
func MustToInt(v any, defaultValue int) int {
	n, err := ToInt(v)
	if err != nil {
		return defaultValue
	}
	return n
}

// ToUint64 将常见基础类型转换为 uint64。
func ToUint64(v any) (uint64, error) {
	switch val := v.(type) {
	case nil:
		return 0, fmt.Errorf("value is nil")
	case uint:
		return uint64(val), nil
	case uint8:
		return uint64(val), nil
	case uint16:
		return uint64(val), nil
	case uint32:
		return uint64(val), nil
	case uint64:
		return val, nil
	case int:
		if val < 0 {
			return 0, fmt.Errorf("negative value cannot convert to uint64: %d", val)
		}
		return uint64(val), nil
	case int8:
		if val < 0 {
			return 0, fmt.Errorf("negative value cannot convert to uint64: %d", val)
		}
		return uint64(val), nil
	case int16:
		if val < 0 {
			return 0, fmt.Errorf("negative value cannot convert to uint64: %d", val)
		}
		return uint64(val), nil
	case int32:
		if val < 0 {
			return 0, fmt.Errorf("negative value cannot convert to uint64: %d", val)
		}
		return uint64(val), nil
	case int64:
		if val < 0 {
			return 0, fmt.Errorf("negative value cannot convert to uint64: %d", val)
		}
		return uint64(val), nil
	case float32:
		if val < 0 {
			return 0, fmt.Errorf("negative value cannot convert to uint64: %f", val)
		}
		return uint64(val), nil
	case float64:
		if val < 0 {
			return 0, fmt.Errorf("negative value cannot convert to uint64: %f", val)
		}
		return uint64(val), nil
	case string:
		return strconv.ParseUint(strings.TrimSpace(val), 10, 64)
	case []byte:
		return strconv.ParseUint(strings.TrimSpace(string(val)), 10, 64)
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type for ToUint64: %T", v)
	}
}

// ToFloat64 将常见基础类型转换为 float64。
func ToFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case nil:
		return 0, fmt.Errorf("value is nil")
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(val), 64)
	case []byte:
		return strconv.ParseFloat(strings.TrimSpace(string(val)), 64)
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type for ToFloat64: %T", v)
	}
}

// ToBool 将常见基础类型转换为 bool。
func ToBool(v any) (bool, error) {
	switch val := v.(type) {
	case nil:
		return false, fmt.Errorf("value is nil")
	case bool:
		return val, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(strings.ToLower(val)))
	case []byte:
		return strconv.ParseBool(strings.TrimSpace(strings.ToLower(string(val))))
	case int:
		return val != 0, nil
	case int8:
		return val != 0, nil
	case int16:
		return val != 0, nil
	case int32:
		return val != 0, nil
	case int64:
		return val != 0, nil
	case uint:
		return val != 0, nil
	case uint8:
		return val != 0, nil
	case uint16:
		return val != 0, nil
	case uint32:
		return val != 0, nil
	case uint64:
		return val != 0, nil
	case float32:
		return val != 0, nil
	case float64:
		return val != 0, nil
	default:
		return false, fmt.Errorf("unsupported type for ToBool: %T", v)
	}
}

// ToStringSlice 将 []any 转换为 []string。
func ToStringSlice(values []any) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, v := range values {
		result = append(result, ToString(v))
	}
	return result
}
