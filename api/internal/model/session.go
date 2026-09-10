package model

import (
	"time"

	"github.com/bytedance/sonic"
)

// SessionStatus 会话状态
type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusWaiting   SessionStatus = "waiting"
	SessionStatusCompleted SessionStatus = "completed"
)

// Session 会话模型，对应一次用户与 Agent 的交互任务。
type Session struct {
	ID                 string        `json:"id"`                          // 会话唯一 ID，UUID 格式
	SandboxID          string        `json:"sandbox_id,omitempty"`        // 关联的沙箱实例 ID
	TaskID             string        `json:"task_id,omitempty"`           // 关联的后台任务 ID，用于追踪长任务
	Title              string        `json:"title"`                       // 会话标题，用户可自定义或由 AI 生成
	UnreadMessageCount int           `json:"unread_message_count"`        // 未读消息数，前端用于显示红点
	LatestMessage      string        `json:"latest_message"`              // 最新一条消息的摘要，用于会话列表展示
	LatestMessageAt    *time.Time    `json:"latest_message_at,omitempty"` // 最新消息时间
	Events             []Event       `json:"events"`                      // 会话事件列表，存储在 Redis 中
	Status             SessionStatus `json:"status"`                      // 会话状态：pending→running→waiting/completed
	DeletedAt          *time.Time    `json:"deleted_at,omitempty"`        // 软删除时间，非空表示已删除
	UpdatedAt          time.Time     `json:"updated_at"`                  // 最后更新时间
	CreatedAt          time.Time     `json:"created_at"`                  // 创建时间
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
