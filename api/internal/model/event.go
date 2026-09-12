package model

import (
	"encoding/json"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// BaseEvent 事件接口
type BaseEvent interface {
	// GetType 返回事件类型
	GetType() EventType
	// ToJSON 将事件转换为 JSON 字符串
	ToJSON() string
}

// toJSON 统一的事件序列化入口
// 序列化失败时记录告警并返回空串
// 避免非法 json.RawMessage 静默变成空事件污染 Redis/SSE 下游。
func toJSON(v any) string {
	s, err := sonic.MarshalString(v)
	if err != nil {
		logger.Warn("event serialize failed", logger.Err(err))
		return ""
	}
	return s
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
)

// EventType 事件类型
type EventType string

const (
	EventTypeMessage      EventType = "message"
	EventTypeMessageDelta EventType = "message_delta"
	EventTypeMessageDone  EventType = "message_done"
	EventTypePlan         EventType = "plan"
	EventTypeToolCalling  EventType = "tool_calling"
	EventTypeToolCalled   EventType = "tool_called"
	EventTypeStep         EventType = "step"
	EventTypeError        EventType = "error"
	EventTypeTitle        EventType = "title"
	EventTypeWait         EventType = "wait"
	EventTypeDone         EventType = "done"
	EventTypeBrowser      EventType = "browser"
	EventTypeSearch       EventType = "search"
	EventTypeShell        EventType = "shell"
	EventTypeShellOutput  EventType = "shell_output"
	EventTypeFile         EventType = "file"
	EventTypeMCP          EventType = "mcp"
	EventTypeA2A          EventType = "a2a"
)

// PlanEventStatus 计划事件状态
type PlanEventStatus string

const (
	PlanEventStatusCreated   PlanEventStatus = "created"
	PlanEventStatusUpdated   PlanEventStatus = "updated"
	PlanEventStatusCompleted PlanEventStatus = "completed"
)

// StepEventStatus 步骤事件状态
type StepEventStatus string

const (
	StepEventStatusStarted   StepEventStatus = "started"
	StepEventStatusCompleted StepEventStatus = "completed"
	StepEventStatusFailed    StepEventStatus = "failed"
)

// Event 事件模型
// 每个事件代表 Agent 执行过程中的一个原子动作或状态变化。
type Event struct {
	ID        string          `json:"id"`         // Redis Stream 游标（ms-seq），用于 SSE 续读
	Type      EventType       `json:"type"`       // 事件类型，标识事件种类
	CreatedAt time.Time       `json:"created_at"` // 创建时间
	Data      json.RawMessage `json:"data"`       // 事件负载，类型由 Type 决定
}

// ToJSON 将事件转换为 JSON 字符串
func (e *Event) ToJSON() string { return toJSON(e) }

// MessageEvent 消息事件
type MessageEvent struct {
	Type        EventType `json:"type"`
	Role        string    `json:"role"`                  // 消息角色: user, assistant
	Message     string    `json:"message"`               // 消息本身
	Attachments []File    `json:"attachments,omitempty"` // 附件列表
}

// MessageDeltaEvent 表示助手消息的一段文本增量。
// MessageID 在同一条助手消息的所有增量中保持不变，Sequence 从 1 开始递增，
// 前端据此进行幂等聚合和断线重放。
type MessageDeltaEvent struct {
	MessageID string `json:"message_id"`
	Delta     string `json:"delta"`
	Sequence  int    `json:"sequence"`
}

func (e *MessageDeltaEvent) GetType() EventType { return EventTypeMessageDelta }

func (e *MessageDeltaEvent) ToJSON() string { return toJSON(e) }

// MessageDoneEvent 表示一条助手消息已经完成，Content 是聚合后的完整文本。
type MessageDoneEvent struct {
	MessageID    string `json:"message_id"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
}

func (e *MessageDoneEvent) GetType() EventType { return EventTypeMessageDone }

func (e *MessageDoneEvent) ToJSON() string { return toJSON(e) }

// GetType 返回事件类型
func (e *MessageEvent) GetType() EventType {
	return EventTypeMessage
}

// ToJSON 将事件转换为 JSON 字符串
func (e *MessageEvent) ToJSON() string { return toJSON(e) }

// Plan 执行计划，Agent 接收用户任务后生成的分解步骤。
type Plan struct {
	ID       string          `json:"id"`              // 计划唯一 ID
	Title    string          `json:"title"`           // 计划标题
	Goal     string          `json:"goal"`            // 用户原始需求描述
	Language string          `json:"language"`        // 计划生成语言，可能与用户输入语言不同（如中文问题生成英文计划）
	Steps    []PlanStep      `json:"steps"`           // 分解后的执行步骤
	Message  string          `json:"message"`         // LLM 生成计划时的原始输出
	Status   ExecutionStatus `json:"status"`          // 计划整体状态
	Error    string          `json:"error,omitempty"` // 计划执行失败时的错误信息
}

// Done 是否完成
func (p *Plan) Done() bool {
	return p.Status == ExecutionStatusCompleted || p.Status == ExecutionStatusFailed
}

// GetNextStep 获取下一个未完成的步骤
func (p *Plan) GetNextStep() *PlanStep {
	for i := range p.Steps {
		if !p.Steps[i].Done() {
			return &p.Steps[i]
		}
	}
	return nil
}

// PlanStep 计划中的单个执行步骤
type PlanStep struct {
	ID           string          `json:"id"`                      // 步骤唯一 ID
	Description  string          `json:"description"`             // 步骤描述，说明该步骤要做什么
	Status       ExecutionStatus `json:"status"`                  // 步骤执行状态
	Result       string          `json:"result,omitempty"`        // 步骤执行结果/输出
	Error        string          `json:"error,omitempty"`         // 步骤执行失败时的错误信息
	Success      bool            `json:"success"`                 // 步骤是否成功完成
	Attachments  []string        `json:"attachments,omitempty"`   // 步骤产生附件（如生成的图片、文件路径）
	UserQuestion string          `json:"user_question,omitempty"` // 等待用户输入时的问题，如 "请确认是否继续？"
}

// Done 步骤是否完成
func (s *PlanStep) Done() bool {
	return s.Status == ExecutionStatusCompleted || s.Status == ExecutionStatusFailed
}

// ErrorEvent 错误事件
type ErrorEvent struct {
	Message string `json:"message"`
}

// GetType 返回事件类型
func (e *ErrorEvent) GetType() EventType {
	return EventTypeError
}

// ToJSON 将事件转换为 JSON 字符串
func (e *ErrorEvent) ToJSON() string { return toJSON(e) }

// DoneEvent 完成事件
type DoneEvent struct {
	Message string `json:"message,omitempty"`
}

// GetType 返回事件类型
func (e *DoneEvent) GetType() EventType {
	return EventTypeDone
}

// ToJSON 将事件转换为 JSON 字符串
func (e *DoneEvent) ToJSON() string { return toJSON(e) }

// BrowserEvent 浏览器自动化事件，记录 Agent 操作浏览器时的状态。
type BrowserEvent struct {
	URL        string `json:"url"`                  // 当前页面 URL
	Action     string `json:"action"`               // 动作类型：navigate（导航）、click（点击）、input（输入）、screenshot（截图）
	Content    string `json:"content"`              // 页面内容或用户输入的文本
	Screenshot string `json:"screenshot,omitempty"` // Base64 编码的截图数据
}

// SearchEvent 搜索事件，记录 Agent 调用搜索工具的结果。
type SearchEvent struct {
	Query  string         `json:"query"`            // 搜索关键词
	Result *SearchResults `json:"result,omitempty"` // 搜索结果，为空表示搜索失败
}

// ShellEvent Shell 命令执行事件
type ShellEvent struct {
	Command  string `json:"command"`          // 执行的命令
	Output   string `json:"output,omitempty"` // 标准输出
	Error    string `json:"error,omitempty"`  // 标准错误输出
	ExitCode int    `json:"exit_code"`        // 进程退出码，0 表示成功
}

// FileEvent 文件操作事件，记录 Agent 读写文件的行为。
type FileEvent struct {
	Path    string `json:"path"`              // 文件路径
	Action  string `json:"action"`            // 操作类型：read（读取）、write（写入）、delete（删除）
	Content string `json:"content,omitempty"` // 文件内容（读操作返回内容，写操作记录写入内容）
	Error   string `json:"error,omitempty"`   // 操作失败时的错误信息
}

// TitleEvent 标题事件
type TitleEvent struct {
	Title string `json:"title"`
}

// GetType 返回事件类型
func (e *TitleEvent) GetType() EventType {
	return EventTypeTitle
}

// ToJSON 将事件转换为 JSON 字符串
func (e *TitleEvent) ToJSON() string { return toJSON(e) }

// WaitEvent 等待事件
type WaitEvent struct {
	Message string `json:"message,omitempty"`
}

// GetType 返回事件类型
func (e *WaitEvent) GetType() EventType {
	return EventTypeWait
}

// ToJSON 将事件转换为 JSON 字符串
func (e *WaitEvent) ToJSON() string { return toJSON(e) }

// PlanEvent 计划事件（用于 Agent 间传递）
type PlanEvent struct {
	Plan   Plan            `json:"plan"`
	Status PlanEventStatus `json:"status"`
}

// GetType 返回事件类型
func (e *PlanEvent) GetType() EventType {
	return EventTypePlan
}

// ToJSON 将事件转换为 JSON 字符串
func (e *PlanEvent) ToJSON() string { return toJSON(e) }

// StepEvent 步骤事件（用于 Agent 间传递，对应 SSE 协议中的 step 类型）
type StepEvent struct {
	Step   PlanStep        `json:"step"`
	Status StepEventStatus `json:"status"`
}

// GetType 返回事件类型
func (e *StepEvent) GetType() EventType {
	return EventTypeStep
}

// ToJSON 将事件转换为 JSON 字符串
func (e *StepEvent) ToJSON() string { return toJSON(e) }

// NewErrorEvent 创建错误事件
func NewErrorEvent(message string) *ErrorEvent {
	return &ErrorEvent{Message: message}
}

// NewTitleEvent 创建标题事件
func NewTitleEvent(title string) *TitleEvent {
	return &TitleEvent{Title: title}
}

// NewMessageEvent 创建消息事件
func NewMessageEvent(role, content string) *MessageEvent {
	return &MessageEvent{
		Type:    EventTypeMessage,
		Role:    role,
		Message: content,
	}
}

// NewMessageDeltaEvent 创建消息增量事件。
func NewMessageDeltaEvent(messageID, delta string, sequence int) *MessageDeltaEvent {
	return &MessageDeltaEvent{MessageID: messageID, Delta: delta, Sequence: sequence}
}

// NewMessageDoneEvent 创建消息完成事件。
func NewMessageDoneEvent(messageID, content, finishReason string) *MessageDoneEvent {
	return &MessageDoneEvent{
		MessageID:    messageID,
		Content:      content,
		FinishReason: finishReason,
	}
}

// NewPlanEvent 创建计划事件
func NewPlanEvent(plan Plan, status PlanEventStatus) *PlanEvent {
	plan.Steps = clonePlanSteps(plan.Steps)
	return &PlanEvent{
		Plan:   plan,
		Status: status,
	}
}

// NewStepEvent 创建步骤事件
func NewStepEvent(step PlanStep, status StepEventStatus) *StepEvent {
	step.Attachments = cloneStrings(step.Attachments)
	return &StepEvent{
		Step:   step,
		Status: status,
	}
}

// clonePlanSteps 复制计划步骤及其附件，确保事件只保存创建时的快照。
func clonePlanSteps(steps []PlanStep) []PlanStep {
	if steps == nil {
		return nil
	}
	cloned := make([]PlanStep, len(steps))
	copy(cloned, steps)
	for i := range cloned {
		cloned[i].Attachments = cloneStrings(steps[i].Attachments)
	}
	return cloned
}

// cloneStrings 保留 nil 与空但非 nil 切片的区别，避免改变 JSON 序列化结果。
func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

// NewDoneEvent 创建完成事件
func NewDoneEvent() *DoneEvent {
	return &DoneEvent{}
}

// NewWaitEvent 创建等待事件
func NewWaitEvent() *WaitEvent {
	return &WaitEvent{}
}

// ToolCallingEvent 工具调用中事件，Agent 开始调用工具时产生。
type ToolCallingEvent struct {
	Name         string                 `json:"name"`          // 事件标识符，固定为 "tool_calling"
	ToolCallID   string                 `json:"tool_call_id"`  // LLM 返回的 tool_use block ID，用于关联调用和结果
	FunctionName string                 `json:"function_name"` // 工具函数名，如 "bash", "read_file", "search"
	Arguments    map[string]interface{} `json:"arguments"`     // 工具参数，JSON 对象格式
}

// GetType 返回事件类型
func (e *ToolCallingEvent) GetType() EventType {
	return EventTypeToolCalling
}

// ToJSON 将事件转换为 JSON 字符串
func (e *ToolCallingEvent) ToJSON() string { return toJSON(e) }

// ToolCalledEvent 工具调用完成事件，工具执行完成后产生。
type ToolCalledEvent struct {
	Name         string                 `json:"name"`          // 事件标识符，固定为 "tool_called"
	ToolCallID   string                 `json:"tool_call_id"`  // 与 ToolCallingEvent 的 ToolCallID 对应
	FunctionName string                 `json:"function_name"` // 工具函数名
	Arguments    map[string]interface{} `json:"arguments"`     // 调用时的参数（可能与 ToolCallingEvent 不同，如敏感信息被脱敏）
	Result       *ToolResult            `json:"result"`        // 工具执行结果
}

// GetType 返回事件类型
func (e *ToolCalledEvent) GetType() EventType {
	return EventTypeToolCalled
}

// ToJSON 将事件转换为 JSON 字符串
func (e *ToolCalledEvent) ToJSON() string { return toJSON(e) }

// ShellOutputEvent 长命令运行期间的控制台输出增量快照。
// 由 BaseAgent 的 shell watcher 在 exec 返回 running 后周期性推送，
// 前端用它实时刷新对应 shell 工具的预览。
type ShellOutputEvent struct {
	SessionID string                   `json:"session_id"`
	Console   []map[string]interface{} `json:"console"`
}

// GetType 返回事件类型
func (e *ShellOutputEvent) GetType() EventType {
	return EventTypeShellOutput
}

// ToJSON 将事件转换为 JSON 字符串
func (e *ShellOutputEvent) ToJSON() string { return toJSON(e) }

// NewShellOutputEvent 创建 shell 输出事件
func NewShellOutputEvent(sessionID string, console []map[string]interface{}) *ShellOutputEvent {
	if console == nil {
		console = []map[string]interface{}{}
	}
	return &ShellOutputEvent{SessionID: sessionID, Console: console}
}

// NewToolCallingEvent 创建工具调用中事件
func NewToolCallingEvent(toolCallID, functionName string, arguments map[string]interface{}) *ToolCallingEvent {
	return &ToolCallingEvent{
		ToolCallID:   toolCallID,
		FunctionName: functionName,
		Arguments:    arguments,
	}
}

// NewToolCalledEvent 创建工具调用完成事件
func NewToolCalledEvent(toolCallID, functionName string, arguments map[string]interface{}, result *ToolResult) *ToolCalledEvent {
	return &ToolCalledEvent{
		ToolCallID:   toolCallID,
		FunctionName: functionName,
		Arguments:    arguments,
		Result:       result,
	}
}
