package agent

import (
	"context"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/model"
)

// ShellTool Shell 工具
type ShellTool struct {
	sandbox external.Sandbox
}

// NewShellTool 创建 Shell 工具
func NewShellTool(sandbox external.Sandbox) *ShellTool {
	return &ShellTool{sandbox: sandbox}
}

// Name 返回工具名称
func (t *ShellTool) Name() string {
	return "shell"
}

// Description 返回工具描述
func (t *ShellTool) Description() string {
	return "用于执行 Shell 命令。可以在终端中运行命令、读取输出、写入输入、等待进程执行、杀死进程等。"
}

// Parameters 返回工具参数定义
func (t *ShellTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: exec, read, write, wait, kill",
				"enum":        []string{"exec", "read", "write", "wait", "kill"},
			},
			"session_id": map[string]interface{}{
				"type":        "string",
				"description": "Shell 会话 ID",
			},
			"exec_dir": map[string]interface{}{
				"type":        "string",
				"description": "执行目录",
			},
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的命令 (仅 exec 操作)",
			},
			"console": map[string]interface{}{
				"type":        "boolean",
				"description": "是否返回控制台记录 (仅 read 操作)",
			},
			"input_text": map[string]interface{}{
				"type":        "string",
				"description": "输入文本 (仅 write 操作)",
			},
			"press_enter": map[string]interface{}{
				"type":        "boolean",
				"description": "是否按回车键 (仅 write 操作)",
			},
			"seconds": map[string]interface{}{
				"type":        "integer",
				"description": "等待秒数 (仅 wait 操作)",
			},
		},
		"required": []string{"action", "session_id"},
	}
}

// Invoke 调用工具
func (t *ShellTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	action, _ := params["action"].(string)
	sessionID, _ := params["session_id"].(string)

	switch action {
	case "exec":
		execDir := ""
		if v, ok := params["exec_dir"].(string); ok {
			execDir = v
		}
		command := ""
		if v, ok := params["command"].(string); ok {
			command = v
		}
		return t.sandbox.ExecCommand(ctx, sessionID, execDir, command)

	case "read":
		console := false
		if v, ok := params["console"].(bool); ok {
			console = v
		}
		return t.sandbox.ReadShellOutput(ctx, sessionID, console)

	case "write":
		inputText := ""
		if v, ok := params["input_text"].(string); ok {
			inputText = v
		}
		pressEnter := true
		if v, ok := params["press_enter"].(bool); ok {
			pressEnter = v
		}
		return t.sandbox.WriteShellInput(ctx, sessionID, inputText, pressEnter)

	case "wait":
		var seconds *int
		if v, ok := params["seconds"].(float64); ok {
			n := int(v)
			seconds = &n
		}
		return t.sandbox.WaitProcess(ctx, sessionID, seconds)

	case "kill":
		return t.sandbox.KillProcess(ctx, sessionID)

	default:
		return model.NewToolError("unknown action: " + action), nil
	}
}
