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
//
// 判定输入：完整消息历史（含 RoleTool 执行痕迹）+ 本次请求声明的工具列表。
// RoleTool 消息的 Name 字段标识执行过哪个工具（Agent 构造时写入）。
//
// 规则（声明即禁止）：
//   - 没有任何工具执行过（无 RoleTool 消息）→ 允许 fallback
//   - 已执行过工具（存在 RoleTool 消息）时，以下任一情况禁止 fallback：
//     1. 本次声明的 tools 中存在任一非 ReadOnly 工具——即使尚未执行
//     （跨模型续接会改变写工具的调用决策，行为漂移不可控，宁可保守）
//     2. RoleTool 消息缺 Name，或执行过的工具不在本次声明集中
//     （无法查证副作用，保守禁止；覆盖工具集被收窄的场景）
func CanFallbackAfterToolUse(messages []Message, tools []ToolSpec) bool {
	hasExecutedTool := false
	for _, msg := range messages {
		if msg.Role == model.RoleTool {
			hasExecutedTool = true
			break
		}
	}
	if !hasExecutedTool {
		return true
	}
	// 声明即禁止：任一声明的工具非 ReadOnly → 禁止
	for _, tool := range tools {
		if !tool.ReadOnly {
			return false
		}
	}
	// 已执行的工具必须可查证：带 Name 且仍在本次声明集中
	// （声明集已全 ReadOnly，可查证即无副作用；查不到则保守禁止）
	declared := make(map[string]bool, len(tools))
	for _, tool := range tools {
		declared[tool.Function.Name] = true
	}
	for _, msg := range messages {
		if msg.Role != model.RoleTool {
			continue
		}
		if msg.Name == "" || !declared[msg.Name] {
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
