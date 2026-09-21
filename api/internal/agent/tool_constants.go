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

// browserScreenshotArtifact 截图产物的键名：ToolResult.Artifacts 用它挂载 PNG 字节，
// 运行期落存储后会在 ToolResult.Display 的同名键上留下文件引用。
const browserScreenshotArtifact = "screenshot"

// 截图产物落存储时使用的元数据，UI 预览按这些信息渲染。
const (
	browserScreenshotFilename = "screenshot.png"
	browserScreenshotMimeType = "image/png"
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
