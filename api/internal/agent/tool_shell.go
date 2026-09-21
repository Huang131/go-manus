package agent

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
)

// ShellTool Shell 工具
type ShellTool struct {
	sandbox sandbox.Sandbox
}

// NewShellTool 创建 Shell 工具
func NewShellTool(sandbox sandbox.Sandbox) *ShellTool {
	return &ShellTool{sandbox: sandbox}
}

// Name 返回工具名称
func (t *ShellTool) Name() string {
	return ToolNameShell
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
				"enum":        []string{ShellActionExec, ShellActionRead, ShellActionWrite, ShellActionWait, ShellActionKill},
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
		// session_id 不暴露给模型：沙箱会话与本次对话一一对应，由 Agent 执行前注入，
		// 模型自行编造会在共享沙箱中串到别的会话。
		"required": []string{"action"},
	}
}

// ReadOnly shell 可执行任意命令，保守视为有副作用。
func (t *ShellTool) ReadOnly() bool {
	return false
}

// Sandbox 返回底层沙箱客户端（供 BaseAgent 的 shell 输出 watcher 轮询）。
func (t *ShellTool) Sandbox() sandbox.Sandbox {
	return t.sandbox
}

// Invoke 调用工具
func (t *ShellTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	action, toolErr := requiredToolString(params, "action")
	if toolErr != nil {
		return toolErr, nil
	}
	// session_id 由 Agent 注入，缺失说明调用链未按约定注入，属于内部错误
	sessionID, toolErr := requiredToolString(params, "session_id")
	if toolErr != nil {
		return toolErr, nil
	}

	switch action {
	case ShellActionExec:
		command, toolErr := requiredToolString(params, "command")
		if toolErr != nil {
			return toolErr, nil
		}
		return t.sandbox.ExecCommand(ctx, sessionID, optionalToolString(params, "exec_dir"), command)

	case ShellActionRead:
		return t.sandbox.ReadShellOutput(ctx, sessionID, optionalToolBool(params, "console"))

	case ShellActionWrite:
		// press_enter 缺省为 true：写输入通常是回答交互式提问，回车才是语义完整的一步
		pressEnter := true
		if _, ok := params["press_enter"]; ok {
			pressEnter = optionalToolBool(params, "press_enter")
		}
		return t.sandbox.WriteShellInput(ctx, sessionID, optionalToolString(params, "input_text"), pressEnter)

	case ShellActionWait:
		return t.sandbox.WaitProcess(ctx, sessionID, optionalToolInt(params, "seconds"))

	case ShellActionKill:
		return t.sandbox.KillProcess(ctx, sessionID)

	default:
		return model.NewToolError("unknown action: " + action), nil
	}
}
