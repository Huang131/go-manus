// Package settings 定义可持久化、可校验的运行配置。
//
// 配置类型位于独立包，避免 model、service 与 agent 分别维护同一份业务语义。
package settings

import "fmt"

const (
	MinAgentIterations    = 1
	MaxAgentIterations    = 100
	MinAgentRetries       = 1
	MaxAgentRetries       = 5
	MinAgentSearchResults = 1
	MaxAgentSearchResults = 10
)

// AgentSettings 控制单次 Agent 执行的循环、重试和搜索结果上限。
type AgentSettings struct {
	MaxIterations    int `json:"max_iterations"`
	MaxRetries       int `json:"max_retries"`
	MaxSearchResults int `json:"max_search_results"`
}

// DefaultAgentSettings 返回配置不存在时使用的显式默认值。
func DefaultAgentSettings() AgentSettings {
	return AgentSettings{
		MaxIterations:    10,
		MaxRetries:       3,
		MaxSearchResults: 10,
	}
}

// Validate 验证完整配置。持久化记录一旦存在，就必须全部合法，不能静默补默认值。
func (s AgentSettings) Validate() error {
	if s.MaxIterations < MinAgentIterations || s.MaxIterations > MaxAgentIterations {
		return fmt.Errorf("max_iterations must be between %d and %d", MinAgentIterations, MaxAgentIterations)
	}
	if s.MaxRetries < MinAgentRetries || s.MaxRetries > MaxAgentRetries {
		return fmt.Errorf("max_retries must be between %d and %d", MinAgentRetries, MaxAgentRetries)
	}
	if s.MaxSearchResults < MinAgentSearchResults || s.MaxSearchResults > MaxAgentSearchResults {
		return fmt.Errorf("max_search_results must be between %d and %d", MinAgentSearchResults, MaxAgentSearchResults)
	}
	return nil
}
