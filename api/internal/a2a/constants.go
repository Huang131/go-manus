package a2a

import "time"

// A2A JSON-RPC 协议值。
const (
	a2aAgentCardPath          = "/.well-known/agent-card.json"
	a2aJSONRPCVersion         = "2.0"
	a2aProtocolBindingJSONRPC = "JSONRPC"

	a2aMethodMessageSend = "message/send"
	a2aMethodTasksGet    = "tasks/get"
	a2aMethodTasksCancel = "tasks/cancel"

	a2aRoleUser = "user"

	a2aPartKindText = "text"
	a2aPartKindFile = "file"
	a2aPartKindData = "data"

	// a2aKindMessage / a2aKindTask 是 message/send 响应 result 的 kind。
	a2aKindMessage = "message"
	a2aKindTask    = "task"

	// A2A 任务状态。
	a2aTaskStateSubmitted     = "submitted"      // 任务已被远程 Agent 接收、排队，尚未开始处理
	a2aTaskStateWorking       = "working"        // 远程 Agent 正在执行
	a2aTaskStateInputRequired = "input-required" // 需要客户端 补充信息 才能继续（HITL）。问题挂在 status.message 上，如"选 A 还是 B？"
	a2aTaskStateCompleted     = "completed"      // 正常完成
	a2aTaskStateFailed        = "failed"
	a2aTaskStateCanceled      = "canceled"
	a2aTaskStateRejected      = "rejected"      // 远程 Agent 拒绝 执行
	a2aTaskStateAuthRequired  = "auth-required" // 需要客户端完成 认证/授权 （如提供 OAuth 凭证）才能继续
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
