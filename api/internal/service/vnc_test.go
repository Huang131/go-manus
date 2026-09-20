package service

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/apperr"
)

// VNC 测试不调用 GetSessionFiles，fileRepo 传 nil 即可
// TestSessionService_GetVNCURL_Success 验证：会话存在时，从 sandbox address 派生 VNC URL (ws://host:5901)
func TestSessionService_GetVNCURL_Success(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "http://sandbox.local:8080")

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

// TestSessionService_GetVNCURL_HTTPS 验证：https:// 派生为 wss://，且端口替换为 VNC 端口
func TestSessionService_GetVNCURL_HTTPS(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "https://sandbox.example.com")

	created, _ := svc.CreateSession(context.Background())
	vncURL, err := svc.GetVNCURL(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetVNCURL() error = %v", err)
	}

	want := "wss://sandbox.example.com:5901"
	if vncURL != want {
		t.Errorf("GetVNCURL() = %s, want %s", vncURL, want)
	}
}

// TestSessionService_GetVNCURL_NotFound 验证：会话不存在时返回 NotFound（而非放行任意 sessionID）
func TestSessionService_GetVNCURL_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "http://sandbox.local:8080")

	_, err := svc.GetVNCURL(context.Background(), "nonexistent-id")
	requireAppErrKind(t, err, apperr.KindNotFound)
}

// TestSessionService_GetVNCURL_EmptyAddress 验证：sandbox address 未配置时返回 FailedPrecondition
func TestSessionService_GetVNCURL_EmptyAddress(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	created, _ := svc.CreateSession(context.Background())
	_, err := svc.GetVNCURL(context.Background(), created.ID)
	requireAppErrKind(t, err, apperr.KindFailedPrecondition)
}
