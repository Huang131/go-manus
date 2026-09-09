package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
)

// mockLLM 用于测试的 LLM mock。
// 按 sequence 返回预置响应，索引越界返回 errEmptyLLM。
//
// 阶段 1d 改造点：req.Messages / req.Tools 改 llmcore 强类型。
// 深拷贝用 slice 内置 copy，不再做 map-by-map 拷贝。
type mockLLM struct {
	responses []*llmcore.LLMResponse
	errs      []error
	calls     []*external.LLMRequest
}

func (m *mockLLM) Invoke(ctx context.Context, req *external.LLMRequest) (*llmcore.LLMResponse, error) {
	// 深拷贝请求以避免后续 mutation 干扰断言
	dup := &external.LLMRequest{
		Messages: append([]llmcore.Message{}, req.Messages...),
	}
	if req.Tools != nil {
		dup.Tools = append([]llmcore.ToolSpec{}, req.Tools...)
	}
	if req.ResponseFormat != nil {
		dup.ResponseFormat = req.ResponseFormat
	}
	if req.ToolChoice != "" {
		dup.ToolChoice = req.ToolChoice
	}
	m.calls = append(m.calls, dup)

	idx := len(m.calls) - 1
	if idx < len(m.errs) && m.errs[idx] != nil {
		return nil, m.errs[idx]
	}
	if idx < len(m.responses) && m.responses[idx] != nil {
		return m.responses[idx], nil
	}
	return nil, errors.New("mockLLM: no more responses")
}

func (m *mockLLM) ModelName() string    { return "mock" }
func (m *mockLLM) Temperature() float64 { return 0 }
func (m *mockLLM) MaxTokens() int       { return 0 }

// TestBaseAgent_InvokeWithEmptyRetry 验证：LLM 首次返回空 content 后，
// BaseAgent.invokeWithEmptyRetry 会注入 "AI 无响应内容，请继续。" 并重试，最终得到有效响应。
func TestBaseAgent_InvokeWithEmptyRetry(t *testing.T) {
	mock := &mockLLM{
		responses: []*llmcore.LLMResponse{
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""}},                  // 第 1 次：空内容
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""}},                  // 第 2 次：仍然空
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"hello":"world"}`}}, // 第 3 次：成功
		},
	}

	agent := NewBaseAgent("test", "session-1", DefaultAgentConfig(), mock, nil)

	resp, attempts, err := agent.invokeWithEmptyRetry(context.Background(), &external.LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
	}, 3)
	if err != nil {
		t.Fatalf("invokeWithEmptyRetry() error = %v", err)
	}
	if resp.Message.ContentText != `{"hello":"world"}` {
		t.Errorf("resp.Message.ContentText = %q, want non-empty JSON", resp.Message.ContentText)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if len(mock.calls) != 3 {
		t.Fatalf("LLM calls = %d, want 3", len(mock.calls))
	}

	// 校验：第二次调用时，messages 已注入 "AI 无响应内容，请继续。"
	second := mock.calls[1]
	found := false
	for _, msg := range second.Messages {
		if msg.Role == llmcore.RoleUser && msg.ContentText == "AI 无响应内容，请继续。" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected retry message 'AI 无响应内容，请继续。' to be injected on 2nd call")
	}
}

// TestBaseAgent_InvokeWithEmptyRetry_AllEmpty 验证：超过 maxRetries 后返回错误。
func TestBaseAgent_InvokeWithEmptyRetry_AllEmpty(t *testing.T) {
	mock := &mockLLM{
		responses: []*llmcore.LLMResponse{
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""}},
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: ""}},
		},
	}
	agent := NewBaseAgent("test", "session-1", DefaultAgentConfig(), mock, nil)

	_, _, err := agent.invokeWithEmptyRetry(context.Background(), &external.LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
	}, 2)
	if err == nil {
		t.Error("expected error when all responses are empty")
	}
}

// TestBaseAgent_InvokeWithEmptyRetry_FirstSuccess 验证：首次返回有效内容时不重试。
func TestBaseAgent_InvokeWithEmptyRetry_FirstSuccess(t *testing.T) {
	mock := &mockLLM{
		responses: []*llmcore.LLMResponse{
			{Message: llmcore.Message{Role: llmcore.RoleAssistant, ContentText: `{"ok":true}`}},
		},
	}
	agent := NewBaseAgent("test", "session-1", DefaultAgentConfig(), mock, nil)

	resp, attempts, err := agent.invokeWithEmptyRetry(context.Background(), &external.LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hi"}},
	}, 3)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if resp.Message.ContentText != `{"ok":true}` {
		t.Errorf("resp.Message.ContentText = %q", resp.Message.ContentText)
	}
}
