package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// AnthropicClient Anthropic API 客户端
type AnthropicClient struct {
	baseURL     string
	apiKey      string
	modelName   string
	temperature float64
	maxTokens   int
	httpClient  *http.Client
	version     string // API 版本
}

// AnthropicClientConfig Anthropic 客户端配置
type AnthropicClientConfig struct {
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	ModelName   string  `mapstructure:"model_name"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Version     string  `mapstructure:"version"` // API 版本，默认 "2023-06-01"
}

// AnthropicRequest Anthropic API 请求
type AnthropicRequest struct {
	Model       string                   `json:"model"`
	Messages    []AnthropicMessage       `json:"messages"`
	System      string                   `json:"system,omitempty"`
	Tools       []map[string]interface{} `json:"tools,omitempty"`
	MaxTokens   int                      `json:"max_tokens"`
	Temperature float64                  `json:"temperature,omitempty"`
}

// AnthropicMessage Anthropic 消息格式
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse Anthropic API 响应
type AnthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []AnthropicContent `json:"content"`
	StopReason   string             `json:"stop_reason"`
	StopSequence string             `json:"stop_sequence"`
	Usage        AnthropicUsage     `json:"usage"`
}

// AnthropicContent Anthropic 内容块
type AnthropicContent struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	InputJSON map[string]interface{} `json:"input,omitempty"`
}

// AnthropicUsage Anthropic 使用量
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewAnthropicClient 创建 Anthropic 客户端
func NewAnthropicClient(cfg *AnthropicClientConfig) *AnthropicClient {
	version := cfg.Version
	if version == "" {
		version = "2023-06-01"
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	return &AnthropicClient{
		baseURL:     baseURL,
		apiKey:      cfg.APIKey,
		modelName:   cfg.ModelName,
		temperature: cfg.Temperature,
		maxTokens:   cfg.MaxTokens,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		version: version,
	}
}

// Invoke 调用 Anthropic API
func (c *AnthropicClient) Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// 转换消息格式
	messages := make([]AnthropicMessage, 0, len(req.Messages))
	var systemMessage string

	for _, msg := range req.Messages {
		role := msg["role"].(string)
		content := ""
		if text, ok := msg["content"].(string); ok {
			content = text
		} else if contentMap, ok := msg["content"].([]interface{}); ok {
			// 处理复杂内容格式
			for _, item := range contentMap {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if text, ok := itemMap["text"].(string); ok {
						content += text
					}
				}
			}
		}

		if role == "system" {
			systemMessage += content + "\n"
		} else {
			messages = append(messages, AnthropicMessage{
				Role:    role,
				Content: content,
			})
		}
	}

	// 构建请求
	anthropicReq := AnthropicRequest{
		Model:       c.modelName,
		Messages:    messages,
		System:      systemMessage,
		Tools:       req.Tools,
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	}

	// 序列化请求
	reqBody, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := fmt.Sprintf("%s/v1/messages", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", c.version)

	// 发送请求
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("Anthropic API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("Anthropic API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 转换响应
	result := &LLMResponse{
		ID: anthropicResp.ID,
	}

	// 解析内容
	for _, content := range anthropicResp.Content {
		switch content.Type {
		case "text":
			result.Content += content.Text
		case "tool_use":
			// Anthropic 的工具调用
			toolUse := map[string]interface{}{
				"id":    content.ID,
				"name":  content.Name,
				"input": content.InputJSON, // 已经是 map[string]interface{} 类型
			}
			result.ToolUse = append(result.ToolUse, toolUse)
		}
	}

	return result, nil
}

// ModelName 返回模型名称
func (c *AnthropicClient) ModelName() string {
	return c.modelName
}

// Temperature 返回温度参数
func (c *AnthropicClient) Temperature() float64 {
	return c.temperature
}

// MaxTokens 返回最大 token 数
func (c *AnthropicClient) MaxTokens() int {
	return c.maxTokens
}

// IsAnthropicModel 判断是否为 Anthropic 模型
func IsAnthropicModel(modelName string) bool {
	anthropicModels := []string{
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
		"claude-2",
		"claude-instant",
	}
	for _, model := range anthropicModels {
		if len(modelName) >= len(model) && modelName[:len(model)] == model {
			return true
		}
	}
	return false
}
