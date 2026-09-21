package a2a

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"
)

// TestA2AAgentCard_ResolveEndpoint 守护卡片端点解析契约：
// 必须兼容 v1.0（supportedInterfaces）与 v0.3（根层 url）两版结构。
func TestA2AAgentCard_ResolveEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		card        *A2AAgentCard
		wantURL     string
		wantVersion string
		wantErr     bool
	}{
		{
			name: "v1.0 supportedInterfaces 选 JSONRPC",
			card: &A2AAgentCard{
				SupportedInterfaces: []A2AInterface{
					{URL: "https://a.example.com/http", ProtocolBinding: "HTTP+JSON", ProtocolVersion: "1.0"},
					{URL: "https://a.example.com/rpc", ProtocolBinding: "JSONRPC", ProtocolVersion: "1.0"},
				},
			},
			wantURL:     "https://a.example.com/rpc",
			wantVersion: "1.0",
		},
		{
			name: "v0.3 回退根层 url",
			card: &A2AAgentCard{
				URL:             "https://a.example.com/spec03",
				ProtocolVersion: "0.3.0",
			},
			wantURL:     "https://a.example.com/spec03",
			wantVersion: "0.3.0",
		},
		{
			name: "v1.0 interface 未声明 binding 默认视为 JSONRPC",
			card: &A2AAgentCard{
				SupportedInterfaces: []A2AInterface{
					{URL: "https://a.example.com/rpc"},
				},
			},
			wantURL: "https://a.example.com/rpc",
		},
		{
			name:    "空卡片报错",
			card:    &A2AAgentCard{},
			wantErr: true,
		},
		{
			name:    "nil 卡片报错",
			card:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, version, err := tt.card.ResolveEndpoint()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ResolveEndpoint() 期望返回错误，实际 url=%q", url)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveEndpoint() 意外错误: %v", err)
			}
			if url != tt.wantURL {
				t.Errorf("url = %q, want %q", url, tt.wantURL)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

// TestA2AAgentCapabilities_CanStream 守护 v1.0/v0.3 命名的聚合判断。
func TestA2AAgentCapabilities_CanStream(t *testing.T) {
	tests := []struct {
		name string
		cap  *A2AAgentCapabilities
		want bool
	}{
		{"nil", nil, false},
		{"v1.0 streaming", &A2AAgentCapabilities{Streaming: true}, true},
		{"v0.3 supportsStreaming", &A2AAgentCapabilities{SupportsStreaming: true}, true},
		{"都不支持", &A2AAgentCapabilities{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cap.CanStream(); got != tt.want {
				t.Errorf("CanStream() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseA2AResult 守护 message/send 响应 kind 分派：task 与 message 二选一。
func TestParseA2AResult(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantTask bool
		wantMsg  bool
		wantText string
	}{
		{
			name:     "task 响应",
			raw:      `{"kind":"task","id":"t1","status":{"state":"completed"},"artifacts":[{"parts":[{"kind":"text","text":"任务完成"}]}]}`,
			wantTask: true,
			wantText: "任务完成",
		},
		{
			name:     "message 直接响应",
			raw:      `{"kind":"message","messageId":"m1","role":"agent","parts":[{"kind":"text","text":"你好"}]}`,
			wantMsg:  true,
			wantText: "你好",
		},
		{
			name:     "缺 kind 默认按 task 解析",
			raw:      `{"id":"t2","status":{"state":"working"}}`,
			wantTask: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseA2AResult(json.RawMessage(tt.raw))
			if err != nil {
				t.Fatalf("parseA2AResult() 错误: %v", err)
			}
			if (result.Task != nil) != tt.wantTask {
				t.Errorf("Task = %v, want %v", result.Task != nil, tt.wantTask)
			}
			if (result.Message != nil) != tt.wantMsg {
				t.Errorf("Message = %v, want %v", result.Message != nil, tt.wantMsg)
			}
			if tt.wantText != "" {
				if got := result.ExtractText(); got != tt.wantText {
					t.Errorf("ExtractText() = %q, want %q", got, tt.wantText)
				}
			}
		})
	}
}

// TestA2AJSONRPCError_CodeIsInt 守护 JSON-RPC 2.0 错误 code 必须是整数解析。
// 旧实现用 string 承载，遇到规范返回的整数 code 会反序列化失败。
func TestA2AJSONRPCError_CodeIsInt(t *testing.T) {
	raw := []byte(`{"jsonrpc":"2.0","id":"1","error":{"code":-32600,"message":"Invalid Request"}}`)
	var resp A2AJSONRPCResponse
	if err := sonic.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("解析 JSON-RPC 错误响应失败: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("期望解析出 Error，实际为 nil")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("Error.Code = %d, want -32600", resp.Error.Code)
	}
	if resp.Error.Message != "Invalid Request" {
		t.Errorf("Error.Message = %q, want %q", resp.Error.Message, "Invalid Request")
	}
}
