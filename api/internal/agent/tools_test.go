package agent

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// mockTool 模拟工具用于测试
type mockTool struct {
	nameVal        string
	descriptionVal string
	readOnlyVal    bool
}

func (m *mockTool) Name() string {
	return m.nameVal
}

func (m *mockTool) Description() string {
	return m.descriptionVal
}

func (m *mockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (m *mockTool) ReadOnly() bool {
	return m.readOnlyVal
}

func (m *mockTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolResult("mock result"), nil
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()
	tool := &mockTool{nameVal: "test_tool", descriptionVal: "A test tool"}

	registry.Register(tool)

	if _, ok := registry.Get("test_tool"); !ok {
		t.Error("Register() should register tool")
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()
	tool := &mockTool{nameVal: "test_tool", descriptionVal: "A test tool"}

	registry.Register(tool)

	got, ok := registry.Get("test_tool")
	if !ok {
		t.Error("Get() should return registered tool")
	}
	if got.Name() != "test_tool" {
		t.Errorf("Get() name = %s, want test_tool", got.Name())
	}
}

func TestToolRegistry_Get_NotFound(t *testing.T) {
	registry := NewToolRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for nonexistent tool")
	}
}

func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	registry.Register(&mockTool{nameVal: "tool1", descriptionVal: "Tool 1"})
	registry.Register(&mockTool{nameVal: "tool2", descriptionVal: "Tool 2"})

	tools := registry.List()
	if len(tools) != 2 {
		t.Errorf("List() got %d tools, want 2", len(tools))
	}
}

func TestToolRegistry_GetToolsForLLM(t *testing.T) {
	registry := NewToolRegistry()

	registry.Register(&mockTool{
		nameVal:        "test_tool",
		descriptionVal: "A test tool",
	})

	tools := registry.GetToolsForLLM()
	if len(tools) != 1 {
		t.Errorf("GetToolsForLLM() got %d tools, want 1", len(tools))
	}

	if tools[0].Type != llmcore.ToolTypeFunction {
		t.Error("GetToolsForLLM() should return function type")
	}
}

func TestToolRegistry_GetToolsForLLM_ReadOnlyFlag(t *testing.T) {
	registry := NewToolRegistry()

	registry.Register(&mockTool{
		nameVal:        "readonly_tool",
		descriptionVal: "readonly",
		readOnlyVal:    true,
	})

	tools := registry.GetToolsForLLM()
	if len(tools) != 1 {
		t.Fatalf("GetToolsForLLM() got %d tools, want 1", len(tools))
	}
	if !tools[0].ReadOnly {
		t.Error("GetToolsForLLM() should keep ReadOnly flag")
	}
}

func TestA2ATool_Name(t *testing.T) {
	tool := NewA2ATool()
	if tool.Name() != "a2a" {
		t.Errorf("Name() = %s, want a2a", tool.Name())
	}
}

func TestA2ATool_Description(t *testing.T) {
	tool := NewA2ATool()
	if tool.Description() == "" {
		t.Error("Description() should not be empty")
	}
}

func TestA2ATool_Parameters(t *testing.T) {
	tool := NewA2ATool()
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v, want object", params["type"])
	}

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters() should have properties")
	}

	// 检查必需字段
	if _, ok := properties["action"]; !ok {
		t.Error("Parameters() should have action field")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Parameters() should have required field")
	}

	found := false
	for _, r := range required {
		if r == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Parameters() required should include action")
	}
}

func TestA2ATool_Invoke_WithoutInit(t *testing.T) {
	tool := NewA2ATool()

	// 未初始化时调用 list_agents
	result, err := tool.Invoke(context.Background(), map[string]interface{}{
		"action": "list_agents",
	})

	if err != nil {
		t.Errorf("Invoke() error = %v, want nil", err)
	}

	if !result.Success {
		t.Logf("list_agents without init: %s", result.Message)
	}
}

