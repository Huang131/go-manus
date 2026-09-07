package external

import (
	"context"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

type mockRuntimeHealthStore struct {
	updates map[string]modelRuntimeHealthSnapshot
}

type modelRuntimeHealthSnapshot struct {
	status           string
	recentFailures   int
	averageLatencyMS int
}

func newMockRuntimeHealthStore() *mockRuntimeHealthStore {
	return &mockRuntimeHealthStore{updates: make(map[string]modelRuntimeHealthSnapshot)}
}

func (m *mockRuntimeHealthStore) UpdateRuntimeHealth(ctx context.Context, id string, health model.RuntimeHealth) error {
	m.updates[id] = modelRuntimeHealthSnapshot{
		status:           health.Status,
		recentFailures:   health.RecentFailures,
		averageLatencyMS: health.AverageLatencyMS,
	}
	return nil
}

func TestRoutedLLM_FallbackOnRateLimit(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "primary",
				},
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					if cfg.ModelName == "primary" {
						return nil, llmcore.NewProviderError(llmcore.KindRateLimit, "openai_compat", cfg.ModelName, "rate limit")
					}
					return &LLMResponse{Content: cfg.ModelName}, nil
				},
			}
		},
	)

	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "backup" {
		t.Fatalf("content = %q, want backup", resp.Content)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %v, want 2 attempts", attempts)
	}
}

func TestRoutedLLM_PreferHealthyCandidate(t *testing.T) {
	var gotModel string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "degraded",
					Health: LLMRuntimeHealth{
						Status:           LLMHealthDegraded,
						RecentFailures:   3,
						AverageLatencyMS: 1200,
					},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "healthy",
					Health: LLMRuntimeHealth{
						Status:           LLMHealthHealthy,
						RecentFailures:   0,
						AverageLatencyMS: 300,
					},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			gotModel = cfg.ModelName
			return &stubLLM{name: cfg.ModelName}
		},
	)

	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if gotModel != "healthy" {
		t.Fatalf("got model %s, want healthy", gotModel)
	}
	if resp.Content != "healthy" {
		t.Fatalf("content = %q, want healthy", resp.Content)
	}
}

func TestRoutedLLM_DoNotFallbackAcrossProtocol(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolAnthropic,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "claude",
				},
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindRateLimit, "openai_compat", cfg.ModelName, "rate limit")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindRateLimit) {
		t.Fatalf("kind = %v, want rate limit", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("attempts = %v, want 1 attempt", attempts)
	}
}

func TestRoutedLLM_DoNotFallbackAfterToolUseSideEffect(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "primary",
				},
				{
					Profile: llmcore.ModelProfile{
						Protocol: llmcore.ProtocolOpenAICompat,
						Capabilities: llmcore.ModelCapabilities{
							SupportsText:      true,
							SupportsToolCalls: true,
							SupportsStreaming: true,
							MaxContextTokens:  4096,
							MaxOutputTokens:   1024,
						},
					},
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{Role: llmcore.RoleUser, ContentText: "hello"},
			{Role: llmcore.RoleTool, Name: "shell", ContentText: "result"},
		},
		Tools: []llmcore.ToolSpec{
			{
				Type: "function",
				Function: llmcore.ToolSpecFunction{
					Name:        "shell",
					Description: "shell",
					Parameters:  map[string]interface{}{"type": "object"},
				},
				ReadOnly: false,
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !llmcore.IsKind(err, llmcore.KindServer) {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("attempts = %v, want 1 attempt", attempts)
	}
}

func TestRoutedLLM_AllowFallbackBeforeToolExecution(t *testing.T) {
	var attempts []string
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "primary",
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "backup",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					attempts = append(attempts, cfg.ModelName)
					return nil, llmcore.NewProviderError(llmcore.KindServer, "openai_compat", cfg.ModelName, "server error")
				},
			}
		},
	)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{
			{
				Role:      llmcore.RoleAssistant,
				ToolCalls: []llmcore.ToolCall{{ID: "tool-1", Type: "function", Function: llmcore.ToolCallFunction{Name: "shell", Arguments: "{}"}}},
			},
		},
		Tools: []llmcore.ToolSpec{{
			Type: "function",
			Function: llmcore.ToolSpecFunction{
				Name:        "shell",
				Description: "shell",
				Parameters:  map[string]interface{}{"type": "object"},
			},
			ReadOnly: false,
		}},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %v, want 2 attempts", attempts)
	}
}

func TestRoutedLLM_RecordSuccessUpdatesRuntimeHealth(t *testing.T) {
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "slow",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 1000},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "fast",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 1000},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{name: cfg.ModelName}
		},
	)

	router.RecordSuccess("fast", 10*time.Millisecond)
	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "fast" {
		t.Fatalf("content = %q, want fast", resp.Content)
	}
}

func TestRoutedLLM_RecordFailureUpdatesRuntimeHealth(t *testing.T) {
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile:   openAITextProfile(),
					ModelName: "primary",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 100},
				},
				{
					Profile:   openAITextProfile(),
					ModelName: "backup",
					Health:    LLMRuntimeHealth{Status: LLMHealthHealthy, AverageLatencyMS: 100},
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{name: cfg.ModelName}
		},
	)

	router.RecordFailure("primary", llmcore.NewProviderError(llmcore.KindServer, "openai_compat", "primary", "server error"), 20*time.Millisecond)
	resp, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if resp.Content != "backup" {
		t.Fatalf("content = %q, want backup", resp.Content)
	}
}

func TestRoutedLLM_PersistRuntimeHealth(t *testing.T) {
	store := newMockRuntimeHealthStore()
	router := NewRoutedLLM(
		func(ctx context.Context) ([]*LLMRuntimeConfig, error) {
			return []*LLMRuntimeConfig{
				{
					Profile: llmcore.ModelProfile{
						ID:           "model-1",
						Protocol:     llmcore.ProtocolOpenAICompat,
						Capabilities: openAITextProfile().Capabilities,
					},
					ModelName: "model-1",
				},
			}, nil
		},
		nil,
		func(cfg *LLMRuntimeConfig) LLM {
			return &stubLLM{
				name: cfg.ModelName,
				invoke: func(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
					return &LLMResponse{Content: "ok"}, nil
				},
			}
		},
	)
	router.SetHealthStore(store)

	_, err := router.Invoke(context.Background(), &LLMRequest{
		Messages: []llmcore.Message{{Role: llmcore.RoleUser, ContentText: "hello"}},
	})
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	got, ok := store.updates["model-1"]
	if !ok {
		t.Fatal("expected runtime health to be persisted")
	}
	if got.status != LLMHealthHealthy {
		t.Fatalf("status = %s, want healthy", got.status)
	}
	if got.averageLatencyMS <= 0 {
		t.Fatalf("average latency = %d, want positive", got.averageLatencyMS)
	}
}

func openAITextProfile() llmcore.ModelProfile {
	return llmcore.ModelProfile{
		Protocol: llmcore.ProtocolOpenAICompat,
		Capabilities: llmcore.ModelCapabilities{
			SupportsText:      true,
			SupportsToolCalls: true,
			SupportsStreaming: true,
			MaxContextTokens:  4096,
			MaxOutputTokens:   1024,
		},
	}
}
