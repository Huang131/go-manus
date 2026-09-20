package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

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

// connectionTestModel 构造连接测试用的模型配置（含 APIKey，满足前置校验）。
func connectionTestModel() *model.LLMModel {
	return &model.LLMModel{
		Name:      "demo",
		Provider:  "openai",
		BaseURL:   "https://example.test/v1",
		APIKey:    "test-key",
		ModelName: "demo",
	}
}

// TestLLMModelService_Test_SendsMinimalRequest 验证连接测试只发一条最小 prompt，
// 不带 tools / response_format——这是"连接测试"与"真实对话"的关键区别。
func TestLLMModelService_Test_SendsMinimalRequest(t *testing.T) {
	var gotReq *external.LLMRequest
	svc := NewLLMModelServiceWithLLMFactory(NewMockLLMModelRepository(), func(_ *external.LLMRuntimeConfig) external.LLM {
		return &connectionTestLLM{invoke: func(req *external.LLMRequest) (*llmcore.LLMResponse, error) {
			gotReq = req
			return &llmcore.LLMResponse{Message: llmcore.Message{ContentText: "连接成功"}}, nil
		}}
	})

	if _, err := svc.Test(context.Background(), connectionTestModel()); err != nil {
		t.Fatal(err)
	}
	if gotReq == nil {
		t.Fatal("Invoke() 未被调用")
	}
	if len(gotReq.Messages) != 1 {
		t.Fatalf("Messages len = %d, want 1", len(gotReq.Messages))
	}
	if gotReq.Messages[0].Role != model.RoleUser || gotReq.Messages[0].ContentText == "" {
		t.Fatalf("Messages[0] = %+v, want 非空 user 消息", gotReq.Messages[0])
	}
	if len(gotReq.Tools) != 0 {
		t.Errorf("Tools len = %d, want 0", len(gotReq.Tools))
	}
	if gotReq.ResponseFormat != nil {
		t.Errorf("ResponseFormat = %v, want nil", gotReq.ResponseFormat)
	}
}

// TestLLMModelService_Test_ReturnsResponseAndLatency 验证返回体透传内容与模型名，
// 且 LatencyMS 确实测量了调用耗时（mock 内 sleep 10ms）。
func TestLLMModelService_Test_ReturnsResponseAndLatency(t *testing.T) {
	svc := NewLLMModelServiceWithLLMFactory(NewMockLLMModelRepository(), func(_ *external.LLMRuntimeConfig) external.LLM {
		return &connectionTestLLM{invoke: func(_ *external.LLMRequest) (*llmcore.LLMResponse, error) {
			time.Sleep(10 * time.Millisecond)
			return &llmcore.LLMResponse{Message: llmcore.Message{ContentText: "连接成功"}}, nil
		}}
	})

	result, err := svc.Test(context.Background(), connectionTestModel())
	if err != nil {
		t.Fatal(err)
	}
	if result.Content != "连接成功" {
		t.Errorf("Content = %q, want 连接成功", result.Content)
	}
	if result.ModelName != "demo" {
		t.Errorf("ModelName = %q, want demo", result.ModelName)
	}
	// sleep 给出确定性下界：Milliseconds() 是截断，只会更大不会更小。
	if result.LatencyMS < 10 {
		t.Errorf("LatencyMS = %d, want >= 10", result.LatencyMS)
	}
}

func TestLLMModelService_Test_RequiresAPIKey(t *testing.T) {
	svc := NewLLMModelService(NewMockLLMModelRepository())
	m := connectionTestModel()
	m.APIKey = ""

	_, err := svc.Test(context.Background(), m)
	if !errors.Is(err, ErrModelAPIKeyRequired) {
		t.Fatalf("err = %v, want API key validation error", err)
	}
}

