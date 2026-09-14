package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

type connectionTestLLM struct {
	invoke func(*external.LLMRequest) (*llmcore.LLMResponse, error)
}

func (c *connectionTestLLM) Invoke(_ context.Context, req *external.LLMRequest) (*llmcore.LLMResponse, error) {
	return c.invoke(req)
}
func (c *connectionTestLLM) ModelName() string    { return "demo" }
func (c *connectionTestLLM) Temperature() float64 { return 0.7 }
func (c *connectionTestLLM) MaxTokens() int       { return 256 }

func TestLLMModelService_Test_SendsMinimalRequestAndReturnsLatency(t *testing.T) {
	svc := NewLLMModelServiceWithLLMFactory(NewMockLLMModelRepository(), func(_ *external.LLMRuntimeConfig) external.LLM {
		return &connectionTestLLM{invoke: func(req *external.LLMRequest) (*llmcore.LLMResponse, error) {
			if len(req.Messages) != 1 || req.Messages[0].Role != llmcore.RoleUser || req.Messages[0].ContentText == "" {
				t.Errorf("unexpected connection request: %+v", req)
			}
			if len(req.Tools) != 0 || req.ResponseFormat != nil {
				t.Error("connection test must not send tools or response format")
			}
			return &llmcore.LLMResponse{Message: llmcore.Message{ContentText: "连接成功"}}, nil
		}}
	})
	result, err := svc.Test(context.Background(), &model.LLMModel{
		Name: "demo", Provider: "openai", BaseURL: "https://example.test/v1", APIKey: "test-key", ModelName: "demo",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "连接成功" || result.ModelName != "demo" || result.LatencyMS < 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestLLMModelService_Test_RequiresAPIKey(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())
	_, err := svc.Test(context.Background(), &model.LLMModel{Name: "demo", Provider: "openai", BaseURL: "https://example.test/v1", ModelName: "demo"})
	if !errors.Is(err, ErrModelAPIKeyRequired) {
		t.Fatalf("err = %v, want API key validation error", err)
	}
}

func TestLLMModelService_Test_MapsProviderAuthError(t *testing.T) {
	svc := NewLLMModelServiceWithLLMFactory(NewMockLLMModelRepository(), func(_ *external.LLMRuntimeConfig) external.LLM {
		return &connectionTestLLM{invoke: func(_ *external.LLMRequest) (*llmcore.LLMResponse, error) {
			return nil, &llmcore.ProviderError{Kind: llmcore.KindAuth, Message: "invalid api key"}
		}}
	})
	_, err := svc.Test(context.Background(), &model.LLMModel{
		Name: "demo", Provider: "openai", BaseURL: "https://example.test/v1", APIKey: "bad-key", ModelName: "demo",
	})
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Status() != 401 || appErr.Msg != "模型认证失败: invalid api key" {
		t.Fatalf("err = %v, want mapped auth error", err)
	}
}
