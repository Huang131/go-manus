package agent

import (
	"errors"
	"unicode/utf8"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

var ErrContextLimitExceeded = errors.New("minimum context exceeds input token budget")

const defaultMaxInputTokens = 32_000

type ContextPolicy struct {
	MaxInputTokens int
}

// ContextBuilder 构建对话上下文
type ContextBuilder struct {
	policy ContextPolicy
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(policy ContextPolicy) *ContextBuilder {
	if policy.MaxInputTokens <= 0 {
		policy.MaxInputTokens = defaultMaxInputTokens
	}
	return &ContextBuilder{policy: policy}
}

func (b *ContextBuilder) Build(systemPrompt string, history []llmcore.Message, currentQuery string) ([]llmcore.Message, error) {
	prefix := make([]llmcore.Message, 0, 1)
	if systemPrompt != "" {
		prefix = append(prefix, llmcore.Message{Role: model.RoleSystem, ContentText: systemPrompt})
	}
	current := llmcore.Message{Role: model.RoleUser, ContentText: currentQuery}
	used := estimateMessages(prefix) + estimateMessage(current)
	if used > b.policy.MaxInputTokens {
		return nil, ErrContextLimitExceeded
	}

	groups := completeConversationGroups(history)
	selected := make([][]llmcore.Message, 0, len(groups))
	for i := len(groups) - 1; i >= 0; i-- {
		cost := estimateMessages(groups[i])
		if used+cost > b.policy.MaxInputTokens {
			break
		}
		used += cost
		selected = append(selected, groups[i])
	}

	result := make([]llmcore.Message, 0, len(prefix)+len(history)+1)
	result = append(result, prefix...)
	for i := len(selected) - 1; i >= 0; i-- {
		result = append(result, selected[i]...)
	}
	result = append(result, current)
	return result, nil
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

func estimateMessages(messages []llmcore.Message) int {
	total := 0
	for _, message := range messages {
		total += estimateMessage(message)
	}
	return total
}

func estimateMessage(message llmcore.Message) int {
	characters := utf8.RuneCountInString(message.ContentText) + utf8.RuneCountInString(message.Reasoning)
	for _, call := range message.ToolCalls {
		characters += utf8.RuneCountInString(call.ID) + utf8.RuneCountInString(call.Function.Name) + utf8.RuneCountInString(call.Function.Arguments)
	}
	// Four characters per token plus a small per-message envelope. This is a
	// conservative fallback and can be replaced by a model tokenizer later.
	return (characters+3)/4 + 4
}