// TestLLMModelService_Test_MapsProviderAuthError 验证 Test 把 Invoke 的错误接到了
// mapModelTestError 上（映射表本身由 TestMapModelTestError 逐项覆盖）。
func TestLLMModelService_Test_MapsProviderAuthError(t *testing.T) {
	svc := NewLLMModelServiceWithLLMFactory(NewMockLLMModelRepository(), func(_ *external.LLMRuntimeConfig) external.LLM {
		return &connectionTestLLM{invoke: func(_ *external.LLMRequest) (*llmcore.LLMResponse, error) {
			return nil, &llmcore.ProviderError{Kind: llmcore.KindAuth, Message: "invalid api key"}
		}}
	})

	_, err := svc.Test(context.Background(), connectionTestModel())
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Status() != http.StatusUnauthorized || appErr.Msg != "模型认证失败: invalid api key" {
		t.Fatalf("err = %v, want mapped auth error", err)
	}
}

// TestMapModelTestError 逐项覆盖 ProviderError.Kind → apperr 的映射表。
// 发生变化的错误码会直接改变前端提示和用户排查方向，因此每类都要钉住。
func TestMapModelTestError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{
			name:       "鉴权失败映射 401",
			err:        &llmcore.ProviderError{Kind: llmcore.KindAuth, StatusCode: 401, Message: "invalid api key"},
			wantStatus: http.StatusUnauthorized,
			wantMsg:    "模型认证失败: invalid api key",
		},
		{
			name:       "模型不存在映射 404",
			err:        &llmcore.ProviderError{Kind: llmcore.KindNotFound, StatusCode: 404, Message: "model not found"},
			wantStatus: http.StatusNotFound,
			wantMsg:    "模型不存在: model not found",
		},
		{
			name:       "协议不兼容映射 400",
			err:        &llmcore.ProviderError{Kind: llmcore.KindBadRequest, StatusCode: 400, Message: "bad payload"},
			wantStatus: http.StatusBadRequest,
			wantMsg:    "模型请求不兼容: bad payload",
		},
		{
			name:       "限流映射 503",
			err:        &llmcore.ProviderError{Kind: llmcore.KindRateLimit, StatusCode: 429, Message: "rate limited"},
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "模型暂时不可用: rate limited",
		},
		{
			name:       "网络错误映射 503",
			err:        &llmcore.ProviderError{Kind: llmcore.KindNetwork, Message: "dial tcp: connection refused"},
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "模型暂时不可用: dial tcp: connection refused",
		},
		{
			name:       "超时映射 503",
			err:        &llmcore.ProviderError{Kind: llmcore.KindTimeout, Message: "context deadline exceeded"},
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "模型暂时不可用: context deadline exceeded",
		},
		{
			name:       "上游 5xx 映射 503",
			err:        &llmcore.ProviderError{Kind: llmcore.KindServer, StatusCode: 502, Message: "upstream bad gateway"},
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "模型暂时不可用: upstream bad gateway",
		},
		{
			name:       "内容安全拦截映射 500",
			err:        &llmcore.ProviderError{Kind: llmcore.KindContentFilter, Message: "content filtered"},
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "模型连接失败: content filtered",
		},
		{
			name:       "未知 Kind 映射 500",
			err:        &llmcore.ProviderError{Kind: llmcore.KindUnknown, Message: "weird failure"},
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "模型连接失败: weird failure",
		},
		{
			name:       "ProviderError 无消息时用兜底文案",
			err:        &llmcore.ProviderError{Kind: llmcore.KindContextLimit},
			wantStatus: http.StatusInternalServerError,
			wantMsg:    "模型连接失败: 模型连接失败",
		},
		{
			name:       "非 ProviderError 映射 503",
			err:        errors.New("boom"),
			wantStatus: http.StatusServiceUnavailable,
			wantMsg:    "模型连接失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapModelTestError(tt.err)
			var appErr *apperr.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("err = %v, want *apperr.Error", err)
			}
			if got := appErr.Status(); got != tt.wantStatus {
				t.Errorf("status = %d, want %d", got, tt.wantStatus)
			}
			if appErr.Msg != tt.wantMsg {
				t.Errorf("msg = %q, want %q", appErr.Msg, tt.wantMsg)
			}
		})
	}
}
