# 工具系统 (Tool System)

## 1. 概述

go-manus 的工具系统基于 **MCP (Model Context Protocol)** 协议设计，提供标准化的工具定义和调用机制。

```
┌─────────────────────────────────────────────────────────────┐
│                      ReAct Agent                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Tool Registry (tools.go)               │   │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │   │
│  │  │  File   │ │  Shell  │ │Browser  │ │  A2A    │   │   │
│  │  │  Tool   │ │  Tool   │ │  Tool   │ │  Tool   │   │   │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘   │   │
│  └───────┼───────────┼───────────┼───────────┼─────────┘   │
│          │           │           │           │              │
└──────────┼───────────┼───────────┼───────────┼──────────────┘
           │           │           │           │
           ▼           ▼           ▼           ▼
        FileSystem   Docker     Browser    Remote Agent
```

## 2. 核心接口

### 2.1 Tool 接口

```go
// api/internal/agent/tools.go
type Tool interface {
    // 获取工具定义（用于 LLM 理解工具能力）
    GetDefinition() ToolDefinition
    
    // 执行工具
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
    
    // 工具名称
    GetName() string
}
```

### 2.2 工具定义

```go
type ToolDefinition struct {
    Name        string              // 工具名称
    Description string              // 工具描述（给 LLM 看）
    Parameters  []ParameterDefinition // 参数定义
}
```

### 2.3 执行结果

```go
type ToolResult struct {
    Success bool                   // 是否成功
    Output  string                 // 输出内容
    Error   string                 // 错误信息（失败时）
    Files   []string               // 产生的文件列表
}
```

## 3. 内置工具

### 3.1 FileTool - 文件操作

**功能**：读写文件、列表目录

```go
type FileTool struct {
    fileService *service.FileService
}
```

**定义**：
```go
func (t *FileTool) GetDefinition() ToolDefinition {
    return ToolDefinition{
        Name:        "file",
        Description: "文件操作工具，支持读写文件和列表目录",
        Parameters: []ParameterDefinition{
            {Name: "action", Type: "string", Enum: []string{"read", "write", "list"}, Required: true},
            {Name: "path", Type: "string", Required: true},
            {Name: "content", Type: "string", Required: false},
        },
    }
}
```

**使用示例**：
```json
{
  "action": "read",
  "path": "/workspace/main.go",
  "content": null
}
```

### 3.2 ShellTool - Shell 命令执行

**功能**：在沙箱环境中执行 Shell 命令

```go
type ShellTool struct {
    sandboxClient *external.SandboxClient
}
```

**定义**：
```go
{
    "name": "shell",
    "description": "在沙箱中执行 Shell 命令",
    "parameters": [
        {"name": "command", "type": "string", "required": true},
        {"name": "timeout", "type": "integer", "required": false}
    ]
}
```

**使用示例**：
```json
{
  "command": "go build -o app main.go",
  "timeout": 60
}
```

### 3.3 BrowserTool - 浏览器自动化

**功能**：打开网页、点击、输入、截图

```go
type BrowserTool struct {
    browserClient *external.BrowserClient
}
```

**定义**：
```go
{
    "name": "browser",
    "description": "浏览器自动化工具，控制浏览器操作网页",
    "parameters": [
        {"name": "action", "type": "string", "enum": ["open", "click", "input", "screenshot", "get_html"]},
        {"name": "url", "type": "string"},
        {"name": "selector", "type": "string"},
        {"name": "value", "type": "string"},
        {"name": "timeout", "type": "integer"}
    ]
}
```

### 3.4 SearchTool - 搜索功能

**功能**：网络搜索

```go
type SearchTool struct {
    searchClient *external.SearchClient
}
```

**定义**：
```go
{
    "name": "search",
    "description": "网络搜索工具",
    "parameters": [
        {"name": "query", "type": "string", "required": true},
        {"name": "num_results", "type": "integer", "required": false}
    ]
}
```

### 3.5 MessageTool - 消息通知

**功能**：发送消息给用户

```go
type MessageTool struct{}
```

**使用**：在需要向用户反馈时使用

### 3.6 A2ATool - Agent 间通信

**功能**：与其他 Agent 协作

```go
type A2ATool struct {
    agents map[string]Agent
}
```

**定义**：
```go
{
    "name": "a2a",
    "description": "Agent 间通信工具，用于委托任务给其他 Agent",
    "parameters": [
        {"name": "agent_id", "type": "string", "required": true},
        {"name": "message", "type": "string", "required": true}
    ]
}
```

