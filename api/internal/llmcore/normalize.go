package llmcore

import "github.com/Huang131/go-manus/api/internal/model"

// MergeDeltas 把流式 delta 累积成完整 response
// 用于在流结束时把 content 拼起来
func MergeDeltas(modelName string, deltas []LLMDelta) *LLMResponse {
	resp := &LLMResponse{
		Model:        modelName,
		FinishReason: FinishReasonStop,
		Message:      Message{Role: model.RoleAssistant},
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

	// finish_reason 为 canonical 的 tool_calls 才保留工具调用
	if !isToolCallFinishReason(resp.FinishReason) {
		resp.Message.ToolCalls = nil
	}

	return resp
}

// isToolCallFinishReason 判断 canonical finish_reason 是否表示工具调用终止。
//
// 只接受 FinishReasonToolCalls。厂商原始值（Anthropic 的 tool_use、
// OpenAI 旧版 function_call）由各 Adapter 在出口处翻译成 canonical 值，
// llmcore 不感知任何厂商协议。
func isToolCallFinishReason(reason string) bool {
	return reason == FinishReasonToolCalls
}
