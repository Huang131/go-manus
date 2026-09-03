package model

import (
	"encoding/json"
	"time"
)

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusWaiting   SessionStatus = "waiting"
	SessionStatusCompleted SessionStatus = "completed"
)

// Session 会话模型
type Session struct {
	ID                 string                 `json:"id"`
	SandboxID          string                 `json:"sandbox_id,omitempty"`
	TaskID             string                 `json:"task_id,omitempty"`
	Title              string                 `json:"title"`
	UnreadMessageCount int                    `json:"unread_message_count"`
	LatestMessage      string                 `json:"latest_message"`
	LatestMessageAt    *time.Time             `json:"latest_message_at,omitempty"`
	Events             []Event                `json:"events"`
	Memories           map[string]interface{} `json:"memories"`
	Status             SessionStatus          `json:"status"`
	UpdatedAt          time.Time              `json:"updated_at"`
	CreatedAt          time.Time              `json:"created_at"`
}

// MarshalJSON 自定义 JSON 序列化
func (s *Session) MarshalJSON() ([]byte, error) {
	type Alias Session
	return json.Marshal(&struct {
		*Alias
		UpdatedAt int64 `json:"updated_at"`
		CreatedAt int64 `json:"created_at"`
	}{
		Alias:     (*Alias)(s),
		UpdatedAt: s.UpdatedAt.Unix(),
		CreatedAt: s.CreatedAt.Unix(),
	})
}
