package external

import (
	"bytes"
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"strings"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/pkg/logger"
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

// AnthropicMessage Anthropic 消息格式（wire）。
// Content 既可以是纯字符串，也可以是内容块数组（assistant 的 tool_use、
// user 的 tool_result / 多模态 part 都要求块数组形状）。
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// AnthropicContent Anthropic 内容块（请求与响应共用）
type AnthropicContent struct {
	Type string `json:"type"`
	// text / thinking 块
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
	// tool_use 块
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
	// image 块
	Source *AnthropicImageSource `json:"source,omitempty"`
	// tool_result 块（仅出现在请求侧 user 消息中）
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
}

// AnthropicImageSource 图片来源（URL 引用）
type AnthropicImageSource struct {
	Type string `json:"type"` // "url"
	URL  string `json:"url"`
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

// AnthropicUsage Anthropic 使用量
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewAnthropicClient 创建 Anthropic 客户端
func NewAnthropicClient(cfg *AnthropicClientConfig) *AnthropicClient {
	version := cfg.Version
	if version == "" {
		version = defaultAnthropicVersion
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
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
			Timeout: defaultExternalHTTPTimeout,
		},
		version: version,
	}
}

// Invoke 调用 Anthropic API
//
// 协议映射：
//  1. system 消息抽到 AnthropicRequest.System；user/assistant 按序转换
//  2. assistant 的 ToolCalls 转为 content[] 中的 tool_use 块（input 解析自 Arguments JSON）
//  3. tool 角色消息转为 user 消息中的 tool_result 块（相邻 tool 消息合并，满足角色交替约束）
//  4. tools 定义转 {name, description, input_schema}
//  5. 响应按块解析：text → ContentText，thinking → Reasoning，tool_use → ToolCalls
func (c *AnthropicClient) Invoke(ctx context.Context, req *LLMRequest) (*llmcore.LLMResponse, error) {
	messages, systemMessage := c.toAnthropicMessages(req.Messages)

	// 转换 tools：llmcore.ToolSpec → Anthropic wire（{name, description, input_schema}）
	tools := make([]map[string]interface{}, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, map[string]interface{}{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": t.Function.Parameters,
		})
	}

	anthropicReq := AnthropicRequest{
		Model:       c.modelName,
		Messages:    messages,
		System:      systemMessage,
		Tools:       tools,
		MaxTokens:   c.effectiveMaxTokens(),
		Temperature: c.effectiveTemperature(),
		Thinking:    c.effectiveThinking(),
	}

	reqBody, err := sonic.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + anthropicMessagesPath
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", c.version)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

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

	var anthropicResp AnthropicResponse
	if err := sonic.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	result := &llmcore.LLMResponse{
		ID: anthropicResp.ID,
		Message: llmcore.Message{
			Role: llmcore.RoleAssistant,
		},
		FinishReason: anthropicResp.StopReason,
	}

	for i := range anthropicResp.Content {
		block := &anthropicResp.Content[i]
		switch block.Type {
		case anthropicContentTypeText:
			result.Message.ContentText += block.Text
		case anthropicContentTypeThinking:
			// 扩展思考块 → Reasoning（不回传给用户，可审计、可随记忆往返）
			result.Message.Reasoning += block.Thinking
		case anthropicContentTypeToolUse:
			result.Message.ToolCalls = append(result.Message.ToolCalls, llmcore.ToolCall{
				ID:   block.ID,
				Type: llmcore.ToolTypeFunction,
				Function: llmcore.ToolCallFunction{
					Name:      block.Name,
					Arguments: string(mustMarshalMap(block.Input)),
				},
			})
		default:
			// redacted_thinking 等未知块：无法还原内容，跳过
			logger.Debug("Anthropic 响应包含未处理的内容块", logger.String("type", block.Type))
		}
	}

	result.Usage = llmcore.Usage{
		PromptTokens:     anthropicResp.Usage.InputTokens,
		CompletionTokens: anthropicResp.Usage.OutputTokens,
		TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
	}
	result.CostUSD = estimateCostUSD(c.costPolicy, result.Usage)

	return result, nil
}

