package a2a

import "time"

// A2A JSON-RPC 协议值。
const (
	a2aAgentCardPath          = "/.well-known/agent-card.json"
	a2aJSONRPCVersion         = "2.0"
	a2aProtocolBindingJSONRPC = "JSONRPC"

	a2aMethodMessageSend   = "message/send"
	a2aMethodMessageStream = "message/stream"
	a2aMethodTasksGet      = "tasks/get"
	a2aMethodTasksCancel   = "tasks/cancel"

	a2aRoleUser = "user"

	a2aPartKindText = "text"
	a2aPartKindFile = "file"
	a2aPartKindData = "data"

	// a2aKindMessage / a2aKindTask 是 message/send 响应 result 的 kind。
	a2aKindMessage = "message"
	a2aKindTask    = "task"

	// message/stream 的 SSE 事件 kind。
	a2aStreamKindStatusUpdate   = "status-update"
	a2aStreamKindArtifactUpdate = "artifact-update"
	a2aStreamKindTask           = "task"
	a2aSSEDoneMarker            = "[DONE]"

	// A2A 任务状态。
	a2aTaskStateSubmitted     = "submitted"
	a2aTaskStateWorking       = "working"
	a2aTaskStateInputRequired = "input-required"
	a2aTaskStateCompleted     = "completed"
	a2aTaskStateFailed        = "failed"
	a2aTaskStateCanceled      = "canceled"
	a2aTaskStateRejected      = "rejected"
	a2aTaskStateAuthRequired  = "auth-required"
)

// A2A 客户端运行参数。
const (
	defaultA2AHTTPTimeout = 10 * time.Minute

	// maxA2AResponseBytes 单次响应体读取上限，防止超大响应撑爆内存。
	maxA2AResponseBytes = 4 << 20
	// maxA2AMessageBytes 任务消息长度上限（字节）。
	maxA2AMessageBytes = 1 << 20
)

// A2A 长任务轮询参数。
const (
	// a2aPollInterval tasks/get 轮询间隔。
	a2aPollInterval = time.Second
	// a2aPollTimeout 单个任务轮询总时长上限。
	a2aPollTimeout = 5 * time.Minute
)
