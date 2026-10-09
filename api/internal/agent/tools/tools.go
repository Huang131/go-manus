// Package tools 实现 Agent 可调用工具系统：Shell/File/Browser/Search 基础工具、
// MCP/A2A 外部代理工具，以及负责工具组装、MCP/A2A 热重载与清理的 ToolProvider。
// 本包不依赖 agent 编排层，仅依赖 sandbox、search、model 等基础能力，保持依赖单向。
package tools

import (
	"context"
	"sort"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// Tool 工具接口
type Tool interface {
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
// GetTools 返回动态 function 的协议描述；ToolRegistry 会将其转换为 ToolDescriptor，
// 因此普通工具和多函数工具最终使用同一份注册索引。
type MultiFunctionTool interface {
	Tool
	GetTools() []map[string]interface{}
	InvokeWithName(functionName string, ctx context.Context, params map[string]interface{}) (*model.ToolResult, error)
}

// ToolDescriptor 描述一个可被 Agent 调用的 function。
// 普通工具和动态工具都落到同一份描述中，避免 schema、实现和只读标记分开维护。
type ToolDescriptor struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	ReadOnly    bool
	Tool        Tool
	Owner       string
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	descriptors map[string]ToolDescriptor
}

// NewToolRegistry 创建工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		descriptors: make(map[string]ToolDescriptor),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool Tool) {
	if tool == nil {
		return
	}
	owner := tool.Name()
	for name, descriptor := range r.descriptors {
		if descriptor.Owner == owner {
			delete(r.descriptors, name)
		}
	}

	if multiTool, ok := tool.(MultiFunctionTool); ok {
		for _, schema := range multiTool.GetTools() {
			descriptor, ok := descriptorFromSchema(tool, owner, schema)
			if !ok {
				continue
			}
			r.descriptors[descriptor.Name] = descriptor
		}
		return
	}
	r.descriptors[owner] = ToolDescriptor{
		Name:        owner,
		Description: tool.Description(),
		Parameters:  tool.Parameters(),
		ReadOnly:    tool.ReadOnly(),
		Tool:        tool,
		Owner:       owner,
	}
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	descriptor, ok := r.descriptors[name]
	if !ok {
		return nil, false
	}
	return descriptor.Tool, true
}

// List 返回所有工具
func (r *ToolRegistry) List() []Tool {
	descriptors := r.sortedDescriptors()
	result := make([]Tool, 0, len(descriptors))
	seenOwners := make(map[string]struct{}, len(descriptors))
	for _, descriptor := range descriptors {
		if _, ok := seenOwners[descriptor.Owner]; ok {
			continue
		}
		seenOwners[descriptor.Owner] = struct{}{}
		result = append(result, descriptor.Tool)
	}
	return result
}

// GetToolsForLLM 返回适合 LLM 调用的工具格式（llmcore 强类型）
//
// 阶段 1d 改造点：返回 []llmcore.ToolSpec 而非 []map。
// 业务侧 / LLM adapter 只看到协议级类型，不再拼接 map。
func (r *ToolRegistry) GetToolsForLLM() []llmcore.ToolSpec {
	descriptors := r.sortedDescriptors()
	result := make([]llmcore.ToolSpec, 0, len(descriptors))
	for _, descriptor := range descriptors {
		result = append(result, llmcore.ToolSpec{
			Type: llmcore.ToolTypeFunction,
			Function: llmcore.ToolSpecFunction{
				Name:        descriptor.Name,
				Description: descriptor.Description,
				Parameters:  descriptor.Parameters,
			},
			ReadOnly: descriptor.ReadOnly,
		})
	}
	return result
}

func descriptorFromSchema(tool Tool, owner string, schema map[string]interface{}) (ToolDescriptor, bool) {
	name, ok := schema["name"].(string)
	if !ok || name == "" {
		return ToolDescriptor{}, false
	}
	parameters, _ := schema["parameters"].(map[string]interface{})
	description, _ := schema["description"].(string)
	return ToolDescriptor{
		Name:        name,
		Description: description,
		Parameters:  parameters,
		ReadOnly:    tool.ReadOnly(),
		Tool:        tool,
		Owner:       owner,
	}, true
}

func (r *ToolRegistry) sortedDescriptors() []ToolDescriptor {
	result := make([]ToolDescriptor, 0, len(r.descriptors))
	for _, descriptor := range r.descriptors {
		result = append(result, descriptor)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