func TestA2ATool_Invoke_CallAgentWithoutAgentID(t *testing.T) {
	tool := NewA2ATool()

	result, err := tool.Invoke(context.Background(), map[string]interface{}{
		"action":   "call_agent",
		"agent_id": "",
		"task":     "test task",
	})

	if err != nil {
		t.Errorf("Invoke() error = %v, want nil", err)
	}

	if result.Success {
		t.Error("Invoke() should return error for empty agent_id")
	}

	if result.Message == "" {
		t.Error("Invoke() should return error message for empty agent_id")
	}
}

func TestA2ATool_Invoke_CallAgentWithoutTask(t *testing.T) {
	tool := NewA2ATool()

	result, err := tool.Invoke(context.Background(), map[string]interface{}{
		"action":   "call_agent",
		"agent_id": "test-agent",
		"task":     "",
	})

	if err != nil {
		t.Errorf("Invoke() error = %v, want nil", err)
	}

	if result.Success {
		t.Error("Invoke() should return error for empty task")
	}
}

func TestA2ATool_Invoke_CallAgentRejectsInvalidRequiredTypes(t *testing.T) {
	t.Run("agent id type", func(t *testing.T) {
		result, err := NewA2ATool().Invoke(context.Background(), map[string]interface{}{
			"action":   A2AActionCallAgent,
			"agent_id": 123,
			"task":     "test task",
		})
		if err != nil {
			t.Fatalf("Invoke() error = %v, want nil", err)
		}
		if result.Success || result.Message != "agent_id 必须是非空字符串" {
			t.Fatalf("unexpected result: %+v", result)
		}
	})

	t.Run("task type", func(t *testing.T) {
		result, err := NewA2ATool().Invoke(context.Background(), map[string]interface{}{
			"action":   A2AActionCallAgent,
			"agent_id": "agent-1",
			"task":     true,
		})
		if err != nil {
			t.Fatalf("Invoke() error = %v, want nil", err)
		}
		if result.Success || result.Message != "task 必须是非空字符串" {
			t.Fatalf("unexpected result: %+v", result)
		}
	})
}

func TestA2ATool_MarshalResponseTextHandlesUnsupportedData(t *testing.T) {
	got := marshalResponseText(func() {})
	if got == "" {
		t.Fatal("marshalResponseText() returned empty string for unsupported data")
	}
	if got == "null" {
		t.Fatalf("marshalResponseText() silently returned null: %q", got)
	}
}

func TestToolValidationRejectsMissingRequiredParameters(t *testing.T) {
	t.Run("shell command", func(t *testing.T) {
		result, err := NewShellTool(nil).Invoke(context.Background(), map[string]interface{}{
			"action":     ShellActionExec,
			"session_id": "session-1",
		})
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		if result.Success || result.Message == "" {
			t.Fatalf("Invoke() = %+v, want parameter error", result)
		}
	})

	t.Run("file content", func(t *testing.T) {
		result, err := NewFileTool(nil).Invoke(context.Background(), map[string]interface{}{
			"action":   FileActionWrite,
			"filepath": "/tmp/a.txt",
		})
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		if result.Success || result.Message == "" {
			t.Fatalf("Invoke() = %+v, want parameter error", result)
		}
	})

	t.Run("browser key", func(t *testing.T) {
		result, err := NewBrowserTool(nil).Invoke(context.Background(), map[string]interface{}{
			"action":     BrowserActionPressKey,
			"session_id": "session-1",
		})
		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}
		if result.Success || result.Message == "" {
			t.Fatalf("Invoke() = %+v, want parameter error", result)
		}
	})
}

func TestA2ATool_Initialize(t *testing.T) {
	tool := NewA2ATool()

	// 初始化空的配置
	err := tool.Initialize(context.Background(), nil)
	if err != nil {
		t.Errorf("Initialize(nil) error = %v, want nil", err)
	}

	// 初始化空数组配置
	err = tool.Initialize(context.Background(), &A2AConfig{Agents: []A2AAgent{}})
	if err != nil {
		t.Errorf("Initialize(empty config) error = %v, want nil", err)
	}
}

func TestA2ATool_Cleanup(t *testing.T) {
	tool := NewA2ATool()

	// 初始化
	err := tool.Initialize(context.Background(), nil)
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil", err)
	}

	// 清理
	err = tool.Cleanup()
	if err != nil {
		t.Errorf("Cleanup() error = %v, want nil", err)
	}
}