// toAnthropicMessages 将 llmcore 消息列表转为 Anthropic wire 格式。
// 返回值第二个是抽取出来的 system 文本。
func (c *AnthropicClient) toAnthropicMessages(reqMessages []llmcore.Message) ([]AnthropicMessage, string) {
	messages := make([]AnthropicMessage, 0, len(reqMessages))
	var systemMessage string

	appendMessage := func(role string, content interface{}) {
		messages = append(messages, AnthropicMessage{Role: role, Content: content})
	}

	// appendToolResult 把 tool_result 块追加到最后一条 user 消息；
	// 若最后一条不是"由 tool 结果聚合出的 user 消息"，则新建。
	// Anthropic 要求 user/assistant 角色严格交替，相邻 tool 结果必须合并进同一条 user 消息。
	appendToolResult := func(block AnthropicContent) {
		if n := len(messages); n > 0 {
			if blocks, ok := messages[n-1].Content.([]AnthropicContent); ok && len(blocks) > 0 && blocks[0].Type == anthropicContentTypeToolResult {
				messages[n-1].Content = append(blocks, block)
				return
			}
		}
		appendMessage("user", []AnthropicContent{block})
	}

	for _, msg := range reqMessages {
		switch msg.Role {
		case llmcore.RoleSystem:
			systemMessage += msg.ContentText + "\n"
			continue

		case llmcore.RoleAssistant:
			if len(msg.ToolCalls) == 0 {
				appendMessage("assistant", msg.ContentText)
				continue
			}
			// assistant + tool_calls → content[]：text 块（可空）+ tool_use 块
			blocks := make([]AnthropicContent, 0, len(msg.ToolCalls)+1)
			if text := textOf(msg); text != "" {
				blocks = append(blocks, AnthropicContent{Type: llmcore.ContentTypeText, Text: text})
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, AnthropicContent{
					Type:  anthropicContentTypeToolUse,
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: parseJSONMap(tc.Function.Arguments),
				})
			}
			appendMessage("assistant", blocks)

		case llmcore.RoleTool:
			appendToolResult(AnthropicContent{
				Type:      anthropicContentTypeToolResult,
				ToolUseID: msg.ToolCallID,
				Content:   msg.ContentText,
			})

		default: // user
			appendMessage("user", userContent(msg))
		}
	}

	return messages, systemMessage
}

// textOf 取消息的纯文本内容（ContentText 优先，ContentParts 中的 text 拼接兜底）
func textOf(msg llmcore.Message) string {
	if msg.ContentText != "" {
		return msg.ContentText
	}
	var sb strings.Builder
	for _, p := range msg.ContentParts {
		if p.Type == llmcore.ContentTypeText {
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

// userContent 构造 user 消息内容：纯文本用 string；含图片等多模态 part 时用块数组
func userContent(msg llmcore.Message) interface{} {
	hasMultimodal := false
	for _, p := range msg.ContentParts {
		if p.Type != llmcore.ContentTypeText {
			hasMultimodal = true
			break
		}
	}
	if !hasMultimodal {
		return textOf(msg)
	}

	blocks := make([]AnthropicContent, 0, len(msg.ContentParts))
	if text := msg.ContentText; text != "" {
		blocks = append(blocks, AnthropicContent{Type: anthropicContentTypeText, Text: text})
	}
	for _, p := range msg.ContentParts {
		switch p.Type {
		case llmcore.ContentTypeText:
			blocks = append(blocks, AnthropicContent{Type: llmcore.ContentTypeText, Text: p.Text})
		case llmcore.ContentTypeImageURL:
			if p.ImageURL != nil && p.ImageURL.URL != "" {
				// Anthropic 支持 URL source 的图片输入
				blocks = append(blocks, AnthropicContent{
					Type: anthropicContentTypeImage,
					Source: &AnthropicImageSource{
						Type: anthropicImageSourceTypeURL,
						URL:  p.ImageURL.URL,
					},
				})
			}
		default:
			logger.Debug("Anthropic 跳过不支持的内容块", logger.String("type", p.Type))
		}
	}
	return blocks
}

// parseJSONMap 把工具调用参数 JSON 字符串解析为 map；空串或非法 JSON 返回空 map
func parseJSONMap(arguments string) map[string]interface{} {
	input := make(map[string]interface{})
	if arguments != "" {
		_ = sonic.Unmarshal([]byte(arguments), &input)
	}
	return input
}

// mustMarshalMap 把 input map 序列化回 JSON 字符串（与 openai 协议的 Arguments 形状保持一致）
func mustMarshalMap(input map[string]interface{}) []byte {
	if input == nil {
		return []byte("{}")
	}
	data, err := sonic.Marshal(input)
	if err != nil {
		return []byte("{}")
	}
	return data
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

// effectiveThinking 构造扩展思考配置。
// Anthropic 要求 thinking 启用时必须带 budget_tokens，且 budget < max_tokens。
func (c *AnthropicClient) effectiveThinking() map[string]interface{} {
	switch c.requestPolicy.ReasoningMode {
	case llmcore.ReasoningOff:
		return map[string]interface{}{"type": anthropicThinkingTypeDisabled}
	case llmcore.ReasoningLow, llmcore.ReasoningAuto, llmcore.ReasoningHigh:
		budget := 2048
		if c.requestPolicy.ReasoningMode == llmcore.ReasoningHigh {
			budget = 8192
		}
		if maxTokens := c.effectiveMaxTokens(); maxTokens <= budget {
			budget = maxTokens / 2
		}
		if budget < 1024 {
			// max_tokens 过小，无法启用扩展思考
			return nil
		}
		return map[string]interface{}{"type": anthropicThinkingTypeEnabled, "budget_tokens": budget}
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
