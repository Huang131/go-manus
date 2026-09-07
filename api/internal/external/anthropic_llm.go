package external

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"time"

	"github.com/mooc-manus/go-manus/api/internal/llmcore"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
)

// AnthropicClient Anthropic API 客户端
type AnthropicClient struct {
	baseURL       string
	apiKey        string
	modelName     string
	temperature   float64
	maxTokens     int
	requestPolicy llmcore.RequestPolicy
	costPolicy    llmcore.CostPolicy
	httpClient    *http.Client
	version       string // API 版本
}

// AnthropicClientConfig Anthropic 客户端配置
type AnthropicClientConfig struct {
	BaseURL       string                `mapstructure:"base_url"`
	APIKey        string                `mapstructure:"api_key"`
	ModelName     string                `mapstructure:"model_name"`
	Temperature   float64               `mapstructure:"temperature"`
	MaxTokens     int                   `mapstructure:"max_tokens"`
	RequestPolicy llmcore.RequestPolicy `mapstructure:"request_policy"`
	CostPolicy    llmcore.CostPolicy    `mapstructure:"cost_policy"`
	Version       string                `mapstructure:"version"` // API 版本，默认 "2023-06-01"
}

// AnthropicRequest Anthropic API 请求（wire format）
type AnthropicRequest struct {
	Model       string                   `json:"model"`
	Messages    []AnthropicMessage       `json:"messages"`
	System      string                   `json:"system,omitempty"`
	Tools       []map[string]interface{} `json:"tools,omitempty"` // wire 仍是 map，Adapter 负责 llmcore→wire
	MaxTokens   int                      `json:"max_tokens"`
	Temperature float64                  `json:"temperature,omitempty"`
	Thinking    map[string]interface{}   `json:"thinking,omitempty"`
}

// AnthropicMessage Anthropic 消息格式（wire）
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse Anthropic API 响应（wire）
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
		baseURL:       baseURL,
		apiKey:        cfg.APIKey,
		modelName:     cfg.ModelName,
		temperature:   cfg.Temperature,
		maxTokens:     cfg.MaxTokens,
		requestPolicy: cfg.RequestPolicy,
		costPolicy:    cfg.CostPolicy,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		version: version,
	}
}

