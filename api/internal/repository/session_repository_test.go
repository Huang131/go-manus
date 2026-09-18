package repository

import (
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestMarshalSessionEventsRejectsInvalidJSON(t *testing.T) {
	events := []model.Event{{
		Type: model.EventTypeMessage,
		Data: []byte(`{"message":`),
	}}

	_, err := marshalSessionEvents(events)
	if err == nil {
		t.Fatal("marshalSessionEvents() error = nil, want invalid JSON error")
	}
	if !strings.Contains(err.Error(), "encode session events") {
		t.Fatalf("marshalSessionEvents() error = %q, want context", err)
	}
}

func TestExtractSessionMessageProjectsCompletedAssistantMessage(t *testing.T) {
	event := &model.Event{
		Type: model.EventTypeMessageDone,
		Data: []byte(`{"message_id":"message-1","content":"final answer"}`),
	}

	message, assistant := extractSessionMessage(event)
	if message != "final answer" {
		t.Fatalf("extractSessionMessage() message = %q, want %q", message, "final answer")
	}
	if !assistant {
		t.Fatal("extractSessionMessage() assistant = false, want true")
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
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// 测试 JSON 序列化（使用自定义 MarshalJSON 将时间转为 Unix 时间戳）
	data, err := sonic.Marshal(session)
	if err != nil {
		t.Errorf("JSON Marshal 失败: %v", err)
	}

	// 解析 JSON 验证基本字段
	var jsonData map[string]interface{}
	err = sonic.Unmarshal(data, &jsonData)
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
