package agent

import (
	"context"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/model"
)

// FileTool 文件工具
type FileTool struct {
	sandbox external.Sandbox
}

// NewFileTool 创建文件工具
func NewFileTool(sandbox external.Sandbox) *FileTool {
	return &FileTool{sandbox: sandbox}
}

// Name 返回工具名称
func (t *FileTool) Name() string {
	return "file"
}

// Description 返回工具描述
func (t *FileTool) Description() string {
	return "用于读取、写入、搜索和管理文件。可以查看文件内容、创建新文件、修改现有文件、搜索文件内容等。"
}

// Parameters 返回工具参数定义
func (t *FileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: read, write, delete, exists, list, search, replace",
				"enum":        []string{"read", "write", "delete", "exists", "list", "search", "replace"},
			},
			"filepath": map[string]interface{}{
				"type":        "string",
				"description": "文件路径",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "写入文件的内容 (仅 write 操作)",
			},
			"append": map[string]interface{}{
				"type":        "boolean",
				"description": "是否追加模式写入",
			},
			"regex": map[string]interface{}{
				"type":        "string",
				"description": "搜索正则表达式 (仅 search 操作)",
			},
			"old_str": map[string]interface{}{
				"type":        "string",
				"description": "要替换的旧字符串 (仅 replace 操作)",
			},
			"new_str": map[string]interface{}{
				"type":        "string",
				"description": "新字符串 (仅 replace 操作)",
			},
			"dir_path": map[string]interface{}{
				"type":        "string",
				"description": "目录路径 (仅 list 操作)",
			},
			"glob_pattern": map[string]interface{}{
				"type":        "string",
				"description": "文件匹配模式 (仅 find 操作)",
			},
			"start_line": map[string]interface{}{
				"type":        "integer",
				"description": "起始行号 (仅 read 操作)",
			},
			"end_line": map[string]interface{}{
				"type":        "integer",
				"description": "结束行号 (仅 read 操作)",
			},
			"max_length": map[string]interface{}{
				"type":        "integer",
				"description": "最大读取长度",
			},
			"sudo": map[string]interface{}{
				"type":        "boolean",
				"description": "是否使用 sudo 权限",
			},
		},
		"required": []string{"action", "filepath"},
	}
}

// Invoke 调用工具
func (t *FileTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	action, _ := params["action"].(string)
	filepath, _ := params["filepath"].(string)

	switch action {
	case "read":
		var startLine, endLine *int
		if v, ok := params["start_line"].(float64); ok {
			n := int(v)
			startLine = &n
		}
		if v, ok := params["end_line"].(float64); ok {
			n := int(v)
			endLine = &n
		}
		maxLength := 10000
		if v, ok := params["max_length"].(float64); ok {
			maxLength = int(v)
		}
		sudo := false
		if v, ok := params["sudo"].(bool); ok {
			sudo = v
		}
		return t.sandbox.ReadFile(ctx, filepath, startLine, endLine, sudo, maxLength)

	case "write":
		content, _ := params["content"].(string)
		append := false
		if v, ok := params["append"].(bool); ok {
			append = v
		}
		sudo := false
		if v, ok := params["sudo"].(bool); ok {
			sudo = v
		}
		return t.sandbox.WriteFile(ctx, filepath, content, append, false, false, sudo)

	case "delete":
		return t.sandbox.DeleteFile(ctx, filepath)

	case "exists":
		return t.sandbox.CheckFileExists(ctx, filepath)

	case "list":
		dirPath := filepath
		if v, ok := params["dir_path"].(string); ok {
			dirPath = v
		}
		return t.sandbox.ListFiles(ctx, dirPath)

	case "search":
		regex, _ := params["regex"].(string)
		sudo := false
		if v, ok := params["sudo"].(bool); ok {
			sudo = v
		}
		return t.sandbox.SearchInFile(ctx, filepath, regex, sudo)

	case "replace":
		oldStr, _ := params["old_str"].(string)
		newStr, _ := params["new_str"].(string)
		sudo := false
		if v, ok := params["sudo"].(bool); ok {
			sudo = v
		}
		return t.sandbox.ReplaceInFile(ctx, filepath, oldStr, newStr, sudo)

	default:
		return model.NewToolError("unknown action: " + action), nil
	}
}
