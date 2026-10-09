package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

var (
	ErrContextLimitExceeded = errors.New("minimum context exceeds input token budget")
	ErrInvalidContextPolicy = errors.New("invalid context policy")
)

const defaultMaxContextTokens = 32_000

// TokenEstimator 估算 LLM 请求各部分的 token 成本。
// 不同 provider 有各自 tokenizer 时，可替换默认近似估算器。
type TokenEstimator interface {
	EstimateMessage(message llmcore.Message) (int, error)
	EstimateTools(tools []llmcore.ToolSpec) (int, error)
}

// ContextPolicy 约束一次 LLM 请求的完整 context window。
// 输出预留和工具 schema 与消息共享同一个窗口，不能单独绕开预算。
type ContextPolicy struct {
	MaxContextTokens    int
	OutputReserveTokens int
	Estimator           TokenEstimator
}

// ContextRequest 是构建初始对话上下文的输入。
type ContextRequest struct {
	SystemPrompt string
	History      []llmcore.Message
	CurrentQuery string
	Tools        []llmcore.ToolSpec
}

// ContextBuilder 构建并裁剪对话上下文。
type ContextBuilder struct {
	policy ContextPolicy
}

// NewContextBuilder 创建上下文构建器。
func NewContextBuilder(policy ContextPolicy) *ContextBuilder {
	if policy.MaxContextTokens <= 0 {
		policy.MaxContextTokens = defaultMaxContextTokens
	}
	if policy.Estimator == nil {
		policy.Estimator = approximateTokenEstimator{}
	}
	return &ContextBuilder{policy: policy}
}

// Build 组装 system、持久化历史和当前请求；当前请求始终保留，历史按完整组从近到远回填。
func (b *ContextBuilder) Build(request ContextRequest) ([]llmcore.Message, error) {
	prefix := make([]llmcore.Message, 0, 1)
	if request.SystemPrompt != "" {
		prefix = append(prefix, llmcore.Message{Role: model.RoleSystem, ContentText: request.SystemPrompt})
	}
	current := llmcore.Message{Role: model.RoleUser, ContentText: request.CurrentQuery}
	return b.selectMessages(prefix, request.History, []llmcore.Message{current}, request.Tools)
}

// Compact 在每次真实 LLM 调用前应用同一策略。
// 它只裁剪发给 provider 的副本，调用方维护的完整消息仍可用于记忆合并。
func (b *ContextBuilder) Compact(messages []llmcore.Message, tools []llmcore.ToolSpec) ([]llmcore.Message, error) {
	prefixEnd := 0
	for prefixEnd < len(messages) && messages[prefixEnd].Role == model.RoleSystem {
		prefixEnd++
	}
	prefix := append([]llmcore.Message(nil), messages[:prefixEnd]...)

	currentIndex := -1
	for i := len(messages) - 1; i >= prefixEnd; i-- {
		if messages[i].Role == model.RoleUser {
			currentIndex = i
			break
		}
	}

	history := messages[prefixEnd:]
	var required []llmcore.Message
	if currentIndex >= 0 {
		history = messages[prefixEnd:currentIndex]
		required = messages[currentIndex:]
	}
	return b.selectMessages(prefix, history, required, tools)
}

func (b *ContextBuilder) selectMessages(prefix, history, required []llmcore.Message, tools []llmcore.ToolSpec) ([]llmcore.Message, error) {
	budget, err := b.inputBudget(tools)
	if err != nil {
		return nil, err
	}
	used, err := b.estimateMessages(prefix)
	if err != nil {
		return nil, err
	}
	requiredCost, err := b.estimateMessages(required)
	if err != nil {
		return nil, err
	}
	used += requiredCost
	if used > budget {
		return nil, ErrContextLimitExceeded
	}

	groups := completeConversationGroups(history)
	selected := make([][]llmcore.Message, 0, len(groups))
	for i := len(groups) - 1; i >= 0; i-- {
		cost, err := b.estimateMessages(groups[i])
		if err != nil {
			return nil, err
		}
		if used+cost > budget {
			break
		}
		used += cost
		selected = append(selected, groups[i])
	}

	result := make([]llmcore.Message, 0, len(prefix)+len(history)+len(required))
	result = append(result, prefix...)
	for i := len(selected) - 1; i >= 0; i-- {
		result = append(result, selected[i]...)
	}
	result = append(result, required...)
	return result, nil
}

func (b *ContextBuilder) inputBudget(tools []llmcore.ToolSpec) (int, error) {
	if b.policy.OutputReserveTokens < 0 || b.policy.OutputReserveTokens >= b.policy.MaxContextTokens {
		return 0, ErrInvalidContextPolicy
	}
	toolTokens, err := b.policy.Estimator.EstimateTools(tools)
	if err != nil {
		return 0, fmt.Errorf("estimate tool schemas: %w", err)
	}
	if toolTokens < 0 {
		return 0, ErrInvalidContextPolicy
	}
	budget := b.policy.MaxContextTokens - b.policy.OutputReserveTokens - toolTokens
	if budget < 0 {
		return 0, ErrContextLimitExceeded
	}
	return budget, nil
}

func (b *ContextBuilder) estimateMessages(messages []llmcore.Message) (int, error) {
	total := 0
	for _, message := range messages {
		cost, err := b.policy.Estimator.EstimateMessage(message)
		if err != nil {
			return 0, fmt.Errorf("estimate message: %w", err)
		}
		if cost < 0 {
			return 0, ErrInvalidContextPolicy
		}
		total += cost
	}
	return total, nil
}

func completeConversationGroups(messages []llmcore.Message) [][]llmcore.Message {
	groups := make([][]llmcore.Message, 0, len(messages))
	for i := 0; i < len(messages); i++ {
		message := messages[i]
		if len(message.ToolCalls) == 0 {
			if message.Role != model.RoleTool {
				groups = append(groups, []llmcore.Message{message})
			}
			continue
		}
		want := make(map[string]struct{}, len(message.ToolCalls))
		for _, call := range message.ToolCalls {
			if call.ID != "" {
				want[call.ID] = struct{}{}
			}
		}
		if len(want) == 0 {
			continue
		}
		group := []llmcore.Message{message}
		seen := make(map[string]struct{}, len(want))
		j := i + 1
		for ; j < len(messages) && messages[j].Role == model.RoleTool; j++ {
			if _, ok := want[messages[j].ToolCallID]; ok {
				group = append(group, messages[j])
				seen[messages[j].ToolCallID] = struct{}{}
			}
		}
		if len(seen) == len(want) {
			groups = append(groups, group)
		}
		i = j - 1
	}
	return groups
}

type approximateTokenEstimator struct{}

func (approximateTokenEstimator) EstimateMessage(message llmcore.Message) (int, error) {
	payload, err := json.Marshal(message)
	if err != nil {
		return 0, err
	}
	return approximateTokens(string(payload)) + 4, nil
}

func (approximateTokenEstimator) EstimateTools(tools []llmcore.ToolSpec) (int, error) {
	if len(tools) == 0 {
		return 0, nil
	}
	payload, err := json.Marshal(tools)
	if err != nil {
		return 0, err
	}
	return approximateTokens(string(payload)), nil
}

func approximateTokens(content string) int {
	characters := utf8.RuneCountInString(content)
	return (characters + 3) / 4
}
