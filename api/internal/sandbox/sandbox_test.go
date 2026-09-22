package sandbox

import (
	"context"
	"encoding/json"
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

// TestRequireAddress 守护契约：未配置沙箱地址时必须在发出请求前给出可读原因，
// 而不是把空地址拼成非法 URL 后抛出难以定位的传输错误。
func TestRequireAddress(t *testing.T) {
	client := NewSandboxClient(&config.SandboxConfig{})
	_, err := client.ExecCommand(context.Background(), "s1", "", "ls")
	if err == nil || !strings.Contains(err.Error(), "address not configured") {
		t.Fatalf("ExecCommand() error = %v, want address-not-configured", err)
	}
}

// TestExecCommandDecodesEnvelope 守护契约：{code,msg,data} 信封的 msg 与 data
// 必须原样进入 ToolResult，工具层不再理解传输格式。
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
// 沙箱侧落回默认值 true，导致“不按回车”变成“按回车”。
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

// TestFindFilesSendsGlobPattern 守护契约：dir_path/glob_pattern 必须真实进入请求体。
// 沙箱侧 glob_pattern 缺省为 "*"，字段被吞掉会把查找范围从指定模式放大为目录下全部文件。
func TestFindFilesSendsGlobPattern(t *testing.T) {
	var body string
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		writeEnvelope(w, http.StatusOK, http.StatusOK, "ok", `{"files":[]}`)
	})

	if _, err := client.FindFiles(context.Background(), "/tmp", "*.go"); err != nil {
		t.Fatalf("FindFiles() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("request body 不是合法 JSON: %v, body=%s", err, body)
	}
	if payload["dir_path"] != "/tmp" || payload["glob_pattern"] != "*.go" {
		t.Fatalf("request payload = %#v, want dir_path=/tmp glob_pattern=*.go", payload)
	}
}

// TestPostJSONErrorMapping 守护契约：失败响应必须映射为可判定的错误，不能被当成成功结果。
//
// 三个分支覆盖三种真实失败形态：
//   - HTTP 200 + 信封 code>=300：协议兜底。当前 Python 侧异常都带真实状态码，
//     此分支用于防止未来某端把业务失败塞进 HTTP 200 后被静默吞掉。
//   - 非 200 + 信封：保留 HTTP 状态码与信封 msg，handler 依赖它映射业务响应。
//   - 非 200/200 + 非 JSON：不能伪装成 SandboxAPIError，否则上层错误分类失真。
func TestPostJSONErrorMapping(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		body           string
		wantStatusCode int
		wantCode       int
		wantMessage    string
		wantNotAPIErr  bool
	}{
		{
			// 兜底分支：SandboxErrorStatus 只能回落到 HTTP 状态码（200），
			// handler 必须自己判定 code>=300 才算失败。
			name:           "HTTP 200 承载业务失败",
			status:         http.StatusOK,
			body:           `{"code":400,"msg":"命令不合法","data":null}`,
			wantStatusCode: http.StatusOK,
			wantCode:       http.StatusBadRequest,
			wantMessage:    "命令不合法",
		},
		{
			name:           "非 200 保留状态码与信封消息",
			status:         http.StatusInternalServerError,
			body:           `{"code":500,"msg":"sandbox exploded","data":null}`,
			wantStatusCode: http.StatusInternalServerError,
			wantCode:       http.StatusInternalServerError,
			wantMessage:    "sandbox exploded",
		},
		{
			name:          "非 JSON 响应不伪装成 SandboxAPIError",
			status:        http.StatusOK,
			body:          "not-json",
			wantNotAPIErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			})

			_, err := client.ExecCommand(context.Background(), "s1", "", "ls")
			if err == nil {
				t.Fatal("ExecCommand() 应返回错误")
			}

			var apiErr *SandboxAPIError
			if tt.wantNotAPIErr {
				if errors.As(err, &apiErr) {
					t.Fatalf("错误被误分类为 SandboxAPIError: %+v", apiErr)
				}
				return
			}

			if !errors.As(err, &apiErr) {
				t.Fatalf("ExecCommand() error = %v, want *SandboxAPIError", err)
			}
			if apiErr.StatusCode != tt.wantStatusCode || apiErr.Code != tt.wantCode || apiErr.Message != tt.wantMessage {
				t.Fatalf("error = %+v, want status=%d code=%d msg=%q",
					apiErr, tt.wantStatusCode, tt.wantCode, tt.wantMessage)
			}
			// handler 通过 SandboxErrorStatus 把客户端错误映射成 HTTP 响应码。
			if got := SandboxErrorStatus(err); got != tt.wantStatusCode {
				t.Errorf("SandboxErrorStatus() = %d, want %d", got, tt.wantStatusCode)
			}
		})
	}
}

// TestScreenshotDownloadsBinary 守护契约：截图必须走“沙箱落文件 + 下载二进制”，
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

// TestScreenshotRejectsEmptyPath 守护契约：沙箱未返回落盘路径时必须立即失败，
// 避免带着空路径继续下载并报出误导性错误。
func TestScreenshotRejectsEmptyPath(t *testing.T) {
	client := newSandboxClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, http.StatusOK, http.StatusOK, "ok", `{"path":"","bytes":0}`)
	})

	_, err := NewBrowserClient(client).Screenshot(context.Background(), false)
	if err == nil || !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("Screenshot() error = %v, want empty-path error", err)
	}
}

// TestHealthCheckReportsFailure 守护契约：/health 非 200 必须映射为 SandboxAPIError，
// 让启动期健康检查能区分“沙箱未就绪”和“客户端代码错误”。
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
