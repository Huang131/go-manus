package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

func parseStreamSeq(id string) uint64 {
	parts := strings.SplitN(id, "-", 2)
	seq, _ := strconv.ParseUint(parts[0], 10, 64)
	return seq
}

// ============================================================================
// 测试辅助组件
// ============================================================================

// inMemoryMessageQueue 内存版 MessageQueue，模拟 Redis Stream 的
// "游标推进 + 阻塞读取"语义，用于在集成测试中替换 Redis，
// 验证事件能从 agent → flow → output_stream 完整流转。
type inMemoryMessageQueue struct {
	mu      sync.Mutex
	streams map[string]*inMemoryStream
}

type inMemoryStream struct {
	nextSeq  uint64
	messages []inMemoryMessage
}

type inMemoryMessage struct {
	seq  uint64
	id   string
	data interface{}
}

// newInMemoryMessageQueue 创建内存消息队列
func newInMemoryMessageQueue() *inMemoryMessageQueue {
	return &inMemoryMessageQueue{streams: make(map[string]*inMemoryStream)}
}

// Put 追加一条消息，返回自增的流 ID
func (q *inMemoryMessageQueue) Put(ctx context.Context, streamName string, message interface{}) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	st := q.streams[streamName]
	if st == nil {
		st = &inMemoryStream{}
		q.streams[streamName] = st
	}
	st.nextSeq++
	id := fmt.Sprintf("%d-0", st.nextSeq)
	st.messages = append(st.messages, inMemoryMessage{seq: st.nextSeq, id: id, data: message})
	return id, nil
}

// fromSeqLocked 计算起始游标（调用方需持有锁）：
//   - 空 / "$"：只读调用开始后新增的消息（对齐 Redis XRead $ 语义）
//   - 具体 ID：严格大于该 ID 的消息
func (q *inMemoryMessageQueue) fromSeqLocked(streamName, startID string) uint64 {
	st := q.streams[streamName]
	if st == nil {
		return 0
	}
	if startID == "" || startID == "$" {
		return st.nextSeq
	}
	return parseStreamSeq(startID)
}

// GetBlocking 阻塞获取消息，支持 context 取消和超时
func (q *inMemoryMessageQueue) GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error) {
	blockTimeout := 3 * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		blockTimeout = timeout[0]
	}
	if blockTimeout > 5*time.Second {
		blockTimeout = 5 * time.Second
	}

	// 起始游标在调用开始时快照，保证 "" 语义为"只读新增消息"
	q.mu.Lock()
	fromSeq := q.fromSeqLocked(streamName, startID)
	q.mu.Unlock()

	deadline := time.Now().Add(blockTimeout)
	for {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}

		q.mu.Lock()
		if st := q.streams[streamName]; st != nil {
			for _, msg := range st.messages {
				if msg.seq > fromSeq {
					q.mu.Unlock()
					return msg.id, msg.data, nil
				}
			}
		}
		q.mu.Unlock()

		if time.Now().After(deadline) {
			return "", nil, nil // 超时无新消息
		}
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		case <-time.After(2 * time.Millisecond):
		}
	}
}

// Clear 清空队列
func (q *inMemoryMessageQueue) Clear(ctx context.Context, streamName string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.streams, streamName)
	return nil
}

func (q *inMemoryMessageQueue) SetRetention(context.Context, string, time.Duration) error {
	return nil
}

// IsEmpty 判断队列是否为空
func (q *inMemoryMessageQueue) IsEmpty(ctx context.Context, streamName string) (bool, error) {
	size, err := q.Size(ctx, streamName)
	return size == 0, err
}

// Size 获取队列长度
func (q *inMemoryMessageQueue) Size(ctx context.Context, streamName string) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	st := q.streams[streamName]
	if st == nil {
		return 0, nil
	}
	return int64(len(st.messages)), nil
}

// mockFailingTool 测试工具：返回错误结果，用于测试 tool_called 失败场景
type mockFailingTool struct{}

func (t *mockFailingTool) Name() string        { return "failing_tool" }
func (t *mockFailingTool) Description() string { return "总是失败的测试工具" }
func (t *mockFailingTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}},
	}
}
func (t *mockFailingTool) ReadOnly() bool { return true }
func (t *mockFailingTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolError("工具执行失败：参数无效"), nil
}

// mockEchoTool 测试工具：记录调用参数并返回回声结果
type mockEchoTool struct{}

