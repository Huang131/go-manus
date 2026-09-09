package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/agent"
	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
)

// SessionHandler 会话处理器
type SessionHandler struct {
	service service.SessionService
	agent   *agent.AgentService
	sandbox external.Sandbox
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(svc service.SessionService, agentService *agent.AgentService, sandbox external.Sandbox) *SessionHandler {
	return &SessionHandler{
		service: svc,
		agent:   agentService,
		sandbox: sandbox,
	}
}

// Service 返回底层 SessionService，用于路由层组装 handler（如 VNC WS 代理）。
func (h *SessionHandler) Service() service.SessionService {
	return h.service
}

// Create 创建会话 (固定标题为"新对话")
func (h *SessionHandler) Create(c *gin.Context) {
	session, err := h.service.CreateSession(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, session)
}

// Get 获取会话
func (h *SessionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	session, err := h.service.GetSession(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	// 用户打开会话时自动清零未读数，并更新返回值
	session.UnreadMessageCount = 0
	if err := h.service.ClearUnreadCount(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, session)
}

// List 获取会话列表
func (h *SessionHandler) List(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 || limit > 100 {
		response.FromError(c, apperr.BadRequest("limit must be between 1 and 100"))
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		response.FromError(c, apperr.BadRequest("offset must be non-negative"))
		return
	}

	sessions, total, err := h.service.ListSessions(c.Request.Context(), limit, offset)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.SuccessWithTotal(c, sessions, total)
}

// Delete 删除会话 (POST /{session_id}/delete)
func (h *SessionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteSession(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// ClearUnread 清除未读数 (POST /{session_id}/clear-unread)
func (h *SessionHandler) ClearUnread(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.ClearUnreadCount(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// Stream SSE 流式推送所有会话列表
func (h *SessionHandler) Stream(c *gin.Context) {
	setSSEHeaders(c)

	clientGone := c.Request.Context().Done()
	ticker := time.NewTicker(sessionStreamPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			return
		case <-ticker.C:
			sessions, err := h.service.GetAllSessions(c.Request.Context())
			if err != nil {
				continue
			}
			data, _ := sonic.MarshalString(sessions)
			c.SSEvent(sseEventSessions, data)
			c.Writer.Flush()
		}
	}
}

// Chat 聊天 (SSE 流式)
func (h *SessionHandler) Chat(c *gin.Context) {
	id := c.Param("id")
	if h.agent == nil {
		response.FromError(c, apperr.FailedPrecondition("agent 服务未配置"))
		return
	}

	// 区分两种调用语义（对齐原 mooc-manus Python 版本 agent_service.chat 的 if message 分支）：
	//   1. 发送新消息：body 含 "message" 键（非空字符串），进入 chat 流程
	//   2. 空流续读：body 不含 "message" 键 或 message 为空，仅传 event_id 订阅当前 task 的事件流
	// 之前所有路径都 h.agent.Chat(...)，导致 startEmptyStream 触发"空消息"被 LLM 误读为合法输入。
	rawBody, err := c.GetRawData()
	if err != nil {
		response.FromError(c, apperr.BadRequest("读取请求体失败"))
		return
	}

	var req struct {
		Message     *string  `json:"message"`
		Attachments []string `json:"attachments"`
		// 兼容两种命名：前端 startEmptyStream 发的 event_id，HTTP 标准 SSE 的 Last-Event-ID
		EventID string `json:"event_id"`
		// 本次 chat 要用的模型 ID（可选）。空表示走 default。
		// 用于"会话中途临时切换模型"，不影响其他 session 也不改 DB 的 default。
		ModelID string `json:"model_id"`
	}
	if len(rawBody) > 0 {
		if err := sonic.Unmarshal(rawBody, &req); err != nil {
			response.FromError(c, apperr.BadRequest(err.Error()))
			return
		}
	}

	hasMessage := req.Message != nil
	if hasMessage && strings.TrimSpace(*req.Message) == "" {
		response.FromError(c, apperr.BadRequest("消息内容不能为空"))
		return
	}

	setSSEHeaders(c)

	// 创建独立的 context 用于事件获取，不受 HTTP 请求影响
	// 这样即使 HTTP 客户端断开，只要 Agent 还在运行，就会继续推送事件
	eventCtx, eventCancel := context.WithCancel(context.Background())
	defer eventCancel()

	// 检测 HTTP 连接断开，但使用独立的 timeout
	clientGone := c.Request.Context().Done()
	// SSE 流最长持续 30 分钟
	streamTimeout := time.AfterFunc(sessionStreamTimeout, eventCancel)

	var taskID string

	// 仅当"显式发送新消息"时才进入 chat 流程；空流续读跳过此步。
	if hasMessage && strings.TrimSpace(*req.Message) != "" {
		// 创建消息对象
		msg := &llmcore.Message{
			Role:        llmcore.RoleUser,
			ContentText: *req.Message,
			Attachments: req.Attachments,
		}

		// 调用 AgentService 处理聊天消息（异步执行）
		// 注意：AgentService.Chat 内部会创建自己的 context，不受 HTTP 请求影响
		var err error
		chatCtx := external.WithModelID(c.Request.Context(), req.ModelID)
		taskID, err = h.agent.Chat(chatCtx, id, msg)
		if err != nil {
			response.FromError(c, err)
			return
		}

		// 先推送用户消息事件（让前端能立即展示用户发送的内容）
		userPayload, _ := sonic.Marshal(&model.MessageEvent{
			Type:    model.EventTypeMessage,
			Role:    string(msg.Role),
			Message: msg.ContentText,
		})
		c.SSEvent(sseEventMessage, string(userPayload))
		c.Writer.Flush()

		// 再推送 task_id 事件（单独业务类型，前端可识别）
		taskIDData, _ := sonic.Marshal(map[string]interface{}{"task_id": taskID})
		c.SSEvent(sseEventTaskID, string(taskIDData))
		c.Writer.Flush()
	} else {
		// 空流续读：从 session 当前活跃 task 续接事件流
		var err error
		taskID, err = h.agent.GetActiveTaskID(c.Request.Context(), id)
		if err != nil || taskID == "" {
			// 没有活跃 task：保持长连接空闲等待，每 15s 推一个心跳注释避免前端超时。
			// 前端 startEmptyStream 不再因立即关闭而 500ms 死循环重连。
			// 当新 chat 请求创建 task 后再向该 session 推流（见 createTaskNotify 后续扩展）。
			logger.InfoContext(c.Request.Context(), "空流续读: session 无活跃 task，保持长连接心跳",
				logger.String("session_id", id))
			heartbeat := time.NewTicker(sessionHeartbeatInterval)
			defer heartbeat.Stop()
			for {
				select {
				case <-clientGone:
					return
				case <-eventCtx.Done():
					return
				case <-heartbeat.C:
					if _, err := c.Writer.WriteString(": heartbeat\n\n"); err != nil {
						return
					}
					c.Writer.Flush()
				}
			}
		}
		logger.InfoContext(c.Request.Context(), "空流续读: 订阅 session 活跃 task 事件流",
			logger.String("session_id", id),
			logger.String("task_id", taskID),
			logger.String("start_event_id", req.EventID))
	}

	startID := req.EventID // 从前端 lastEventIdRef 续读；空字符串表示从头开始

	for {
		select {
		case <-clientGone:
			// 客户端已断开（页面刷新 / 关闭 / 网络断开）：立刻结束 SSE 流，
			// 不再轮询 Redis 也不再写 Flush，避免在 writer 关闭后疯狂刷写日志。
			// 后端 Agent task 会通过独立 context 继续运行，事件保留在 Redis，
			// 下次前端连上来时通过 lastEventId 续读即可。
			logger.InfoContext(c.Request.Context(), "HTTP client disconnected, stopping SSE stream", logger.String("task_id", taskID))
			eventCancel()
			return
		case <-eventCtx.Done():
			logger.InfoContext(c.Request.Context(), "SSE stream ended", logger.String("task_id", taskID))
			return
		default:
			events, err := h.agent.GetTaskEvents(eventCtx, taskID, startID)
			if err != nil {
				if eventCtx.Err() != nil {
					return
				}
				time.Sleep(sessionEventPollInterval)
				continue
			}

			for _, event := range events {
				startID = event.ID

				// 对齐 Python 版本：event 字段为业务类型，data 字段平铺业务 payload。
				payload := mergeEventMetadata(event)
				c.SSEvent(string(event.Type), string(payload))
				c.Writer.Flush()

				// done / error 都视为终态：结束 SSE 流，避免前端 0/N 计数永远卡在等待态。
				// 后端 task 已结束，下一次连入会通过 lastEventId 续读到 done/error。
				if event.Type == model.EventTypeDone || event.Type == model.EventTypeError {
					streamTimeout.Stop()
					eventCancel()
					return
				}
			}
			time.Sleep(sessionEventPollInterval)
		}
	}
}

// GetFiles 获取会话文件
func (h *SessionHandler) GetFiles(c *gin.Context) {
	id := c.Param("id")
	files, err := h.service.GetSessionFiles(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, files)
}

// ReadFile 查看沙箱文件内容（对齐原项目 POST /sessions/:id/file）
func (h *SessionHandler) ReadFile(c *gin.Context) {
	_ = c.Param("id")
	var req struct {
		Filepath string `json:"filepath"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, apperr.BadRequest("请求参数错误: "+err.Error()))
		return
	}
	if strings.TrimSpace(req.Filepath) == "" {
		response.FromError(c, apperr.BadRequest("filepath 不能为空"))
		return
	}
	if h.sandbox == nil {
		response.FromError(c, apperr.FailedPrecondition("沙箱服务未配置"))
		return
	}
	result, err := h.sandbox.ReadFile(c.Request.Context(), req.Filepath, nil, nil, false, 0)
	if err != nil {
		response.FromError(c, apperr.Internal("读取文件失败: "+err.Error()))
		return
	}
	if !result.Success {
		response.FromError(c, apperr.Internal(result.Message))
		return
	}
	response.Success(c, result.Data)
}

// ReadShell 查看 Shell 输出（对齐原项目 POST /sessions/:id/shell）
func (h *SessionHandler) ReadShell(c *gin.Context) {
	_ = c.Param("id")
	var req struct {
		ShellSessionID string `json:"shell_session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, apperr.BadRequest("请求参数错误: "+err.Error()))
		return
	}
	if strings.TrimSpace(req.ShellSessionID) == "" {
		response.FromError(c, apperr.BadRequest("shell_session_id 不能为空"))
		return
	}
	if h.sandbox == nil {
		response.FromError(c, apperr.FailedPrecondition("沙箱服务未配置"))
		return
	}
	result, err := h.sandbox.ReadShellOutput(c.Request.Context(), req.ShellSessionID, true)
	if err != nil {
		response.FromError(c, apperr.Internal("读取 Shell 输出失败: "+err.Error()))
		return
	}
	if !result.Success {
		response.FromError(c, apperr.Internal(result.Message))
		return
	}
	response.Success(c, result.Data)
}

// Stop 停止会话
func (h *SessionHandler) Stop(c *gin.Context) {
	id := c.Param("id")
	if err := h.agent.StopSession(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// mergeEventMetadata 将 model.Event 的 Data 业务 payload 与元数据（event_id、created_at）合并
// 对齐原 Python 版本的 BaseEventData 平铺结构。
//
// task_runner 创建事件时已把元数据平铺进 payload 并填充 Event.ID，
// 此处对这类事件零序列化直传；仅对旧格式/其他生产方（payload 无元数据）
// 的事件退化为解析重组。Data 解析失败时直接返回原始 Data，保证前端不卡死。
func mergeEventMetadata(event *model.Event) []byte {
	// 新格式：元数据已在 payload 内（以 Event.ID 是否回填为判据）
	if event.ID != "" && len(event.Data) > 0 {
		return event.Data
	}

	// 兜底：payload 未携带元数据，解析重组注入
	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	if len(event.Data) == 0 {
		out, _ := sonic.Marshal(map[string]interface{}{
			"event_id":   event.ID,
			"created_at": createdAt.Unix(),
		})
		return out
	}

	var payload map[string]interface{}
	if err := sonic.Unmarshal(event.Data, &payload); err != nil {
		// 业务 payload 不是对象，无法平铺。直接返回原始 Data，由前端按 type 自行解析。
		return event.Data
	}
	payload["event_id"] = event.ID
	payload["created_at"] = createdAt.Unix()
	out, err := sonic.Marshal(payload)
	if err != nil {
		return event.Data
	}
	return out
}
