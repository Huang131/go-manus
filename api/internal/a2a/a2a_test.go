package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

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

func TestNewA2AClientUsesDefaultTimeout(t *testing.T) {
	if got := NewA2AClient(0).httpClient.Timeout; got != defaultA2AHTTPTimeout {
		t.Fatalf("HTTP timeout = %s, want default %s", got, defaultA2AHTTPTimeout)
	}
}

func TestA2AClientSendMessageValidatesRPCEnvelope(t *testing.T) {
	client := NewA2AClient(time.Second)
	client.httpClient.Transport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("request method = %s, want POST", req.Method)
		}
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		var rpcReq A2AJSONRPCRequest
		if err := sonic.ConfigDefault.NewDecoder(req.Body).Decode(&rpcReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if rpcReq.JSONRPC != a2aJSONRPCVersion || rpcReq.Method != a2aMethodMessageSend {
			t.Errorf("request = %+v, want JSON-RPC message/send", rpcReq)
		}
		responseBody, err := sonic.Marshal(map[string]interface{}{
			"jsonrpc": a2aJSONRPCVersion,
			"id":      rpcReq.ID,
			"result":  map[string]interface{}{"kind": "message", "role": "agent", "parts": []interface{}{map[string]interface{}{"kind": "text", "text": "ok"}}},
		})
		if err != nil {
			t.Fatalf("marshal response: %v", err)
		}
		return jsonResponse(string(responseBody)), nil
	})

	result, err := client.SendMessage(context.Background(), "https://remote.example/rpc", A2AMessage{
		Role:  a2aRoleUser,
		Parts: []A2APart{{Kind: a2aPartKindText, Text: "hello"}},
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if result.Message == nil || result.ExtractText() != "ok" {
		t.Fatalf("SendMessage() result = %+v, want message text ok", result)
	}
}

func TestA2AClientRejectsInvalidRPCEnvelope(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "wrong JSON-RPC version", body: `{"jsonrpc":"1.0","id":"request-id","result":{}}`},
		{name: "mismatched request id", body: `{"jsonrpc":"2.0","id":"other-id","result":{}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewA2AClient(time.Second)
			client.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(tt.body), nil
			})
			if _, err := client.doRPC(context.Background(), "https://remote.example/rpc", a2aMethodTasksGet, A2ATaskParams{ID: "t1"}); err == nil {
				t.Fatal("doRPC() error = nil, want invalid response envelope error")
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

func TestParseA2AResultRejectsUnknownKind(t *testing.T) {
	for _, raw := range []string{
		`{"kind":"unknown","id":"t1"}`,
		`{"id":"legacy-task","status":{"state":"working"}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			if _, err := parseA2AResult(json.RawMessage(raw)); err == nil {
				t.Fatal("parseA2AResult() error = nil, want unsupported kind error")
			}
		})
	}
}

func TestA2AClientManagerUsesConfiguredTimeout(t *testing.T) {
	manager := NewA2AClientManager()
	want := 250 * time.Millisecond

	if err := manager.Initialize(context.Background(), &A2AClientManagerConfig{Timeout: want}); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if got := manager.client.httpClient.Timeout; got != want {
		t.Fatalf("HTTP timeout = %s, want %s", got, want)
	}
}

func TestA2AClientManagerCleanupInvalidatesInFlightInitialize(t *testing.T) {
	manager := NewA2AClientManager()
	requestStarted := make(chan struct{})
	releaseResponse := make(chan struct{})
	var once sync.Once
	manager.client.httpClient.Transport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		once.Do(func() { close(requestStarted) })
		<-releaseResponse
		return jsonResponse(`{"name":"remote","url":"https://remote.example/rpc"}`), nil
	})

	initErr := make(chan error, 1)
	go func() {
		initErr <- manager.Initialize(context.Background(), &A2AClientManagerConfig{
			Servers: []A2AServerConfig{{ID: "remote", BaseURL: "https://remote.example"}},
		})
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("Initialize() did not start the Agent Card request")
	}
	if err := manager.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	close(releaseResponse)

	select {
	case err := <-initErr:
		if !errors.Is(err, ErrA2AInitializationSuperseded) {
			t.Fatalf("Initialize() error = %v, want %v", err, ErrA2AInitializationSuperseded)
		}
	case <-time.After(time.Second):
		t.Fatal("Initialize() did not return after its response was released")
	}
	if got := len(manager.GetAgentCards()); got != 0 {
		t.Fatalf("GetAgentCards() count = %d, want 0 after Cleanup", got)
	}
}

func TestA2AClientManagerAgentCardsAreIsolatedSnapshots(t *testing.T) {
	manager := NewA2AClientManager()
	manager.agents["remote"] = &A2ARemoteAgent{Card: &A2AAgentCard{
		Name:     "remote",
		Metadata: map[string]interface{}{"owner": "original"},
	}}

	first := manager.GetAgentCards()
	first["remote"].Name = "mutated"
	first["remote"].Metadata["owner"] = "mutated"

	second := manager.GetAgentCards()
	if second["remote"].Name != "remote" {
		t.Fatalf("card name = %q, want isolated original", second["remote"].Name)
	}
	if second["remote"].Metadata["owner"] != "original" {
		t.Fatalf("card metadata = %v, want isolated original", second["remote"].Metadata)
	}
}

func TestA2AClientManagerAuthRequiredDoesNotPoll(t *testing.T) {
	manager := NewA2AClientManager()
	task := &A2ATask{Status: A2ATaskStatus{State: a2aTaskStateAuthRequired}}

	got, err := manager.pollUntilSettled(context.Background(), manager.client, "https://remote.example/rpc", task)
	if err != nil {
		t.Fatalf("pollUntilSettled() error = %v, want nil", err)
	}
	if got != task {
		t.Fatalf("pollUntilSettled() returned a different task: got %+v, want %+v", got, task)
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