func (t *mockEchoTool) Name() string        { return "test_tool" }
func (t *mockEchoTool) Description() string { return "回声测试工具" }
func (t *mockEchoTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}},
	}
}
func (t *mockEchoTool) ReadOnly() bool { return true }
func (t *mockEchoTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	return model.NewToolResult(map[string]interface{}{"echo": params["query"]}), nil
}

// mockSessionRepo 测试用会话仓储：记录事件与状态更新，供集成测试断言
type mockSessionRepo struct {
	mu       sync.Mutex
	events   []*model.Event
	statuses []model.SessionStatus
}

func (m *mockSessionRepo) Create(ctx context.Context, session *model.Session) error { return nil }
func (m *mockSessionRepo) GetByID(ctx context.Context, id string) (*model.Session, error) {
	return &model.Session{ID: id}, nil
}
func (m *mockSessionRepo) GetAll(ctx context.Context) ([]*model.Session, error) { return nil, nil }
func (m *mockSessionRepo) List(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	return nil, 0, nil
}
func (m *mockSessionRepo) Update(ctx context.Context, session *model.Session) error { return nil }
func (m *mockSessionRepo) Delete(ctx context.Context, id string) error              { return nil }
func (m *mockSessionRepo) AppendEvent(ctx context.Context, id string, event *model.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}
func (m *mockSessionRepo) GetMemory(ctx context.Context, id, agentName string) ([]llmcore.Message, error) {
	return nil, nil
}
func (m *mockSessionRepo) SaveMemory(ctx context.Context, id, agentName string, messages []llmcore.Message) error {
	return nil
}
func (m *mockSessionRepo) UpdateTitle(ctx context.Context, id, title string) error { return nil }
func (m *mockSessionRepo) UpdateLatestMessage(ctx context.Context, id, message string) error {
	return nil
}
func (m *mockSessionRepo) UpdateStatus(ctx context.Context, id string, status model.SessionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statuses = append(m.statuses, status)
	return nil
}
func (m *mockSessionRepo) IncrementUnreadCount(ctx context.Context, id string) error { return nil }
func (m *mockSessionRepo) DecrementUnreadCount(ctx context.Context, id string) error { return nil }
func (m *mockSessionRepo) SetUnreadCount(ctx context.Context, id string, count int) error {
	return nil
}
func (m *mockSessionRepo) WithTx(ctx context.Context, fn func(repo repository.SessionRepository) error) error {
	return fn(m)
}

// ============================================================================
// 集成测试：agent 调用工具 → SSE 事件流推送 ToolCallingEvent / ToolCalledEvent
// ============================================================================

