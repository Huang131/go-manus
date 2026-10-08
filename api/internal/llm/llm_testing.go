package llm

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

// stubLLM 通用测试桩，支持自定义 Invoke 行为。
type stubLLM struct {
	name   string
	invoke func(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error)
}

func (s *stubLLM) Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
	if s.invoke != nil {
		return s.invoke(ctx, req)
	}
	return &llmcore.LLMResponse{
		Message: llmcore.Message{Role: model.RoleAssistant, ContentText: s.name},
	}, nil
}

func (s *stubLLM) ModelName() string    { return s.name }
func (s *stubLLM) Temperature() float64 { return 0.7 }
func (s *stubLLM) MaxTokens() int       { return 1024 }

// streamingStubLLM 支持流式能力的测试桩。
type streamingStubLLM struct {
	*stubLLM
	deltas []llmcore.LLMDelta
}

func (s *streamingStubLLM) Stream(context.Context, *LLMRequest) (<-chan llmcore.LLMDelta, error) {
	ch := make(chan llmcore.LLMDelta, len(s.deltas))
	for _, delta := range s.deltas {
		ch <- delta
	}
	close(ch)
	return ch, nil
}

// openAITextProfile 标准 OpenAI 兼容文本模型 profile，用于测试。
func openAITextProfile() llmcore.ModelProfile {
	return llmcore.ModelProfile{
		Protocol: llmcore.ProtocolOpenAICompat,
		Capabilities: model.ModelCapabilities{
			SupportsText:      true,
			SupportsToolCalls: true,
			SupportsStreaming: true,
			MaxContextTokens:  4096,
			MaxOutputTokens:   1024,
		},
	}
}
