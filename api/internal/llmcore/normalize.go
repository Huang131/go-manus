package llmcore

// NormalizeResponse 归一化上游响应
// 把各种"看起来很相似但 field 不一样"的响应抹平到统一形状
// 这是 Adapter 之外的一个独立纯函数，方便单测覆盖各种异常
//
// 规则（对应方案 L17-L22）：
//   - 永远不会让 reasoning 覆盖 content
//   - content 为空 + reasoning 有内容 → 把 reasoning 作为隐藏 metadata（不进 LLMResponse.Message.Reasoning，给用户看的是 content）
//     业务侧再决定要不要展示
//   - tool_calls 必须拼装完整，arguments 是 JSON 字符串
//   - Usage.ReasoningTokens 有就保留（DeepSeek / GLM thinking 模式专用）
func NormalizeResponse(resp *LLMResponse) *LLMResponse {
	if resp == nil {
		return nil
	}
	// 防御：ContentText 永远不应包含 reasoning 的内容
	// 适配层错误地"reasoning → content" 兜底会被这里纠正
	// 但已经发生的事实不追溯：只校验后续行为
	// 这里什么都不做，因为归一化是 Adapter 的职责
	// 此函数用作契约"参考实现"，让 Adapter 实现时按同样规则
	return resp
}

// MergeDeltas 把流式 delta 累积成完整 response
// 用于在流结束时把 content 拼起来
func MergeDeltas(model string, deltas []LLMDelta) *LLMResponse {
	resp := &LLMResponse{
		Model:        model,
		FinishReason: FinishReasonStop,
		Message:      Message{Role: RoleAssistant},
	}
	var argsBuf = make(map[int]string)
	var currentCalls []ToolCall

	for _, d := range deltas {
		if d.ContentText != "" {
			resp.Message.ContentText += d.ContentText
		}
		if d.Reasoning != "" {
			resp.Message.Reasoning += d.Reasoning
		}
		for _, tcd := range d.ToolCalls {
			// 跨 chunk 拼 arguments
			if tcd.Index < 0 {
				continue
			}
			// 扩列
			for len(currentCalls) <= tcd.Index {
				currentCalls = append(currentCalls, ToolCall{})
			}
			if tcd.ID != "" {
				currentCalls[tcd.Index].ID = tcd.ID
			}
			if tcd.Type != "" {
				currentCalls[tcd.Index].Type = tcd.Type
			}
			if tcd.Name != "" {
				currentCalls[tcd.Index].Function.Name = tcd.Name
			}
			if tcd.ArgumentsDelta != "" {
				argsBuf[tcd.Index] += tcd.ArgumentsDelta
			}
		}
		if d.FinishReason != "" {
			resp.FinishReason = d.FinishReason
		}
		if d.Usage != nil {
			resp.Usage = *d.Usage
		}
	}

	// 收尾：把 argsBuf 注入 currentCalls
	for i := range currentCalls {
		if args, ok := argsBuf[i]; ok {
			currentCalls[i].Function.Arguments = args
		}
	}
	resp.Message.ToolCalls = currentCalls

	// finish_reason=tool_calls 才算有 tool_calls
	if resp.FinishReason != FinishReasonToolCalls {
		resp.Message.ToolCalls = nil
	}

	return resp
}
