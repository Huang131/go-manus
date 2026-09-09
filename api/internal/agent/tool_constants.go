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

// Browser 工具动作。
const (
	BrowserActionNavigate   = "navigate"
	BrowserActionView       = "view"
	BrowserActionScreenshot = "screenshot"
	BrowserActionClick      = "click"
	BrowserActionInput      = "input"
	BrowserActionScrollUp   = "scroll_up"
	BrowserActionScrollDown = "scroll_down"
	BrowserActionPressKey   = "press_key"
)

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
