package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/config"
)

// 编译期契约：真实实现必须满足接口，接口漂移时在此处失败。
var (
	_ Sandbox = (*SandboxClient)(nil)
	_ Browser = (*BrowserClient)(nil)
)

func newSandboxClient(t *testing.T, handler http.HandlerFunc) *SandboxClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewSandboxClient(&config.SandboxConfig{Address: srv.URL, HTTPTimeout: 5})
}

// writeEnvelope 按沙箱统一响应格式 {code,msg,data} 返回，data 为空时序列化为 null。
func writeEnvelope(w http.ResponseWriter, status, code int, msg, data string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == "" {
		data = "null"
	}
	_, _ = io.WriteString(w, fmt.Sprintf(`{"code":%d,"msg":%q,"data":%s}`, code, msg, data))
}

func TestRequireAddress(t *testing.T) {
	client := NewSandboxClient(&config.SandboxConfig{})
	_, err := client.ExecCommand(context.Background(), "s1", "", "ls")
	if err == nil || !strings.Contains(err.Error(), "address not configured") {
		t.Fatalf("ExecCommand() error = %v, want address-not-configured", err)
	}
}

func TestExecCommandDecodesEnvelope(t *testing.T) {
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/shell/exec-command" {
			t.Errorf("path = %q, want /api/shell/exec-command", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q, want application/json", ct)
		}
		writeEnvelope(w, http.StatusOK, http.StatusOK, "命令已提交", `{"status":"running"}`)
	})

	result, err := client.ExecCommand(context.Background(), "s1", "/tmp", "ls -la")
	if err != nil {
		t.Fatalf("ExecCommand() error = %v", err)
	}
	if !result.Success || result.Message != "命令已提交" {
		t.Fatalf("ExecCommand() = %+v, want success with message", result)
	}
	data, ok := result.Data.(map[string]interface{})
	if !ok || data["status"] != "running" {
		t.Fatalf("ExecCommand() data = %#v, want status=running", result.Data)
	}
}

// TestWriteShellInputPreservesFalseBool 回归：bool 字段带 omitempty 会让 false 被省略，
// 沙箱侧落回默认值 true，导致"不按回车"变成"按回车"。
func TestWriteShellInputPreservesFalseBool(t *testing.T) {
	var body string
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeEnvelope(w, http.StatusOK, http.StatusOK, "ok", `{}`)
	})

	if _, err := client.WriteShellInput(context.Background(), "s1", "y", false); err != nil {
		t.Fatalf("WriteShellInput() error = %v", err)
	}
	if !strings.Contains(body, `"press_enter":false`) {
		t.Fatalf("request body = %s, want explicit press_enter=false", body)
	}
	if !strings.Contains(body, `"session_id":"s1"`) {
		t.Fatalf("request body = %s, want session_id", body)
	}
}

// TestPostJSONBusinessErrorOnHTTP200 沙箱用 HTTP 200 承载业务失败（code>=300），
// 必须按错误处理而不是当成成功结果。
func TestPostJSONBusinessErrorOnHTTP200(t *testing.T) {
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, http.StatusOK, http.StatusBadRequest, "命令不合法", "")
	})

	_, err := client.ExecCommand(context.Background(), "s1", "", "ls")
	var apiErr *SandboxAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("ExecCommand() error = %v, want *SandboxAPIError", err)
	}
	if apiErr.Code != http.StatusBadRequest || apiErr.Message != "命令不合法" {
		t.Fatalf("error = %+v, want code=400 message=命令不合法", apiErr)
	}
}

func TestPostJSONHTTPErrorUsesEnvelopeMessage(t *testing.T) {
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, http.StatusInternalServerError, http.StatusInternalServerError, "sandbox exploded", "")
	})

	_, err := client.ExecCommand(context.Background(), "s1", "", "ls")
	var apiErr *SandboxAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("ExecCommand() error = %v, want *SandboxAPIError", err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError || apiErr.Message != "sandbox exploded" {
		t.Fatalf("error = %+v, want status=500 message from envelope", apiErr)
	}
}