// TestToolCallingEvents_SSEStream 集成测试
//
// 验证链路：
//
//	LLM 返回 tool_call
//	  → BaseAgent.handleToolCall 发出 ToolCallingEvent / ToolCalledEvent
//	  → PlannerReActFlow 事件通道
//	  → AgentTaskRunner 包装为 model.Event 写入 output_stream
//	  → SSE 消费端（task.GetOutput 轮询，与 handler 的 GetTaskEvents 同路径）读取
//
// 断言：
//  1. 事件流中能读到 type=tool_calling 与 type=tool_called 两条事件
//  2. 两条事件的载荷完整（function_name / tool_call_id / arguments / result）
//  3. tool_calling 先于 tool_called 到达，且都先于 done 事件
func TestToolCallingEvents_SSEStream(t *testing.T) {
	// 清理默认注册表，避免测试间相互影响
	defaultTaskRegistry.Clear()

	// 1. mock LLM：按调用顺序返回 计划 → 工具调用 → 最终结果 → 总结
	mock := &mockLLM{
		responses: []*llmcore.LLMResponse{
			// planner.CreatePlan
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"message":"已制定计划","goal":"测试工具调用","title":"工具调用集成测试","language":"zh","steps":[{"id":"s1","description":"调用测试工具"}]}`}},
			// react.BaseAgent.Invoke 第一轮：要求调用 test_tool
			{Message: llmcore.Message{
				Role:        llmcore.RoleAssistant,
				ContentText: "",
				ToolCalls: []llmcore.ToolCall{{
					ID:   "call-1",
					Type: llmcore.ToolTypeFunction,
					Function: llmcore.ToolCallFunction{
						Name:      "test_tool",
						Arguments: `{"query":"hello"}`,
					},
				}},
			}},
			// react.BaseAgent.Invoke 第二轮：工具执行后的最终结果
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"success":true,"result":"测试工具执行成功"}`}},
			// react.Summarize
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"message":"任务总结","attachments":[]}`}},
		},
	}

	// 2. 组装任务执行链路（内存版消息队列替代 Redis）
	mq := newInMemoryMessageQueue()
	mockRepo := &mockSessionRepo{}
	runner := NewAgentTaskRunner(&AgentTaskRunnerConfig{
		SessionID:   "session-tool-event-test",
		AgentConfig: DefaultAgentConfig(),
		LLM:         mock,
		Tools:       []Tool{&mockEchoTool{}},
		SessionRep:  mockRepo,
	})
	task := NewRedisStreamTask(mq, runner)
	defer task.Cancel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. 启动任务执行（后台 goroutine 从 input_stream 消费并驱动 flow）
	if err := task.Invoke(ctx); err != nil {
		t.Fatalf("task.Invoke() error = %v", err)
	}

	// 4. 模拟 SSE 消费端：与 handler 的 GetTaskEvents 相同，用 GetOutput 轮询 output_stream
	msgEvent := &model.MessageEvent{
		Type:    model.EventTypeMessage,
		Role:    "user",
		Message: "请调用测试工具",
	}
	if _, err := task.PutInput(ctx, msgEvent); err != nil {
		t.Fatalf("task.PutInput() error = %v", err)
	}

	var gotCalling, gotCalled bool
	callingSeq, calledSeq := 0, 0
	seq := 0
	startID := ""
	deadline := time.After(15 * time.Second)

	for {
		events, err := task.GetOutput(ctx, startID, 500)
		if err != nil {
			t.Fatalf("task.GetOutput() error = %v", err)
		}

		for _, ev := range events {
			if ev == nil {
				continue
			}
			seq++
			switch ev.Type {
			case model.EventTypeToolCalling:
				var calling model.ToolCallingEvent
				if err := sonic.Unmarshal(ev.Data, &calling); err != nil {
					t.Fatalf("unmarshal ToolCallingEvent error = %v", err)
				}
				if calling.FunctionName != "test_tool" {
					t.Errorf("ToolCallingEvent.FunctionName = %q, want %q", calling.FunctionName, "test_tool")
				}
				if calling.ToolCallID != "call-1" {
					t.Errorf("ToolCallingEvent.ToolCallID = %q, want %q", calling.ToolCallID, "call-1")
				}
				if v, ok := calling.Arguments["query"].(string); !ok || v != "hello" {
					t.Errorf("ToolCallingEvent.Arguments = %v, want query=hello", calling.Arguments)
				}
				if gotCalling {
					t.Error("收到重复的 ToolCallingEvent")
				}
				gotCalling = true
				callingSeq = seq

			case model.EventTypeToolCalled:
				var called model.ToolCalledEvent
				if err := sonic.Unmarshal(ev.Data, &called); err != nil {
					t.Fatalf("unmarshal ToolCalledEvent error = %v", err)
				}
				if called.FunctionName != "test_tool" {
					t.Errorf("ToolCalledEvent.FunctionName = %q, want %q", called.FunctionName, "test_tool")
				}
				if called.Result == nil || !called.Result.Success {
					t.Errorf("ToolCalledEvent.Result = %+v, want success result", called.Result)
				}
				if gotCalled {
					t.Error("收到重复的 ToolCalledEvent")
				}
				gotCalled = true
				calledSeq = seq

			case model.EventTypeDone:
				// SSE 流以 done 事件收尾：此时两个工具事件必须都已收到且顺序正确
				if !gotCalling || !gotCalled {
					t.Errorf("done 前未收到完整工具事件: calling=%v called=%v", gotCalling, gotCalled)
					return
				}
				if callingSeq > calledSeq {
					t.Errorf("ToolCallingEvent(seq=%d) 应先于 ToolCalledEvent(seq=%d) 到达", callingSeq, calledSeq)
				}
				return
			}
			startID = ev.ID
		}

		select {
		case <-deadline:
			t.Fatalf("等待工具事件超时: calling=%v called=%v, 已收到事件数=%d", gotCalling, gotCalled, seq)
		default:
		}
	}
}

// TestToolCallingEvents_SSEStream_Failure 测试工具调用失败的场景
//
// 验证链路：与成功路径相同，但工具 Invoke 返回错误结果
//
// 断言：
//  1. tool_called 事件的 result.success = false
//  2. result.message 包含错误信息
//  3. result.data = nil（失败时无数据）
func TestToolCallingEvents_SSEStream_Failure(t *testing.T) {
	defaultTaskRegistry.Clear()

	// mock LLM：返回 计划 → 工具调用 → 最终结果 → 总结
	mock := &mockLLM{
		responses: []*llmcore.LLMResponse{
			// planner.CreatePlan
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"message":"已制定计划","goal":"测试失败工具","title":"失败测试","language":"zh","steps":[{"id":"s1","description":"调用会失败的测试工具"}]}`}},
			// react.BaseAgent.Invoke：要求调用 failing_tool
			{Message: llmcore.Message{
				Role:        llmcore.RoleAssistant,
				ContentText: "",
				ToolCalls: []llmcore.ToolCall{{
					ID:   "call-fail-1",
					Type: llmcore.ToolTypeFunction,
					Function: llmcore.ToolCallFunction{
						Name:      "failing_tool",
						Arguments: `{"query":"test"}`,
					},
				}},
			}},
			// react.BaseAgent.Invoke：工具失败后的最终结果
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"success":false,"result":"工具执行失败，请重试"}`}},
			// react.Summarize
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"message":"任务失败总结","attachments":[]}`}},
		},
	}

	mq := newInMemoryMessageQueue()
	mockRepo := &mockSessionRepo{}
	runner := NewAgentTaskRunner(&AgentTaskRunnerConfig{
		SessionID:   "session-tool-event-fail-test",
		AgentConfig: DefaultAgentConfig(),
		LLM:         mock,
		Tools:       []Tool{&mockFailingTool{}},
		SessionRep:  mockRepo,
	})
	task := NewRedisStreamTask(mq, runner)
	defer task.Cancel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := task.Invoke(ctx); err != nil {
		t.Fatalf("task.Invoke() error = %v", err)
	}

	msgEvent := &model.MessageEvent{
		Type:    model.EventTypeMessage,
		Role:    "user",
		Message: "请调用会失败的测试工具",
	}
	if _, err := task.PutInput(ctx, msgEvent); err != nil {
		t.Fatalf("task.PutInput() error = %v", err)
	}

	var gotCalling, gotCalled bool
	var calledEvent model.ToolCalledEvent
	deadline := time.After(15 * time.Second)
	startID := ""

	for {
		events, err := task.GetOutput(ctx, startID, 500)
		if err != nil {
			t.Fatalf("task.GetOutput() error = %v", err)
		}

		for _, ev := range events {
			if ev == nil {
				continue
			}
			switch ev.Type {
			case model.EventTypeToolCalling:
				var calling model.ToolCallingEvent
				if err := sonic.Unmarshal(ev.Data, &calling); err != nil {
					t.Fatalf("unmarshal ToolCallingEvent error = %v", err)
				}
				if calling.FunctionName != "failing_tool" {
					t.Errorf("ToolCallingEvent.FunctionName = %q, want %q", calling.FunctionName, "failing_tool")
				}
				if calling.ToolCallID != "call-fail-1" {
					t.Errorf("ToolCallingEvent.ToolCallID = %q, want %q", calling.ToolCallID, "call-fail-1")
				}
				gotCalling = true

			case model.EventTypeToolCalled:
				var called model.ToolCalledEvent
				if err := sonic.Unmarshal(ev.Data, &called); err != nil {
					t.Fatalf("unmarshal ToolCalledEvent error = %v", err)
				}
				// 关键断言：失败结果的 success=false
				if called.Result == nil {
					t.Error("ToolCalledEvent.Result = nil, want non-nil")
				} else {
					if called.Result.Success {
						t.Error("ToolCalledEvent.Result.Success = true, want false")
					}
					// 错误信息应包含错误描述
					if called.Result.Message == "" {
						t.Error("ToolCalledEvent.Result.Message is empty, want error message")
					}
					if !strings.Contains(called.Result.Message, "参数无效") {
						t.Errorf("ToolCalledEvent.Result.Message = %q, want to contain '参数无效'", called.Result.Message)
					}
					// 失败时 data 应为 nil
					if called.Result.Data != nil {
						t.Errorf("ToolCalledEvent.Result.Data = %v, want nil for failure case", called.Result.Data)
					}
				}
				calledEvent = called
				gotCalled = true

			case model.EventTypeDone:
				if !gotCalling || !gotCalled {
					t.Errorf("done 前未收到完整工具事件: calling=%v called=%v", gotCalling, gotCalled)
					return
				}
				// 确保 FunctionName 一致
				if calledEvent.FunctionName != "failing_tool" {
					t.Errorf("calledEvent.FunctionName = %q, want %q", calledEvent.FunctionName, "failing_tool")
				}
				return
			}
			startID = ev.ID
		}

		select {
		case <-deadline:
			t.Fatalf("等待工具事件超时: calling=%v called=%v", gotCalling, gotCalled)
		default:
		}
	}
}
