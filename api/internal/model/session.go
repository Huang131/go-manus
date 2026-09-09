package model

import (
	"github.com/bytedance/sonic"
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
	ID                 string        `json:"id"`
	SandboxID          string        `json:"sandbox_id,omitempty"`
	TaskID             string        `json:"task_id,omitempty"`
	Title              string        `json:"title"`
	UnreadMessageCount int           `json:"unread_message_count"`
	LatestMessage      string        `json:"latest_message"`
	LatestMessageAt    *time.Time    `json:"latest_message_at,omitempty"`
	Events             []Event       `json:"events"`
	Status             SessionStatus `json:"status"`
	DeletedAt          *time.Time    `json:"deleted_at,omitempty"` // 软删除时间，非空表示已删除
	UpdatedAt          time.Time     `json:"updated_at"`
	CreatedAt          time.Time     `json:"created_at"`
}

func unixPointer(value *time.Time) *int64 {
	if value == nil {
		return nil
	}
	u := value.Unix()
	return &u
}

// MarshalJSON 自定义 JSON 序列化
func (s *Session) MarshalJSON() ([]byte, error) {
	type Alias Session
	return sonic.Marshal(&struct {
		*Alias
		UpdatedAt       int64  `json:"updated_at"`
		CreatedAt       int64  `json:"created_at"`
		LatestMessageAt *int64 `json:"latest_message_at,omitempty"`
		DeletedAt       *int64 `json:"deleted_at,omitempty"`
	}{
		Alias:           (*Alias)(s),
		UpdatedAt:       s.UpdatedAt.Unix(),
		CreatedAt:       s.CreatedAt.Unix(),
		LatestMessageAt: unixPointer(s.LatestMessageAt),
		DeletedAt:       unixPointer(s.DeletedAt),
	})
}