func TestPostJSONRejectsMalformedBody(t *testing.T) {
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "not-json")
	})

	_, err := client.ExecCommand(context.Background(), "s1", "", "ls")
	if err == nil {
		t.Fatal("ExecCommand() should fail on malformed response body")
	}
	var apiErr *SandboxAPIError
	if errors.As(err, &apiErr) {
		t.Fatalf("malformed body should not be reported as SandboxAPIError: %+v", apiErr)
	}
}

// TestScreenshotDownloadsBinary 截图必须走"沙箱落文件 + 下载二进制"，
// JSON 信封里的 base64 会被中间环节截断。
func TestScreenshotDownloadsBinary(t *testing.T) {
	want := []byte("png-binary-payload")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/browser/screenshot":
			writeEnvelope(w, http.StatusOK, http.StatusOK, "截图成功", `{"path":"/tmp/shot.png","bytes":18}`)
		case "/api/file/download-file":
			if got := r.URL.Query().Get("filepath"); got != "/tmp/shot.png" {
				t.Errorf("filepath = %q, want /tmp/shot.png", got)
			}
			_, _ = w.Write(want)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewSandboxClient(&config.SandboxConfig{Address: srv.URL, HTTPTimeout: 5})
	got, err := NewBrowserClient(client).Screenshot(context.Background(), false)
	if err != nil {
		t.Fatalf("Screenshot() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("Screenshot() = %q, want %q", got, want)
	}
}

func TestScreenshotRejectsEmptyPath(t *testing.T) {
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, http.StatusOK, http.StatusOK, "ok", `{"path":"","bytes":0}`)
	})

	_, err := NewBrowserClient(client).Screenshot(context.Background(), false)
	if err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("Screenshot() error = %v, want empty-path error", err)
	}
}

func TestFindFilesForwardsGlobPattern(t *testing.T) {
	var body string
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeEnvelope(w, http.StatusOK, http.StatusOK, "ok", `{"files":[]}`)
	})

	if _, err := client.FindFiles(context.Background(), "/tmp", "*.go"); err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}
	if !strings.Contains(body, `"dir_path":"/tmp"`) || !strings.Contains(body, `"glob_pattern":"*.go"`) {
		t.Fatalf("request body = %s, want dir_path and glob_pattern", body)
	}
}

func TestHealthCheckReportsFailure(t *testing.T) {
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	err := client.HealthCheck(context.Background())
	var apiErr *SandboxAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("HealthCheck() error = %v, want SandboxAPIError(503)", err)
	}
}

func TestSandboxAPIErrorFormatting(t *testing.T) {
	err := &SandboxAPIError{StatusCode: 500, Code: 500, Message: "internal error"}
	if err.Error() != "sandbox API error: status=500, message=internal error" {
		t.Errorf("SandboxAPIError.Error() = %q", err.Error())
	}
	empty := &SandboxAPIError{StatusCode: 404}
	if empty.Error() != "sandbox API error: status=404" {
		t.Errorf("SandboxAPIError.Error() = %q", empty.Error())
	}
}

func TestSandboxErrorStatus(t *testing.T) {
	if got := SandboxErrorStatus(&SandboxAPIError{StatusCode: 404, Code: 404}); got != 404 {
		t.Errorf("SandboxErrorStatus() = %d, want 404", got)
	}
	if got := SandboxErrorStatus(context.DeadlineExceeded); got != 0 {
		t.Errorf("SandboxErrorStatus(non-api-error) = %d, want 0", got)
	}
}

func TestScrollDirectionConstants(t *testing.T) {
	// 与沙箱侧 BrowserScrollRequest 的 pattern 保持一致，防止单边改动。
	if ScrollDirectionUp != "up" || ScrollDirectionDown != "down" {
		t.Fatalf("scroll directions = %q/%q, want up/down", ScrollDirectionUp, ScrollDirectionDown)
	}
}