## 4. MCP 集成

### 4.1 MCP 客户端

```go
// api/internal/agent/tool_mcp.go
type MCPTool struct {
    client *mcp.Client
    name   string
}
```

支持连接外部 MCP 服务器，扩展工具能力：

```go
// 连接到 MCP 服务器
mcpClient, err := mcp.NewClient("http://mcp-server:8080")

// 获取远程工具
tools, err := mcpClient.ListTools()
```

### 4.2 工具注册

```go
// api/internal/agent/tools.go
func RegisterTools(
    fileService *service.FileService,
    sandboxClient *external.SandboxClient,
    browserClient *external.BrowserClient,
    searchClient *external.SearchClient,
    mcpClients []*mcp.Client,
) []Tool {
    tools := []Tool{
        NewFileTool(fileService),
        NewShellTool(sandboxClient),
        NewBrowserTool(browserClient),
        NewSearchTool(searchClient),
        NewMessageTool(),
    }
    
    // 添加 A2A 工具
    tools = append(tools, NewA2ATool(agentRegistry))
    
    // 添加 MCP 工具
    for _, client := range mcpClients {
        tools = append(tools, NewMCPTool(client))
    }
    
    return tools
}
```

## 5. LLM 工具调用

### 5.1 转换为 LLM 格式

```go
// 获取 LLM 可用的工具定义
func (a *BaseAgent) GetToolsForLLM() []map[string]interface{} {
    var result []map[string]interface{}
    for _, tool := range a.tools {
        def := tool.GetDefinition()
        result = append(result, map[string]interface{}{
            "type": "function",
            "function": map[string]interface{}{
                "name":        def.Name,
                "description": def.Description,
                "parameters":  def.ToOpenAISchema(), // 转换为 OpenAI 格式
            },
        })
    }
    return result
}
```

### 5.2 解析 LLM 响应

```go
// 从 LLM 响应中提取工具调用
func parseToolCalls(response string) ([]ToolCall, error) {
    var calls []ToolCall
    // 解析 JSON 格式的 tool_calls
    json.Unmarshal([]byte(response), &calls)
    return calls, nil
}
```

## 6. 自定义工具开发

### 6.1 步骤 1：定义工具

```go
type MyTool struct {
    dependency *SomeService
}

func NewMyTool(dep *SomeService) *MyTool {
    return &MyTool{dependency: dep}
}
```

### 6.2 步骤 2：实现接口

```go
func (t *MyTool) GetName() string {
    return "my_tool"
}

func (t *MyTool) GetDefinition() ToolDefinition {
    return ToolDefinition{
        Name:        "my_tool",
        Description: "我的自定义工具",
        Parameters: []ParameterDefinition{
            {Name: "param1", Type: "string", Required: true},
            {Name: "param2", Type: "integer", Required: false},
        },
    }
}

func (t *MyTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
    param1, _ := params["param1"].(string)
    param2, _ := params["param2"].(int)
    
    // 执行逻辑
    result, err := t.dependency.DoSomething(param1, param2)
    if err != nil {
        return &ToolResult{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    
    return &ToolResult{
        Success: true,
        Output:  result,
    }, nil
}
```

### 6.3 步骤 3：注册工具

在 `RegisterTools()` 函数中添加：

```go
tools = append(tools, NewMyTool(dependency))
```

## 7. 安全考虑

### 7.1 沙箱隔离

Shell 命令在 Docker 沙箱中执行，与主机隔离：

```go
// sandbox/Dockerfile
FROM python:3.11-slim
# 只读文件系统
RUN pip install supervisord
# 限制资源
```

### 7.2 权限控制

- 文件操作限制在特定目录
- Shell 命令超时控制
- 敏感操作需用户确认

### 7.3 输入验证

```go
func validatePath(path string) error {
    // 防止路径遍历
    if strings.Contains(path, "..") {
        return errors.New("invalid path")
    }
    // 白名单目录
    allowedDirs := []string{"/workspace", "/tmp"}
    // ...
}
```

## 8. 最佳实践

1. **工具描述要清晰**：LLM 依靠描述理解工具用途
2. **参数定义要完整**：包含类型、是否必填、取值范围
3. **错误处理要友好**：返回有意义的错误信息
4. **考虑超时控制**：防止长时间运行的命令
5. **资源清理**：确保临时文件被清理