package llm

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// TestLLMInterface 测试 LLM 接口定义
func TestLLMInterface(t *testing.T) {
	// 验证 LLM 接口存在
	var _ LLM = (*MockLLM)(nil)
}

// MockLLM 用于测试的 LLM Mock 实现
type MockLLM struct {
	invokeResult string
	invokeErr    error
}

func (m *MockLLM) Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
	if m.invokeErr != nil {
		return nil, m.invokeErr
	}
	return &llmcore.LLMResponse{
		ID: "mock-response",
		Message: llmcore.Message{
			Role:        model.RoleAssistant,
			ContentText: m.invokeResult,
		},
	}, nil
}

func (m *MockLLM) ModelName() string {
	return "mock-model"
}

func (m *MockLLM) Temperature() float64 {
	return 0.7
}

func (m *MockLLM) MaxTokens() int {
	return 4096
}

// TestLLM_Invoke 测试 LLM 调用
func TestLLM_Invoke(t *testing.T) {
	llm := &MockLLM{
		invokeResult: "This is a mock response",
	}

	req := &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "Hello, how are you?"},
		},
	}

	result, err := llm.Invoke(context.Background(), req)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if result == nil {
		t.Error("Invoke() should return a result")
	}

	if result.Message.ContentText != "This is a mock response" {
		t.Errorf("Invoke() Content = %s, want This is a mock response", result.Message.ContentText)
	}
}

// TestLLM_Invoke_WithError 测试带错误的 LLM 调用
func TestLLM_Invoke_WithError(t *testing.T) {
	llm := &MockLLM{
		invokeErr: context.DeadlineExceeded,
	}

	req := &LLMRequest{
		Messages: []llmcore.Message{
			{Role: model.RoleUser, ContentText: "Hello"},
		},
	}

	result, err := llm.Invoke(context.Background(), req)
	if err == nil {
		t.Error("Invoke() should return an error")
	}

	if result != nil {
		t.Error("Invoke() with error should return nil result")
	}
}

// TestLLM_ModelInfo 测试 LLM 模型信息
func TestLLM_ModelInfo(t *testing.T) {
	llm := &MockLLM{}

	if llm.ModelName() != "mock-model" {
		t.Errorf("ModelName() = %s, want mock-model", llm.ModelName())
	}

	if llm.Temperature() != 0.7 {
		t.Errorf("Temperature() = %f, want 0.7", llm.Temperature())
	}

	if llm.MaxTokens() != 4096 {
		t.Errorf("MaxTokens() = %d, want 4096", llm.MaxTokens())
	}
}
