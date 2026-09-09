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
	EventTypeMessage     EventType = "message"
	EventTypePlan        EventType = "plan"
	EventTypeToolCalling EventType = "tool_calling"
	EventTypeToolCalled  EventType = "tool_called"
	EventTypeStep        EventType = "step"
	EventTypeError       EventType = "error"
	EventTypeTitle       EventType = "title"
	EventTypeWait        EventType = "wait"
	EventTypeDone        EventType = "done"
	EventTypeBrowser     EventType = "browser"
	EventTypeSearch      EventType = "search"
	EventTypeShell       EventType = "shell"
	EventTypeFile        EventType = "file"
	EventTypeMCP         EventType = "mcp"
	EventTypeA2A         EventType = "a2a"
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
type Event struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	CreatedAt time.Time       `json:"created_at"`
	Data      json.RawMessage `json:"data"`
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

// GetType 返回事件类型
func (e *MessageEvent) GetType() EventType {
	return EventTypeMessage
}

// ToJSON 将事件转换为 JSON 字符串
func (e *MessageEvent) ToJSON() string { return toJSON(e) }

// Plan 计划
type Plan struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Goal     string          `json:"goal"`
	Language string          `json:"language"`
	Steps    []PlanStep      `json:"steps"`
	Message  string          `json:"message"`
	Status   ExecutionStatus `json:"status"`
	Error    string          `json:"error,omitempty"`
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

// PlanStep 计划步骤
type PlanStep struct {
	ID           string          `json:"id"`
	Description  string          `json:"description"`
	Status       ExecutionStatus `json:"status"`
	Result       string          `json:"result,omitempty"`
	Error        string          `json:"error,omitempty"`
	Success      bool            `json:"success"`
	Attachments  []string        `json:"attachments,omitempty"`
	UserQuestion string          `json:"user_question,omitempty"` // 等待用户输入时的问题
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

// BrowserEvent 浏览器事件
type BrowserEvent struct {
	URL        string `json:"url"`
	Action     string `json:"action"`  // navigate/click/input/screenshot
	Content    string `json:"content"` // 页面内容或截图
	Screenshot string `json:"screenshot,omitempty"`
}

// SearchEvent 搜索事件
type SearchEvent struct {
	Query  string         `json:"query"`
	Result *SearchResults `json:"result,omitempty"`
}

// ShellEvent Shell 执行事件
type ShellEvent struct {
	Command  string `json:"command"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// FileEvent 文件操作事件
type FileEvent struct {
	Path    string `json:"path"`
	Action  string `json:"action"` // read/write/delete
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
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

// ToolCallingEvent 工具调用中事件
type ToolCallingEvent struct {
	Name         string                 `json:"name"`
	ToolCallID   string                 `json:"tool_call_id"`
	FunctionName string                 `json:"function_name"`
	Arguments    map[string]interface{} `json:"arguments"`
}

// GetType 返回事件类型
func (e *ToolCallingEvent) GetType() EventType {
	return EventTypeToolCalling
}

// ToJSON 将事件转换为 JSON 字符串
func (e *ToolCallingEvent) ToJSON() string { return toJSON(e) }

// ToolCalledEvent 工具调用完成事件
type ToolCalledEvent struct {
	Name         string                 `json:"name"`
	ToolCallID   string                 `json:"tool_call_id"`
	FunctionName string                 `json:"function_name"`
	Arguments    map[string]interface{} `json:"arguments"`
	Result       *ToolResult            `json:"result"`
}

// GetType 返回事件类型
func (e *ToolCalledEvent) GetType() EventType {
	return EventTypeToolCalled
}

// ToJSON 将事件转换为 JSON 字符串
func (e *ToolCalledEvent) ToJSON() string { return toJSON(e) }

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
