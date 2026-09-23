package tools

import (
	"fmt"
	"strings"

	"github.com/Huang131/go-manus/api/internal/model"
)

// requiredToolString 读取工具必填字符串参数，统一返回可展示的工具错误。
func requiredToolString(params map[string]interface{}, key string) (string, *model.ToolResult) {
	value, ok := params[key]
	if !ok {
		return "", model.NewToolError(fmt.Sprintf("%s 不能为空", key))
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", model.NewToolError(fmt.Sprintf("%s 必须是非空字符串", key))
	}
	return text, nil
}

// optionalToolString 读取可选字符串参数，缺失或类型不符时返回空串。
func optionalToolString(params map[string]interface{}, key string) string {
	text, _ := params[key].(string)
	return text
}

// optionalToolBool 读取可选布尔参数，缺失或类型不符时返回 false。
func optionalToolBool(params map[string]interface{}, key string) bool {
	value, _ := params[key].(bool)
	return value
}

// optionalToolInt 读取可选整型参数，返回指针以便区分"未传"与"传了 0"。
func optionalToolInt(params map[string]interface{}, key string) *int {
	value, ok := params[key].(float64)
	if !ok {
		return nil
	}
	n := int(value)
	return &n
}

// optionalToolFloat 读取可选浮点参数，返回指针以便区分"未传"与"传了 0"。
func optionalToolFloat(params map[string]interface{}, key string) *float64 {
	value, ok := params[key].(float64)
	if !ok {
		return nil
	}
	return &value
}
