package external

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// OpenAIClient OpenAI 兼容 API 客户端
type OpenAIClient struct {
	baseURL         string
	apiKey          string
	modelName       string
	temperature     float64
	maxTokens       int
	toolCallTimeout time.Duration // tool calling 请求超时
	httpClient      *http.Client
}

// OpenAIClientConfig OpenAI 客户端配置
type OpenAIClientConfig struct {
	BaseURL         string  `mapstructure:"base_url"`
	APIKey          string  `mapstructure:"api_key"`
	ModelName       string  `mapstructure:"model_name"`
	Temperature     float64 `mapstructure:"temperature"`
	MaxTokens       int     `mapstructure:"max_tokens"`
	ToolCallTimeout int     `mapstructure:"tool_call_timeout"` // tool calling 请求超时秒数，默认 15
}

func (c *OpenAIClientConfig) setDefaults() {
	if c.ToolCallTimeout == 0 {
		c.ToolCallTimeout = 15
	}
}

// NewOpenAIClient 创建 OpenAI 客户端
func NewOpenAIClient(cfg *OpenAIClientConfig) *OpenAIClient {
	cfg.setDefaults()
	return &OpenAIClient{
		baseURL:         cfg.BaseURL,
		apiKey:          cfg.APIKey,
		modelName:       cfg.ModelName,
		temperature:     cfg.Temperature,
		maxTokens:       cfg.MaxTokens,
		toolCallTimeout: time.Duration(cfg.ToolCallTimeout) * time.Second,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// chatRequest OpenAI Chat API 请求
type chatRequest struct {
	Model          string                   `json:"model"`
	Messages       []map[string]interface{} `json:"messages"`
	Tools          []map[string]interface{} `json:"tools,omitempty"`
	ToolChoice     string                   `json:"tool_choice,omitempty"`
	Temperature    float64                  `json:"temperature"`
	MaxTokens      int                      `json:"max_tokens"`
	ResponseFormat map[string]interface{}   `json:"response_format,omitempty"` // 例如 {"type": "json_object"}
}

// chatResponse OpenAI Chat API 响应
type chatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string `json:"content"`
			Reasoning string `json:"reasoning_content"`
			ToolCalls []struct {
				ID       string                 `json:"id"`
				Type     string                 `json:"type"`
				Function map[string]interface{} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Invoke 调用 OpenAI Chat API
func (c *OpenAIClient) Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// 构建请求
	chatReq := chatRequest{
		Model:          c.modelName,
		Messages:       req.Messages,
		Tools:          req.Tools,
		ToolChoice:     req.ToolChoice,
		Temperature:    c.temperature,
		MaxTokens:      c.maxTokens,
		ResponseFormat: req.ResponseFormat,
	}

	// 序列化请求
	reqBody, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 构建 HTTP 请求上下文：tool calling 请求使用独立超时（默认 15s），
	// 不依赖硬编码模型黑名单——任何不支持 tool calling 的模型都会在此时超时失败，
	// 而不是等到全局 120s 才暴露。
	httpCtx := ctx
	if len(req.Tools) > 0 {
		var cancel context.CancelFunc
		httpCtx, cancel = context.WithTimeout(ctx, c.toolCallTimeout)
		defer cancel()
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(httpCtx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// context 超时（tool calling）转为友好错误，不暴露底层细节
		if len(req.Tools) > 0 && errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("tool calling 请求超时（%v），模型 %q 可能不支持 tool calling", c.toolCallTimeout, c.modelName)
		}
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("OpenAI API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("OpenAI API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// 转换响应
	content := chatResp.Choices[0].Message.Content
	reasoning := chatResp.Choices[0].Message.Reasoning
	// 推理模型（如 MiniMax-M2.7）可能把输出放在 reasoning_content，content 为空。
	// 此时用 reasoning_content 兜底，否则下游 planner JSON 解析会拿到空串报错。
	if content == "" && chatResp.Choices[0].Message.Reasoning != "" {
		logger.Info("OpenAI API content 为空，使用 reasoning_content 兜底",
			zap.Int("reasoning_len", len(chatResp.Choices[0].Message.Reasoning)))
		content = chatResp.Choices[0].Message.Reasoning
	}
	result := &LLMResponse{
		ID:               chatResp.ID,
		Content:          content,
		ReasoningContent: reasoning,
		RawContent:       chatResp.Choices[0].Message.Content,
	}

	// 转换工具调用
	if len(chatResp.Choices[0].Message.ToolCalls) > 0 {
		result.ToolUse = make([]map[string]interface{}, len(chatResp.Choices[0].Message.ToolCalls))
		for i, tc := range chatResp.Choices[0].Message.ToolCalls {
			result.ToolUse[i] = map[string]interface{}{
				"id":       tc.ID,
				"type":     tc.Type,
				"function": tc.Function,
			}
		}
	}

	return NormalizeLLMResponse(result), nil
}

// ModelName 返回模型名称
func (c *OpenAIClient) ModelName() string {
	return c.modelName
}

// Temperature 返回温度参数
func (c *OpenAIClient) Temperature() float64 {
	return c.temperature
}

// MaxTokens 返回最大 token 数
func (c *OpenAIClient) MaxTokens() int {
	return c.maxTokens
}
