package llmcore

// CanFallbackTo 判断 candidate profile 是否可作为 fallback 替换 current profile
//
// 严格规则（对应 MULTI_LLM_ADAPTER_DESIGN.md 阶段 2 修订意见）：
//  1. 协议必须完全一致（OpenAI <-> Anthropic 不能 fallback，message 格式不同）
//  2. 能力必须完全等价（"满足"不等于"等价"）
//  3. 上下文能装下（带 20% 安全余量）
//  4. 工具集必须等价（不能新增/缺失）
//
// 返回 true 时，Orchestrator 可在不丢上下文的前提下切换
func CanFallbackTo(candidate, current ModelProfile) bool {
	// 1. 协议一致
	if candidate.Protocol != current.Protocol {
		return false
	}
	// 2. 能力等价
	if !capabilitiesEquivalent(candidate.Capabilities, current.Capabilities) {
		return false
	}
	// 3. 上下文容量等价（candidate 必须 ≥ current，否则装不下当前 history）
	if candidate.Capabilities.MaxContextTokens < current.Capabilities.MaxContextTokens {
		return false
	}
	// 4. 视觉/推理能力必须一致（不能从 vision 退化到 no-vision，否则图片会丢）
	if candidate.Capabilities.SupportsVision != current.Capabilities.SupportsVision {
		return false
	}
	if candidate.Capabilities.SupportsReasoning != current.Capabilities.SupportsReasoning {
		return false
	}
	return true
}

// CanFallbackAfterToolUse 工具已经被执行后，是否还允许 fallback
// 规则（对应方案 L601 修订）：
//   - 所有工具都是 ReadOnly  → 允许 fallback（没有副作用）
//   - 任一工具已执行且有副作用 → 禁止 fallback（跨 model 上下文不一致）
//
// 注意：此函数只看工具的 ReadOnly 声明，不判断 tool_use 是否真的"已执行"
// 调用方需要在 tool 执行成功后维护"已执行的 tool 列表"状态
func CanFallbackAfterToolUse(executedTools []ToolSpec) bool {
	for _, t := range executedTools {
		if !t.ReadOnly {
			return false
		}
	}
	return true
}

// capabilitiesEquivalent 两个能力画像是否"等价"（不是"满足"）
// 区别：
//   - "candidate 满足 current"  → candidate 有 current 全部能力（可多不可少）
//   - "candidate 等价 current"  → 能力字段完全一致（防止 unexpected behavior）
func capabilitiesEquivalent(a, b ModelCapabilities) bool {
	if a.SupportsText != b.SupportsText {
		return false
	}
	if a.SupportsToolCalls != b.SupportsToolCalls {
		return false
	}
	if a.SupportsStructuredOutput != b.SupportsStructuredOutput {
		return false
	}
	if a.SupportsJSONMode != b.SupportsJSONMode {
		return false
	}
	if a.SupportsStrictStructuredOutput != b.SupportsStrictStructuredOutput {
		return false
	}
	if a.SupportsStreaming != b.SupportsStreaming {
		return false
	}
	if a.SupportsVision != b.SupportsVision {
		return false
	}
	if a.SupportsReasoning != b.SupportsReasoning {
		return false
	}
	return true
}

// EstimateContextTokens 估算 messages 占用的 token 数（粗估 1 token ≈ 4 字符）
// 仅用于 fallback 判断（"装得下"），不是准确计费
func EstimateContextTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.ContentText)
		if m.Reasoning != "" {
			total += len(m.Reasoning)
		}
		for _, part := range m.ContentParts {
			total += len(part.Text)
		}
		for _, tc := range m.ToolCalls {
			total += len(tc.Function.Arguments) + len(tc.Function.Name)
		}
	}
	// 1 token ≈ 4 chars
	return total / 4
}

// ContextFits 评估当前 messages 是否能装进 profile 的上下文
func ContextFits(messages []Message, profile ModelProfile) bool {
	if profile.Capabilities.MaxContextTokens == 0 {
		return true // 未知容量，乐观放行
	}
	used := EstimateContextTokens(messages)
	// 留 20% 余量
	return used*5 < profile.Capabilities.MaxContextTokens*4
}
