package agent

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
