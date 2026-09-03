package llmcore

import (
	"errors"
	"fmt"
)

// ErrorKind 上游错误分类
// Orchestrator 据此决定是否 fallback / 重试
type ErrorKind string

const (
	// KindAuth 401/403 鉴权失败：fallback 也救不了，让用户重新配 key
	KindAuth ErrorKind = "auth"
	// KindNotFound 404 模型不存在
	KindNotFound ErrorKind = "not_found"
	// KindRateLimit 429 限流：fallback 到备用
	KindRateLimit ErrorKind = "rate_limit"
	// KindServer 5xx 上游服务异常：fallback 到备用
	KindServer ErrorKind = "server"
	// KindBadRequest 400 协议不兼容：fallback 救不了，但提示重新检查配置
	KindBadRequest ErrorKind = "bad_request"
	// KindContentFilter 内容安全：fallback 也救不了
	KindContentFilter ErrorKind = "content_filter"
	// KindNetwork 网络层错误：fallback / 重试都可能
	KindNetwork ErrorKind = "network"
	// KindTimeout 超时：fallback / 重试都可能
	KindTimeout ErrorKind = "timeout"
	// KindContextLimit 上下文超限：fallback 救不了，提示压缩
	KindContextLimit ErrorKind = "context_limit"
	// KindUnknown 其他
	KindUnknown ErrorKind = "unknown"
)

// ProviderError 上游错误
// Adapter 把所有厂商错误归一化到这种结构
// Orchestrator 据此做路由决策
type ProviderError struct {
	Kind ErrorKind
	// StatusCode HTTP 状态码（如果非 HTTP 错误，置 0）
	StatusCode int
	// Message 错误描述（已脱敏，不含 api_key）
	Message string
	// Provider 哪个 provider
	Provider string
	// Model 哪个 model
	Model string
	// Cause 原始错误
	Cause error
	// Retryable 是否可重试（同 provider 重试，不是 fallback）
	Retryable bool
	// Fallbackable 是否可 fallback（切到其他 model）
	Fallbackable bool
}

func (e *ProviderError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("provider %s/%s error [%s] status=%d: %s (cause: %v)",
			e.Provider, e.Model, e.Kind, e.StatusCode, e.Message, e.Cause)
	}
	return fmt.Sprintf("provider %s/%s error [%s] status=%d: %s",
		e.Provider, e.Model, e.Kind, e.StatusCode, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Cause
}

// Is 判断 error 是否为 ProviderError 且 Kind 匹配
func IsKind(err error, kind ErrorKind) bool {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Kind == kind
	}
	return false
}

// NewProviderError 构造 ProviderError
func NewProviderError(kind ErrorKind, provider, model, message string) *ProviderError {
	return &ProviderError{
		Kind:         kind,
		Provider:     provider,
		Model:        model,
		Message:      message,
		Retryable:    isRetryable(kind),
		Fallbackable: isFallbackable(kind),
	}
}

// isRetryable 默认规则
func isRetryable(kind ErrorKind) bool {
	switch kind {
	case KindRateLimit, KindServer, KindNetwork, KindTimeout:
		return true
	default:
		return false
	}
}

// isFallbackable 默认规则
// 即使同 provider 重试能恢复，仍允许 fallback
// 注意：内容安全/限流/网络错误都允许 fallback
func isFallbackable(kind ErrorKind) bool {
	switch kind {
	case KindAuth, KindNotFound, KindBadRequest, KindContentFilter, KindContextLimit:
		return false // 配置/语义类问题，fallback 也救不了
	default:
		return true
	}
}
