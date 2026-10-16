package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	maxRunTracePayloadBytes = 64 * 1024
	redactedRunTraceValue   = "[REDACTED]"
)

// RunTraceEvent 是一次 Run 中可长期回放的关键 UI 过程。
// Data 仅保存结构化 JSON；大对象必须由调用方先转为文件引用。
type RunTraceEvent struct {
	RunID     string          `json:"run_id"`
	Sequence  int64           `json:"sequence"`
	Type      EventType       `json:"event_type"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

// IsPersistableRunTraceEvent 限制长期追踪的事件集合，避免 Redis 高频实时事件变成无界数据库写入。
func IsPersistableRunTraceEvent(eventType EventType) bool {
	switch eventType {
	case EventTypePlan, EventTypeStep, EventTypeToolCalling, EventTypeToolCalled, EventTypeWait, EventTypeError:
		return true
	default:
		return false
	}
}

// SafeRunTracePayload 返回可长期保存的事件负载。
// 工具参数和结果常包含凭据，先从 JSON 副本递归脱敏，避免修改仍要发往实时流的原始事件。
func SafeRunTracePayload(event BaseEvent) (json.RawMessage, error) {
	if event == nil {
		return nil, errors.New("run trace event is nil")
	}
	raw := event.ToJSON()
	if !json.Valid([]byte(raw)) {
		return nil, errors.New("run trace event payload is invalid json")
	}
	var payload interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, errors.New("decode run trace event payload")
	}
	sanitized, err := json.Marshal(sanitizeRunTraceValue(payload, false))
	if err != nil {
		return nil, errors.New("encode run trace event payload")
	}
	if len(sanitized) <= maxRunTracePayloadBytes {
		return sanitized, nil
	}
	return json.Marshal(map[string]interface{}{
		"event_type": event.GetType(),
		"truncated":  true,
		"reason":     "payload exceeds 64KiB",
	})
}

// sanitizeRunTraceValue 仅在工具 arguments/result 作用域内按字段名递归遮盖凭据。
func sanitizeRunTraceValue(value interface{}, sensitiveScope bool) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			if sensitiveScope && isSensitiveRunTraceKey(key) {
				sanitized[key] = redactedRunTraceValue
				continue
			}
			nestedScope := sensitiveScope || key == "arguments" || key == "result"
			sanitized[key] = sanitizeRunTraceValue(nested, nestedScope)
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(typed))
		for index, nested := range typed {
			sanitized[index] = sanitizeRunTraceValue(nested, sensitiveScope)
		}
		return sanitized
	default:
		return value
	}
}

func isSensitiveRunTraceKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "password") ||
		strings.Contains(key, "token") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "authorization")
}
