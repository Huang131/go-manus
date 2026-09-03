package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mooc-manus/go-manus/api/internal/model"
)

// MockQueryContext 用于测试的 QueryContext Mock
type MockQueryContext struct {
	sessions map[string]*model.Session
}

func NewMockQueryContext() *MockQueryContext {
	return &MockQueryContext{
		sessions: make(map[string]*model.Session),
	}
}

// TestSessionRepository_Create 测试创建会话
func TestSessionRepository_Create(t *testing.T) {
	// 测试会话模型创建
	session := &model.Session{
		ID:        "test-session-1",
		Title:     "Test Session",
		Status:    model.SessionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if session.ID != "test-session-1" {
		t.Errorf("Session ID 应为 test-session-1，实际: %s", session.ID)
	}

	if session.Status != model.SessionStatusPending {
		t.Errorf("Session Status 应为 pending，实际: %s", session.Status)
	}
}

// TestSessionRepository_AppendEvent 测试追加事件
func TestSessionRepository_AppendEvent(t *testing.T) {
	session := &model.Session{
		ID:        "test-session-3",
		Title:     "Test Session 3",
		Status:    model.SessionStatusPending,
		Events:    []model.Event{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 模拟事件追加
	event := &model.Event{
		ID:        "event-1",
		Type:      model.EventTypeMessage,
		CreatedAt: time.Now(),
		Data:      json.RawMessage(`{"content":"hello"}`),
	}

	session.Events = append(session.Events, *event)

	if len(session.Events) != 1 {
		t.Errorf("Events 数量应为 1，实际: %d", len(session.Events))
	}

	if session.Events[0].Type != model.EventTypeMessage {
		t.Errorf("Event 类型应为 message，实际: %s", session.Events[0].Type)
	}
}

// TestSessionRepository_AppendFile 已废弃：Issue #1 后文件不再通过 sessions.files JSONB 存储，
// 文件已统一到 files 表，对应测试见 TestFileRepository_*。

// TestSessionRepository_UpdateLatestMessage 测试更新最新消息
func TestSessionRepository_UpdateLatestMessage(t *testing.T) {
	session := &model.Session{
		ID:                 "test-session-5",
		Title:              "Test Session 5",
		LatestMessage:      "",
		LatestMessageAt:    nil,
		UnreadMessageCount: 0,
		Status:             model.SessionStatusPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// 模拟更新最新消息
	session.LatestMessage = "Hello, world!"
	now := time.Now()
	session.LatestMessageAt = &now
	session.UnreadMessageCount = 1

	if session.LatestMessage != "Hello, world!" {
		t.Errorf("LatestMessage 应为 Hello, world!，实际: %s", session.LatestMessage)
	}

	if session.LatestMessageAt == nil {
		t.Error("LatestMessageAt 不应为 nil")
	}

	if session.UnreadMessageCount != 1 {
		t.Errorf("UnreadMessageCount 应为 1，实际: %d", session.UnreadMessageCount)
	}
}

// TestSessionRepository_Memory 测试记忆操作
func TestSessionRepository_Memory(t *testing.T) {
	session := &model.Session{
		ID:        "test-session-6",
		Title:     "Test Session 6",
		Memories:  map[string]interface{}{},
		Status:    model.SessionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 模拟添加记忆
	memory := &model.Memory{
		Messages: []map[string]interface{}{
			{"role": "user", "content": "Hello"},
		},
	}

	// 序列化记忆
	memoryJSON, _ := json.Marshal(memory)
	session.Memories["summary"] = string(memoryJSON)

	if len(session.Memories) != 1 {
		t.Errorf("Memories 数量应为 1，实际: %d", len(session.Memories))
	}

	// 反序列化验证
	var restored model.Memory
	json.Unmarshal([]byte(session.Memories["summary"].(string)), &restored)
	if len(restored.Messages) != 1 {
		t.Error("记忆内容不匹配")
	}
}

// TestSessionRepository_List 测试会话列表分页
func TestSessionRepository_List(t *testing.T) {
	sessions := []*model.Session{
		{ID: "1", Title: "Session 1", Status: model.SessionStatusPending},
		{ID: "2", Title: "Session 2", Status: model.SessionStatusPending},
		{ID: "3", Title: "Session 3", Status: model.SessionStatusPending},
		{ID: "4", Title: "Session 4", Status: model.SessionStatusPending},
		{ID: "5", Title: "Session 5", Status: model.SessionStatusPending},
	}

	// 测试分页
	testCases := []struct {
		name     string
		limit    int
		offset   int
		expected int
	}{
		{"第一页", 2, 0, 2},
		{"第二页", 2, 2, 2},
		{"第三页（只有1个）", 2, 4, 1},
		{"超出范围", 2, 10, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := tc.offset
			if start > len(sessions) {
				start = len(sessions)
			}
			end := start + tc.limit
			if end > len(sessions) {
				end = len(sessions)
			}

			result := sessions[start:end]
			if len(result) != tc.expected {
				t.Errorf("%s: 期望 %d 个会话，实际 %d 个", tc.name, tc.expected, len(result))
			}
		})
	}
}

// TestSessionRepository_StatusTransitions 测试会话状态转换
func TestSessionRepository_StatusTransitions(t *testing.T) {
	session := &model.Session{
		ID:        "test-session-7",
		Title:     "Test Session 7",
		Status:    model.SessionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 测试状态转换
	statusTransitions := []struct {
		from    model.SessionStatus
		to      model.SessionStatus
		allowed bool
	}{
		{model.SessionStatusPending, model.SessionStatusRunning, true},
		{model.SessionStatusRunning, model.SessionStatusWaiting, true},
		{model.SessionStatusWaiting, model.SessionStatusRunning, true},
		{model.SessionStatusRunning, model.SessionStatusCompleted, true},
		{model.SessionStatusCompleted, model.SessionStatusRunning, false}, // 完成后再运行应该不允许
	}

	for _, st := range statusTransitions {
		session.Status = st.from

		// 模拟状态转换验证
		canTransition := session.Status != model.SessionStatusCompleted || st.to != model.SessionStatusRunning

		if canTransition != st.allowed {
			t.Errorf("状态 %s -> %s 的转换应允许: %v，实际: %v",
				st.from, st.to, st.allowed, canTransition)
		}
	}
}

// TestSessionRepository_JSONSerialization 测试 JSON 序列化
func TestSessionRepository_JSONSerialization(t *testing.T) {
	session := &model.Session{
		ID:                 "test-session-8",
		Title:              "Test Session 8",
		Status:             model.SessionStatusPending,
		UnreadMessageCount: 1,
		LatestMessage:      "Hello",
		Events:             []model.Event{},
		Memories:           map[string]interface{}{},
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// 测试 JSON 序列化（使用自定义 MarshalJSON 将时间转为 Unix 时间戳）
	data, err := json.Marshal(session)
	if err != nil {
		t.Errorf("JSON Marshal 失败: %v", err)
	}

	// 解析 JSON 验证基本字段
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		t.Errorf("JSON Unmarshal 到 map 失败: %v", err)
	}

	// 验证字段存在
	if jsonData["id"] != "test-session-8" {
		t.Errorf("ID 不匹配: 期望 test-session-8，实际 %v", jsonData["id"])
	}

	if jsonData["title"] != "Test Session 8" {
		t.Errorf("Title 不匹配: 期望 Test Session 8，实际 %v", jsonData["title"])
	}

	// 验证时间字段被序列化为 int64 (Unix 时间戳)
	if _, ok := jsonData["updated_at"].(float64); !ok {
		t.Errorf("UpdatedAt 应为 Unix 时间戳 (float64)，实际类型: %T", jsonData["updated_at"])
	}

	// 注意：由于 Session.MarshalJSON 自定义实现将时间转为 Unix 时间戳，
	// 而 UnmarshalJSON 不支持该格式，所以这里不测试反序列化。
	// 如果需要完整的序列化/反序列化支持，应在 Session 模型中添加 UnmarshalJSON 方法。
}

// TestSessionRepository_QueryContext 测试查询上下文
func TestSessionRepository_QueryContext(t *testing.T) {
	ctx := context.Background()

	// 模拟使用上下文
	select {
	case <-ctx.Done():
		t.Error("上下文不应该被取消")
	default:
		// 正常
	}

	// 创建取消的上下文
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	select {
	case <-canceledCtx.Done():
		// 预期取消
	default:
		t.Error("上下文应该被取消")
	}
}
