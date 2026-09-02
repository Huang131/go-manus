package agent

import (
	"context"
	"strings"

	"github.com/mooc-manus/go-manus/api/internal/llmcore"
	"github.com/mooc-manus/go-manus/api/internal/model"
)

// Tool 工具接口
type Tool interface {
	// Name 返回工具名称
	Name() string
	// Description 返回工具描述
	Description() string
	// Parameters 返回工具参数定义 (JSON Schema)
	Parameters() map[string]interface{}
	// ReadOnly 标记工具是否只读，供 fallback 规则判断
	ReadOnly() bool
	// Invoke 调用工具
	Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error)
}

// MultiFunctionTool 一个工具包可以向 LLM 暴露多个函数。
// MessageTool 使用该接口同时提供通知用户和询问用户两个函数。
//
// 阶段 1d 决策：保留 GetTools() []map 契约不变。
// 原因：MessageTool / tool_message 内部把每个 function 写死成 map[string]interface{}，
// 改 llmcore.ToolSpec 会引入"双 schema 表达"成本（map → ToolSpec → map 反向）。
// 阶段 2 可以整体重构成 []llmcore.ToolSpec。
type MultiFunctionTool interface {
	Tool
	GetTools() []map[string]interface{}
	InvokeWithName(functionName string, ctx context.Context, params map[string]interface{}) (*model.ToolResult, error)
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools      map[string]Tool
	schemas    map[string]map[string]interface{}
	registered map[string]Tool
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:      make(map[string]Tool),
		schemas:    make(map[string]map[string]interface{}),
		registered: make(map[string]Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool Tool) {
	r.registered[tool.Name()] = tool
	if multiTool, ok := tool.(MultiFunctionTool); ok {
		for _, schema := range multiTool.GetTools() {
			name, ok := schema["name"].(string)
			if !ok || name == "" {
				continue
			}
			r.tools[name] = tool
			r.schemas[name] = schema
		}
		return
	}
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List 返回所有工具
func (r *ToolRegistry) List() []Tool {
	result := make([]Tool, 0, len(r.registered))
	for _, tool := range r.registered {
		result = append(result, tool)
	}
	return result
}

// GetToolsForLLM 返回适合 LLM 调用的工具格式（llmcore 强类型）
//
// 阶段 1d 改造点：返回 []llmcore.ToolSpec 而非 []map。
// 业务侧 / LLM adapter 只看到协议级类型，不再拼接 map。
func (r *ToolRegistry) GetToolsForLLM() []llmcore.ToolSpec {
	result := make([]llmcore.ToolSpec, 0, len(r.tools))
	for name, schema := range r.schemas {
		parameters, _ := schema["parameters"].(map[string]interface{})
		description, _ := schema["description"].(string)
		result = append(result, llmcore.ToolSpec{
			Type: "function",
			Function: llmcore.ToolSpecFunction{
				Name:        name,
				Description: description,
				Parameters:  parameters,
			},
			ReadOnly: isReadOnlyToolSchema(name),
		})
	}

	for name, tool := range r.registered {
		if _, ok := tool.(MultiFunctionTool); ok {
			continue
		}
		result = append(result, llmcore.ToolSpec{
			Type: "function",
			Function: llmcore.ToolSpecFunction{
				Name:        name,
				Description: tool.Description(),
				Parameters:  tool.Parameters(),
			},
			ReadOnly: tool.ReadOnly(),
		})
	}
	return result
}

// isReadOnlyToolSchema 根据工具名给出保守的只读判断。
// 先保证 shell/browser/a2a 这类显式写操作默认为 false，其余默认 true。
func isReadOnlyToolSchema(name string) bool {
	if strings.HasPrefix(name, "message_") {
		return false
	}
	switch name {
	case "shell", "browser", "a2a", "file", "message":
		return false
	default:
		return true
	}
}
