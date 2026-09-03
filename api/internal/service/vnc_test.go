package service

import (
	"context"
	"strings"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/model"
)

// VNC 测试不调用 GetSessionFiles，传 nil 即可
// TestSessionService_GetVNCURL_Success 验证：会话存在时，从 sandbox address 派生 VNC URL (ws://host:5901)
func TestSessionService_GetVNCURL_Success(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionServiceWithSandbox(repo, nil, "http://sandbox.local:8080")

	created, err := svc.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	vncURL, err := svc.GetVNCURL(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetVNCURL() error = %v", err)
	}

	want := "ws://sandbox.local:5901"
	if vncURL != want {
		t.Errorf("GetVNCURL() = %s, want %s", vncURL, want)
	}
}

// TestSessionService_GetVNCURL_HTTPS 验证：https:// 派生为 wss://
func TestSessionService_GetVNCURL_HTTPS(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionServiceWithSandbox(repo, nil, "https://sandbox.example.com")

	created, _ := svc.CreateSession(context.Background())
	vncURL, err := svc.GetVNCURL(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetVNCURL() error = %v", err)
	}

	if !strings.HasPrefix(vncURL, "wss://") {
		t.Errorf("GetVNCURL() = %s, want wss:// prefix", vncURL)
	}
	if !strings.HasSuffix(vncURL, ":5901") {
		t.Errorf("GetVNCURL() = %s, want :5901 suffix", vncURL)
	}
}

// TestSessionService_GetVNCURL_NotFound 验证：会话不存在时返回错误
func TestSessionService_GetVNCURL_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionServiceWithSandbox(repo, nil, "http://sandbox.local:8080")

	_, err := svc.GetVNCURL(context.Background(), "nonexistent-id")
	if err == nil {
		t.Error("GetVNCURL() should return error for nonexistent session")
	}
}

// TestSessionService_GetVNCURL_EmptyAddress 验证：sandbox address 为空时返回错误
func TestSessionService_GetVNCURL_EmptyAddress(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionServiceWithSandbox(repo, nil, "")

	created, _ := svc.CreateSession(context.Background())
	_, err := svc.GetVNCURL(context.Background(), created.ID)
	if err == nil {
		t.Error("GetVNCURL() should return error when sandbox address is empty")
	}
}

// 防止 model 包未使用导致编译失败
var _ = model.SessionStatusPending
