package model

import "time"

// LLMModel 一个 LLM 模型条目
// 支持多模型并存，agent 启动读取 is_default=true 的那一个
type LLMModel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`     // 用户自定义显示名 e.g. "我的 Claude"
	Provider    string    `json:"provider"` // openai/anthropic/google/deepseek/custom
	BaseURL     string    `json:"base_url"`
	APIKey      string    `json:"api_key,omitempty"`
	ModelName   string    `json:"model_name"` // 真实模型名 e.g. claude-3-5-sonnet-20241022
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Tags        []string  `json:"tags"` // 能力标签: vision/tools/long_ctx/...
	IsDefault   bool      `json:"is_default"`
	IsEnabled   bool      `json:"is_enabled"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LLMModelListResponse 列表接口返回
type LLMModelListResponse struct {
	Models []*LLMModel `json:"models"`
}
