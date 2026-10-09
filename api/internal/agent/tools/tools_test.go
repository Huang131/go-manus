package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Huang131/go-manus/api/internal/a2a"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// mockTool 模拟工具用于测试
type mockTool struct {
	nameVal        string
	descriptionVal string
	readOnlyVal    bool
}

type dynamicTestTool struct {
	name  string
	tools []map[string]interface{}
}

func (t *dynamicTestTool) Name() string { return t.name }

func (t *dynamicTestTool) Description() string { return "dynamic test tool" }

func (t *dynamicTestTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}

func (t *dynamicTestTool) ReadOnly() bool { return true }

func (t *dynamicTestTool) Invoke(context.Context, map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolResult("ok"), nil
}

func (t *dynamicTestTool) GetTools() []map[string]interface{} { return t.tools }

func (t *dynamicTestTool) InvokeWithName(string, context.Context, map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolResult("ok"), nil
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

func TestToolRegistryGetToolsForLLMSortsByName(t *testing.T) {
	registry := NewToolRegistry()
	registry.Register(NewMessageTool())
	registry.Register(&mockTool{nameVal: "z_tool", descriptionVal: "z"})
	registry.Register(&mockTool{nameVal: "a_tool", descriptionVal: "a"})

	for i := 0; i < 20; i++ {
		tools := registry.GetToolsForLLM()
		for j := 1; j < len(tools); j++ {
			if tools[j-1].Function.Name > tools[j].Function.Name {
				t.Fatalf("tool order = %q before %q, want sorted order", tools[j-1].Function.Name, tools[j].Function.Name)
			}
		}
	}
}

func TestToolRegistryReRegisterRemovesStaleFunctions(t *testing.T) {
	tool := &dynamicTestTool{
		name: "dynamic",
		tools: []map[string]interface{}{
			{"name": "dynamic_old", "description": "old", "parameters": map[string]interface{}{"type": "object"}},
			{"name": "dynamic_current", "description": "current", "parameters": map[string]interface{}{"type": "object"}},
		},
	}
	registry := NewToolRegistry()
	registry.Register(tool)

	tool.tools = []map[string]interface{}{
		{"name": "dynamic_current", "description": "updated", "parameters": map[string]interface{}{"type": "object"}},
	}
	registry.Register(tool)

	if _, ok := registry.Get("dynamic_old"); ok {
		t.Fatal("re-registering a dynamic tool retained stale function")
	}
	specs := registry.GetToolsForLLM()
	if len(specs) != 1 || specs[0].Function.Description != "updated" {
		t.Fatalf("re-registered specs = %+v, want one updated function", specs)
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

	// 声明即实现：schema 的每个属性都必须被 Invoke 实际读取（如 agent_id/task 由
	// call_agent 消费），反向也不得出现只声明不消费的参数——历史上 context 就是
	// 一个被静默丢弃的字段，会误导 LLM 传入无效参数。
	wantProps := map[string]bool{"action": true, "agent_id": true, "task": true}
	for name := range properties {
		if !wantProps[name] {
			t.Errorf("Parameters() declares unexpected property %q (must be consumed by Invoke)", name)
		}
	}
	for name := range wantProps {
		if _, ok := properties[name]; !ok {
			t.Errorf("Parameters() should declare %q field", name)
		}
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

// TestA2ATool_Invoke_UnknownActionFailsLoud 守护未知 action 必须报错而非静默兜底。
// 历史实现会把拼错的 action（如 "listagent"）默默当成 call_agent，触发真实的远程执行。
func TestA2ATool_Invoke_UnknownActionFailsLoud(t *testing.T) {
	tool := NewA2ATool()

	for _, action := range []string{"listagent", "call", "LIST_AGENTS"} {
		t.Run(action, func(t *testing.T) {
			result, err := tool.Invoke(context.Background(), map[string]interface{}{
				"action": action,
			})
			if err != nil {
				t.Fatalf("Invoke() error = %v, want nil（业务错误应走 ToolResult 双通道）", err)
			}
			if result.Success {
				t.Fatal("Invoke() should fail for unknown action, got success")
			}
			if !strings.Contains(result.Message, action) {
				t.Errorf("Invoke() message = %q, want it to mention the invalid action %q", result.Message, action)
			}
		})
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

func TestA2ATool_InitializeSkipsDuplicateAgentNames(t *testing.T) {
	newCardServer := func(hits *int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(hits, 1)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"dup-agent","url":"http://placeholder"}`)
		}))
	}

	var hitsA, hitsB int32
	srvA := newCardServer(&hitsA)
	defer srvA.Close()
	srvB := newCardServer(&hitsB)
	defer srvB.Close()

	tool := NewA2ATool()
	err := tool.Initialize(context.Background(), &A2AConfig{
		Agents: []A2AAgent{
			{Name: "dup-agent", URL: srvA.URL},
			{Name: "dup-agent", URL: srvB.URL},
		},
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	// 同名 Agent 只保留先配置者：第一个服务被请求一次，重复配置的服务不应被访问。
	if got := atomic.LoadInt32(&hitsA); got != 1 {
		t.Errorf("first server hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&hitsB); got != 0 {
		t.Errorf("duplicate server hits = %d, want 0 (duplicate config should be skipped)", got)
	}

	if n := len(tool.GetManager().GetAgentCards()); n != 1 {
		t.Errorf("GetAgentCards() count = %d, want 1", n)
	}
}

// TestA2ATool_ListAgentsOmitsEnabledField 守护 list_agents 输出不再携带死字段 enabled。
// 被停用的 Agent 已在 RuntimeA2AConfig 边界被过滤，运行时 Enabled 恒为 true，无信息量。
func TestA2ATool_ListAgentsOmitsEnabledField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":"agent-x","description":"a demo agent","url":"http://placeholder"}`)
	}))
	defer srv.Close()

	tool := NewA2ATool()
	if err := tool.Initialize(context.Background(), &A2AConfig{
		Agents: []A2AAgent{{Name: "agent-x", URL: srv.URL}},
	}); err != nil {
		t.Fatalf("Initialize() error = %v, want nil", err)
	}

	result, err := tool.Invoke(context.Background(), map[string]interface{}{
		"action": A2AActionListAgents,
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v, want nil", err)
	}
	if !result.Success {
		t.Fatalf("Invoke() = %+v, want success", result)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("result.Data type = %T, want map[string]interface{}", result.Data)
	}
	agents, ok := data["agents"].([]map[string]interface{})
	if !ok {
		t.Fatalf("agents type = %T, want []map[string]interface{}", data["agents"])
	}
	if len(agents) != 1 {
		t.Fatalf("agents len = %d, want 1", len(agents))
	}
	if _, exists := agents[0]["enabled"]; exists {
		t.Error("list_agents output should not expose dead 'enabled' field")
	}
	if agents[0]["id"] != "agent-x" {
		t.Errorf("agents[0][id] = %v, want agent-x", agents[0]["id"])
	}
}

func TestA2ATool_ListAgentsSortsByID(t *testing.T) {
	ids := sortedAgentIDs(map[string]*a2a.A2AAgentCard{
		"z-agent": {},
		"a-agent": {},
	})
	if got, want := strings.Join(ids, ","), "a-agent,z-agent"; got != want {
		t.Fatalf("sortedAgentIDs() = %q, want %q", got, want)
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
