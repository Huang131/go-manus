package a2a

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/pkg/httpconst"
)

// newA2ARequestID 生成 JSON-RPC 请求 ID。
func newA2ARequestID() string {
	return uuid.New().String()
}

// ============================================================================
// A2A 协议模型（JSON-RPC 2.0 over HTTP）
// ============================================================================

// A2APart 消息部分。
type A2APart struct {
	Kind     string                 `json:"kind"`               // "text" | "file" | "data"
	Text     string                 `json:"text,omitempty"`     // kind == "text"
	File     *A2AFilePart           `json:"file,omitempty"`     // kind == "file"
	Data     json.RawMessage        `json:"data,omitempty"`     // kind == "data"
	Metadata map[string]interface{} `json:"metadata,omitempty"` // 可选元数据
}

// A2AFilePart 文件部分。
type A2AFilePart struct {
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Bytes    string `json:"bytes,omitempty"` // base64 内容，与 URI 二选一
	URI      string `json:"uri,omitempty"`   // 远程地址，与 Bytes 二选一
}

// A2AMessage A2A 消息。
type A2AMessage struct {
	MessageID string                 `json:"messageId,omitempty"`
	ContextID string                 `json:"contextId,omitempty"`
	TaskID    string                 `json:"taskId,omitempty"`
	Role      string                 `json:"role"`           // "user" | "agent"
	Parts     []A2APart              `json:"parts"`          // 消息内容
	Kind      string                 `json:"kind,omitempty"` // message/send 直接响应时为 "message"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// A2ATaskStatus 任务状态。
type A2ATaskStatus struct {
	State     string      `json:"state"`             // 见 A2A task state 常量
	Message   *A2AMessage `json:"message,omitempty"` // 伴随状态的消息（如 input-required 提问）
	Timestamp string      `json:"timestamp,omitempty"`
}

// A2AArtifact 任务工件。
type A2AArtifact struct {
	ArtifactID string                 `json:"artifactId,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Parts      []A2APart              `json:"parts,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// A2ATask A2A 任务对象。
type A2ATask struct {
	ID        string                 `json:"id"`
	ContextID string                 `json:"contextId,omitempty"`
	Status    A2ATaskStatus          `json:"status"`
	Artifacts []A2AArtifact          `json:"artifacts,omitempty"`
	History   []A2AMessage           `json:"history,omitempty"`
	Kind      string                 `json:"kind,omitempty"` // "task"
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// A2AResult message/send 的二选一结果：Task 或 Message。
type A2AResult struct {
	Task    *A2ATask    // kind == "task"
	Message *A2AMessage // kind == "message"
}

// ExtractText 从结果中提取纯文本内容（用于对齐工具层返回给 LLM 的可见文本）。
func (r *A2AResult) ExtractText() string {
	if r == nil {
		return ""
	}

	var parts []A2APart
	if r.Message != nil {
		parts = r.Message.Parts
	} else if r.Task != nil {
		// 任务结果优先取 artifacts，其次取 status.message。
		for _, a := range r.Task.Artifacts {
			parts = append(parts, a.Parts...)
		}
		if len(parts) == 0 && r.Task.Status.Message != nil {
			parts = r.Task.Status.Message.Parts
		}
	}

	var b strings.Builder
	for _, p := range parts {
		if p.Kind == a2aPartKindText && p.Text != "" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// ============================================================================
// JSON-RPC 2.0 信封
// ============================================================================

// A2AJSONRPCRequest JSON-RPC 2.0 请求。
type A2AJSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// A2AJSONRPCError JSON-RPC 2.0 错误对象。
// code 是整数（JSON-RPC 2.0 规范），不是字符串。
type A2AJSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error 实现 error 接口。
func (e *A2AJSONRPCError) Error() string {
	if e == nil {
		return "A2A JSON-RPC 错误"
	}
	return fmt.Sprintf("A2A JSON-RPC 错误 %d: %s", e.Code, e.Message)
}

// A2AJSONRPCResponse JSON-RPC 2.0 响应。
// id 用 RawMessage 承载，因为规范允许 string 或 number 两种形式。
type A2AJSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      json.RawMessage  `json:"id,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *A2AJSONRPCError `json:"error,omitempty"`
}

// A2ARequestParams message/send、message/stream 共用的请求参数。
type A2ARequestParams struct {
	Message       A2AMessage             `json:"message"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// A2ATaskParams tasks/get、tasks/cancel 共用的请求参数。
type A2ATaskParams struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// A2AStreamEvent message/stream 的单个 SSE 事件。
type A2AStreamEvent struct {
	Kind     string       // "status-update" | "artifact-update" | "task"
	Task     *A2ATask     // status-update / task
	Artifact *A2AArtifact // artifact-update
}

// ============================================================================
// A2A JSON-RPC 客户端
// ============================================================================

// A2AClient 面向单个远程 Agent 的 A2A JSON-RPC 客户端。
// 持有 HTTP client，负责协议编码、发送与错误映射；不管理 Agent 卡片与路由。
type A2AClient struct {
	httpClient *http.Client
}

// NewA2AClient 创建 A2A 客户端。timeout 仅作兜底上限，实际以请求 ctx 的 deadline 为准。
func NewA2AClient(timeout time.Duration) *A2AClient {
	return &A2AClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

// SendMessage 同步发送消息（message/send）。
// endpoint 为 JSON-RPC 调用端点；返回 Task 或 Message 二选一结果。
func (c *A2AClient) SendMessage(ctx context.Context, endpoint string, message A2AMessage) (*A2AResult, error) {
	resp, err := c.doRPC(ctx, endpoint, a2aMethodMessageSend, A2ARequestParams{Message: message})
	if err != nil {
		return nil, err
	}
	return parseA2AResult(resp.Result)
}

// StreamMessage 流式发送消息（message/stream），逐 SSE 事件回调 handler。
// handler 返回错误会提前终止流（用于调用方提前取消）。
func (c *A2AClient) StreamMessage(ctx context.Context, endpoint string, message A2AMessage, handler func(A2AStreamEvent) error) error {
	req := A2AJSONRPCRequest{
		JSONRPC: a2aJSONRPCVersion,
		ID:      newA2ARequestID(),
		Method:  a2aMethodMessageStream,
		Params:  A2ARequestParams{Message: message},
	}

	body, err := sonic.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", httpconst.ContentTypeJSON)
	httpReq.Header.Set("Accept", httpconst.ContentTypeSSE)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("调用远程 Agent 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxA2AResponseBytes))
		return fmt.Errorf("调用远程 Agent 出错: HTTP %d: %s", resp.StatusCode, string(b))
	}

	return c.consumeSSE(resp.Body, handler)
}

// GetTask 查询任务状态（tasks/get）。
func (c *A2AClient) GetTask(ctx context.Context, endpoint string, taskID string) (*A2ATask, error) {
	resp, err := c.doRPC(ctx, endpoint, a2aMethodTasksGet, A2ATaskParams{ID: taskID})
	if err != nil {
		return nil, err
	}
	var task A2ATask
	if err := sonic.Unmarshal(resp.Result, &task); err != nil {
		return nil, fmt.Errorf("解析任务失败: %w", err)
	}
	return &task, nil
}

// CancelTask 取消任务（tasks/cancel）。
func (c *A2AClient) CancelTask(ctx context.Context, endpoint string, taskID string) (*A2ATask, error) {
	resp, err := c.doRPC(ctx, endpoint, a2aMethodTasksCancel, A2ATaskParams{ID: taskID})
	if err != nil {
		return nil, err
	}
	var task A2ATask
	if err := sonic.Unmarshal(resp.Result, &task); err != nil {
		return nil, fmt.Errorf("解析任务失败: %w", err)
	}
	return &task, nil
}

// doRPC 发送非流式 JSON-RPC 请求并返回已校验的响应。
func (c *A2AClient) doRPC(ctx context.Context, endpoint, method string, params interface{}) (*A2AJSONRPCResponse, error) {
	req := A2AJSONRPCRequest{
		JSONRPC: a2aJSONRPCVersion,
		ID:      newA2ARequestID(),
		Method:  method,
		Params:  params,
	}

	body, err := sonic.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", httpconst.ContentTypeJSON)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("调用远程 Agent 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxA2AResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("调用远程 Agent 出错: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var rpcResp A2AJSONRPCResponse
	if err := sonic.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// JSON-RPC 层的错误应作为 error 返回（而非塞进结果），这是本次重构的关键修正。
	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}
	if len(rpcResp.Result) == 0 {
		return nil, fmt.Errorf("远程 Agent 返回空结果")
	}
	return &rpcResp, nil
}

// consumeSSE 逐行解析 text/event-stream，回调 handler。
func (c *A2AClient) consumeSSE(r io.Reader, handler func(A2AStreamEvent) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			// 空行标志一个 SSE 事件结束。
			if len(dataLines) == 0 {
				continue
			}
			if err := c.handleSSEData(strings.Join(dataLines, "\n"), handler); err != nil {
				return err
			}
			dataLines = dataLines[:0]
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 SSE 流失败: %w", err)
	}
	// 处理末尾可能缺少空行的事件。
	if len(dataLines) > 0 {
		return c.handleSSEData(strings.Join(dataLines, "\n"), handler)
	}
	return nil
}

func (c *A2AClient) handleSSEData(data string, handler func(A2AStreamEvent) error) error {
	data = strings.TrimSpace(data)
	if data == "" || data == a2aSSEDoneMarker {
		return nil
	}

	var rpcResp A2AJSONRPCResponse
	if err := sonic.Unmarshal([]byte(data), &rpcResp); err != nil {
		return fmt.Errorf("解析 SSE 事件失败: %w", err)
	}
	if rpcResp.Error != nil {
		return rpcResp.Error
	}
	if len(rpcResp.Result) == 0 {
		return nil
	}

	// 根据 kind 分派事件类型。
	var kind struct {
		Kind string `json:"kind"`
	}
	_ = sonic.Unmarshal(rpcResp.Result, &kind)

	event := A2AStreamEvent{Kind: kind.Kind}
	switch kind.Kind {
	case a2aStreamKindArtifactUpdate:
		var payload struct {
			Artifact A2AArtifact `json:"artifact"`
		}
		_ = sonic.Unmarshal(rpcResp.Result, &payload)
		event.Artifact = &payload.Artifact
	default: // status-update / 无 kind（task）
		var task A2ATask
		_ = sonic.Unmarshal(rpcResp.Result, &task)
		event.Task = &task
		if kind.Kind == "" {
			event.Kind = a2aStreamKindTask
		}
	}

	return handler(event)
}

// parseA2AResult 依据 kind 字段将 result 解析为 Task 或 Message。
func parseA2AResult(raw json.RawMessage) (*A2AResult, error) {
	var kind struct {
		Kind string `json:"kind"`
	}
	if err := sonic.Unmarshal(raw, &kind); err != nil {
		return nil, fmt.Errorf("解析响应结果失败: %w", err)
	}

	switch kind.Kind {
	case a2aKindMessage:
		var msg A2AMessage
		if err := sonic.Unmarshal(raw, &msg); err != nil {
			return nil, fmt.Errorf("解析消息失败: %w", err)
		}
		return &A2AResult{Message: &msg}, nil
	default: // "task" 或未声明（旧实现默认 task）
		var task A2ATask
		if err := sonic.Unmarshal(raw, &task); err != nil {
			return nil, fmt.Errorf("解析任务失败: %w", err)
		}
		return &A2AResult{Task: &task}, nil
	}
}
