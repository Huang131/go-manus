package agent

// 工具名称是 Agent 与 LLM 之间的稳定协议值。
const (
	ToolNameShell   = "shell"
	ToolNameBrowser = "browser"
	ToolNameFile    = "file"
	ToolNameSearch  = "search"
	ToolNameMessage = "message"
	ToolNameMCP     = "mcp"
	ToolNameA2A     = "a2a"
)

const (
	MessageFunctionPrefix = ToolNameMessage + "_"
	MCPFunctionPrefix     = ToolNameMCP + "_"
)

// File 工具动作。
const (
	FileActionRead    = "read"
	FileActionWrite   = "write"
	FileActionDelete  = "delete"
	FileActionExists  = "exists"
	FileActionList    = "list"
	FileActionSearch  = "search"
	FileActionReplace = "replace"
)

// Shell 工具动作。
const (
	ShellActionExec  = "exec"
	ShellActionRead  = "read"
	ShellActionWrite = "write"
	ShellActionWait  = "wait"
	ShellActionKill  = "kill"
)

// Browser 工具动作，与沙箱侧浏览器端点一一对应。
const (
	BrowserActionNavigate    = "navigate"
	BrowserActionSnapshot    = "snapshot"
	BrowserActionScreenshot  = "screenshot"
	BrowserActionClick       = "click"
	BrowserActionInput       = "input"
	BrowserActionPressKey    = "press_key"
	BrowserActionScroll      = "scroll"
	BrowserActionConsoleExec = "console_exec"
	BrowserActionConsoleView = "console_view"
)

// browserDisplayScreenshot 截图在 ToolResult.Display 中的键名：
// data URI 只给 UI 渲染，不进入 LLM 上下文。
const browserDisplayScreenshot = "screenshot"

// A2A 工具动作。
const (
	A2AActionListAgents = "list_agents"
	A2AActionCallAgent  = "call_agent"
)

// Message 工具函数名。
const (
	MessageFunctionNotifyUser = "message_notify_user"
	MessageFunctionAskUser    = "message_ask_user"
)
