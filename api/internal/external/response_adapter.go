package external

import "strings"

// NormalizeLLMResponse 将不同模型返回的字段统一成业务层可消费的响应。
//
// 规则：
// - Content 优先使用模型明确返回的正文
// - Content 为空时，使用 ReasoningContent 兜底
// - RawContent 保留原始正文，便于排障与后续兼容
// - ToolUse 保持原样，业务层只消费统一后的结构
func NormalizeLLMResponse(resp *LLMResponse) *LLMResponse {
	if resp == nil {
		return nil
	}

	normalized := &LLMResponse{
		ID:               resp.ID,
		Content:          strings.TrimSpace(resp.Content),
		ReasoningContent: strings.TrimSpace(resp.ReasoningContent),
		RawContent:       resp.RawContent,
		ToolUse:          resp.ToolUse,
	}

	if normalized.RawContent == "" {
		normalized.RawContent = normalized.Content
	}
	if normalized.Content == "" && normalized.ReasoningContent != "" {
		normalized.Content = normalized.ReasoningContent
		if normalized.RawContent == "" {
			normalized.RawContent = normalized.ReasoningContent
		}
	}

	return normalized
}
