package model

import "time"

// File 文件模型
type File struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Filepath  string    `json:"filepath"`
	Key       string    `json:"key"` // COS 中的路径
	Extension string    `json:"extension"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	SessionID string    `json:"session_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
