package llmcore

import "github.com/Huang131/go-manus/api/internal/model"

// SafeContextRatio 安全上下文容量阈值（留 20% 余量，防止边界截断）
const SafeContextRatio = 0.8

// CanFallbackTo 判断 candidate profile 是否可作为 fallback 替换 current profile
//
//  1. 协议必须完全一致（OpenAI <-> Anthropic 不能 fallback，message 格式不同）
//  2. 能力必须完全等价（"满足"不等于"等价"）
//  3. 上下文能装下（带 SafeContextRatio 安全余量）
//  4. MaxOutputTokens 不得缩小（否则生成可能被截断）
//
// 返回 true 时，Orchestrator 可在不丢上下文的前提下切换
func CanFallbackTo(candidate, current ModelProfile) bool {
	// 1. 协议一致
	if candidate.Protocol != current.Protocol {
		return false
	}
	// 2. 能力等价（包含 Vision/Reasoning）
	if !capabilitiesEquivalent(candidate.Capabilities, current.Capabilities) {
		return false
	}
	// 3. 上下文容量：candidate 不能过小（保留 SafeContextRatio 安全余量）
	//    candidate 必须 >= current * SafeContextRatio，否则历史消息可能装不下
	if candidate.Capabilities.MaxContextTokens < int(float64(current.Capabilities.MaxContextTokens)*SafeContextRatio) {
		return false
	}
	// 4. MaxOutputTokens 不得缩小
	if candidate.Capabilities.MaxOutputTokens < current.Capabilities.MaxOutputTokens {
		return false
	}
	return true
}

// CanFallbackAfterToolUse 工具已经被执行后，是否还允许 fallback
// 规则：
//   - 所有工具都是 ReadOnly  → 允许 fallback（没有副作用）
//   - 任一工具已执行且有副作用 → 禁止 fallback（跨 model 上下文不一致）
//
// 注意：此函数只看工具的 ReadOnly 声明，不判断工具是否真的"已执行"
// 调用方需要在工具执行成功后维护"已执行的 tool 列表"状态
func CanFallbackAfterToolUse(executedTools []ToolSpec) bool {
	for _, t := range executedTools {
		if !t.ReadOnly {
			return false
		}
	}
	return true
}

// capabilitiesEquivalent 两个能力画像是否"等价"（不是"满足"）
// 区别（以 a=candidate、b=current 为例）：
//   - "a 满足 b" → a 有 b 的全部能力（可多不可少）
//   - "a 等价 b" → 能力字段完全一致（防止 unexpected behavior）
func capabilitiesEquivalent(a, b model.ModelCapabilities) bool {
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

// EstimateContextTokens 估算 messages 占用的 token 数（粗估）
//
// 实现：按 UTF-8 字节数 / 4 计算，与 "1 token ≈ 4 字符" 的经验值同量级。
//
// 注意：这是近似值，不是真实 tokenizer，中英混排时会有偏差：
// 一个汉字 3 字节，估算约 0.75 token/字，而真实 tokenizer 通常高于此值，
// 所以中文场景可能被低估。调用方必须配合 SafeContextRatio 留出余量，
// 不能把本函数的返回值当作"精确容量"。
//
// 仅用于 fallback 判断（"装得下"），不是准确计费。
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
	// 4 bytes ≈ 1 token
	return total / 4
}
