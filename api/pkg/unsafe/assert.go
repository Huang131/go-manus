package unsafe

import "fmt"

// MustString 安全获取字符串值，如果类型不对则返回默认值
func MustString(v interface{}, defaultVal string) string {
	if s, ok := v.(string); ok {
		return s
	}
	return defaultVal
}

// GetString 安全获取字符串值，如果类型不对则返回错误
func GetString(v interface{}) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("expected string, got %T", v)
}

// MustMap 安全获取 map 值
func MustMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// GetMap 安全获取 map 值，类型不对返回错误
func GetMap(v interface{}) (map[string]interface{}, error) {
	if m, ok := v.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("expected map[string]interface{}, got %T", v)
}

// GetNestedString 安全获取嵌套字符串值
func GetNestedString(data map[string]interface{}, keys ...string) string {
	current := interface{}(data)
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return ""
		}
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

// GetNestedMap 安全获取嵌套 map
func GetNestedMap(data map[string]interface{}, keys ...string) map[string]interface{} {
	current := interface{}(data)
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return nil
		}
	}
	if m, ok := current.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// GetInt 安全获取 int 值
func GetInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	default:
		return 0, fmt.Errorf("expected int, got %T", v)
	}
}

// GetInt64 安全获取 int64 值
func GetInt64(v interface{}) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case float64:
		return int64(val), nil
	default:
		return 0, fmt.Errorf("expected int64, got %T", v)
	}
}

// GetBool 安全获取 bool 值
func GetBool(v interface{}) (bool, error) {
	if b, ok := v.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("expected bool, got %T", v)
}

// GetFloat64 安全获取 float64 值
func GetFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("expected float64, got %T", v)
	}
}