// Invoke 调用 Anthropic API
//
// 阶段 1d 改造点：
//  1. 入参 req.Messages 是 []llmcore.Message，迭代时按 role 拆 system / 非 system
//  2. req.Tools 是 []llmcore.ToolSpec，序列化前转成 wire format（map）
//     （Anthropic wire 仍吃 map —— 阶段 2 可以引入 anthropic 专属结构体）
//  3. 响应 tool_use 解析后转 []llmcore.ToolCall，与 openai 协议统一
//  4. 去掉 NormalizeLLMResponse 调用：Content 与 ReasoningContent 严格分离，
//     业务兜底下沉到 agent 消费侧
func (c *AnthropicClient) Invoke(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// 1. 转换 messages：llmcore.Message → wire
	//    system 单独抽到 AnthropicRequest.System 字段；其余按 role/content 构造 AnthropicMessage
	messages := make([]AnthropicMessage, 0, len(req.Messages))
	var systemMessage string

	for _, msg := range req.Messages {
		role := string(msg.Role)

		// 工具结果消息：Anthropic 协议上是 user + tool_result block，
		// 这里走纯文本降级路径，把 ToolCallID/ContentText 拼成可读字符串
		// 阶段 2 再补 Anthropic 原生 tool_result block 支持
		content := msg.ContentText
		if content == "" && len(msg.ContentParts) > 0 {
			// 简化处理：只取 parts 里的 text
			for _, p := range msg.ContentParts {
				if p.Type == "text" {
					content += p.Text
				}
			}
		}

		if role == string(llmcore.RoleSystem) {
			systemMessage += content + "\n"
			continue
		}

		// assistant 消息带 tool_calls → Anthropic wire 是 content[] tool_use block
		// 阶段 1d 简化：tool_use 走 map 注入；这里只放纯文本
		if role == string(llmcore.RoleAssistant) && len(msg.ToolCalls) > 0 {
			// 暂不展开 tool_use block；只把 content 文本带上，tool_calls 由业务侧另传
			// （实际生产 Anthropic 走法在阶段 2 重做）
			messages = append(messages, AnthropicMessage{
				Role:    "assistant",
				Content: content,
			})
			continue
		}

		// tool role 消息：Anthropic 协议上是 user + tool_result block
		// 阶段 1d 简化：合并成 user 文本（带 tool name 标识）
		if role == string(llmcore.RoleTool) {
			name := msg.Name
			if name == "" {
				name = "tool"
			}
			messages = append(messages, AnthropicMessage{
				Role:    "user",
				Content: fmt.Sprintf("[%s result] %s", name, content),
			})
			continue
		}

		// user / assistant 普通文本
		messages = append(messages, AnthropicMessage{
			Role:    role,
			Content: content,
		})
	}

	// 2. 转换 tools：llmcore.ToolSpec → wire map
	//    阶段 1d：直接用 struct 自己的 JSON 序列化结果（外层再包一层）
	tools := make([]map[string]interface{}, 0, len(req.Tools))
	for _, t := range req.Tools {
		// ToolSpec 序列化：{type, function:{name,description,parameters}, read_only}
		// Anthropic wire 期望 {name, description, input_schema}，所以重写字段名
		fn := map[string]interface{}{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": t.Function.Parameters,
		}
		tools = append(tools, fn)
	}

	// 构建请求
	anthropicReq := AnthropicRequest{
		Model:       c.modelName,
		Messages:    messages,
		System:      systemMessage,
		Tools:       tools,
		MaxTokens:   c.effectiveMaxTokens(),
		Temperature: c.effectiveTemperature(),
		Thinking:    c.effectiveThinking(),
	}

	// 序列化请求
	reqBody, err := sonic.Marshal(anthropicReq)
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
			logger.Int("status", resp.StatusCode),
			logger.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("Anthropic API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// 解析响应
	var anthropicResp AnthropicResponse
	if err := sonic.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 转换响应：按 content block 区分 text / tool_use
	result := &LLMResponse{
		ID: anthropicResp.ID,
	}

	for _, content := range anthropicResp.Content {
		switch content.Type {
		case "text":
			result.Content += content.Text
		case "tool_use":
			// 阶段 1d：转 llmcore.ToolCall（ID + Function{Name, Arguments} JSON 字符串）
			// Arguments 把 input map 重新 marshal 成 JSON 字符串，保持和 openai 协议形状一致
			argsBytes, _ := sonic.Marshal(content.InputJSON)
			result.ToolUse = append(result.ToolUse, llmcore.ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: llmcore.ToolCallFunction{
					Name:      content.Name,
					Arguments: string(argsBytes),
				},
			})
		}
	}

	result.Usage = llmcore.Usage{
		PromptTokens:     anthropicResp.Usage.InputTokens,
		CompletionTokens: anthropicResp.Usage.OutputTokens,
		TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
	}
	result.CostUSD = estimateCostUSD(c.costPolicy, result.Usage)

	// 阶段 1d：不再调用 NormalizeLLMResponse —— 协议层严格分离 Content / ReasoningContent
	// 业务兜底（如有）由调用方读 ReasoningContent 自行决定
	return result, nil
}

func (c *AnthropicClient) effectiveTemperature() float64 {
	if c.requestPolicy.DefaultTemperature != nil {
		return *c.requestPolicy.DefaultTemperature
	}
	return c.temperature
}

func (c *AnthropicClient) effectiveMaxTokens() int {
	if c.requestPolicy.DefaultMaxTokens != nil {
		return *c.requestPolicy.DefaultMaxTokens
	}
	return c.maxTokens
}

func (c *AnthropicClient) effectiveThinking() map[string]interface{} {
	switch c.requestPolicy.ReasoningMode {
	case llmcore.ReasoningOff:
		return map[string]interface{}{"type": "disabled"}
	case llmcore.ReasoningLow, llmcore.ReasoningAuto, llmcore.ReasoningHigh:
		return map[string]interface{}{"type": "enabled"}
	default:
		return nil
	}
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
