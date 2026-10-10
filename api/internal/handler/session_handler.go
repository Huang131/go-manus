package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SessionHandler 会话处理器
type SessionHandler struct {
	service service.SessionService
	sandbox sandbox.Sandbox
	run     service.RunApplicationService
	events  service.RunEventReader
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(svc service.SessionService, sandbox sandbox.Sandbox, run service.RunApplicationService, events service.RunEventReader) *SessionHandler {
	return &SessionHandler{
		service: svc,
		sandbox: sandbox,
		run:     run,
		events:  events,
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
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(service.DefaultSessionListLimit)))
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
	// 先请求取消活跃 Run，避免软删会话后仍继续消耗 LLM 和沙箱资源。
	if h.run != nil {
		if active, err := h.run.GetActiveBySessionID(c.Request.Context(), id); err != nil {
			logger.Warn("删除会话前查询活跃 Run 失败（继续删除）", logger.String("session_id", id), logger.Err(err))
		} else if active != nil {
			if _, cancelErr := h.run.Cancel(c.Request.Context(), active.ID); cancelErr != nil {
				logger.Warn("删除会话前停止活跃 Run 失败（继续删除）", logger.String("session_id", id), logger.Err(cancelErr))
			}
		}
	}
	if err := h.service.DeleteSession(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// RenameSession 重命名会话 (POST /{session_id}/rename)
func (h *SessionHandler) RenameSession(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, apperr.BadRequest("请求参数错误: "+err.Error()))
		return
	}
	if err := h.service.RenameSession(c.Request.Context(), id, req.Title); err != nil {
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
				logger.DebugContext(c.Request.Context(), "获取会话列表失败，跳过本轮 SSE 推送", logger.Err(err))
				continue
			}
			data, err := sonic.MarshalString(map[string]interface{}{"sessions": sessions})
			if err != nil {
				logger.ErrorContext(c.Request.Context(), "序列化会话 SSE 数据失败", logger.Err(err))
				continue
			}
			if err := writeSSEEvent(c, sseEventSessions, "", []byte(data)); err != nil {
				return
			}
		}
	}
}

// chatRequest Chat 请求体
type chatRequest struct {
	Message     *string  `json:"message"`
	Attachments []string `json:"attachments"`
	// 兼容两种命名：前端 startEmptyStream 发的 event_id，HTTP 标准 SSE 的 Last-Event-ID
	EventID string `json:"event_id"`
	// 本次 chat 要用的模型 ID（可选）。空表示走 default。
	// 用于"会话中途临时切换模型"，不影响其他 session 也不改 DB 的 default。
	ModelID string `json:"model_id"`
}

// maxChatBodyBytes Chat 请求体大小上限，防止超大 body 全量读入内存
const maxChatBodyBytes = 1 << 20 // 1MB

// parseChatRequest 解析并校验 Chat 请求体。
// Message 用 *string 区分"未传 message 键"（空流续读）与"传了空串"（参数错误）。
func parseChatRequest(c *gin.Context) (*chatRequest, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChatBodyBytes)
	rawBody, err := c.GetRawData()
	if err != nil {
		return nil, apperr.BadRequest("读取请求体失败")
	}

	var req chatRequest
	if len(rawBody) > 0 {
		if err := sonic.Unmarshal(rawBody, &req); err != nil {
			return nil, apperr.BadRequest(err.Error())
		}
	}
	if req.Message != nil && strings.TrimSpace(*req.Message) == "" {
		return nil, apperr.BadRequest("消息内容不能为空")
	}
	return &req, nil
}

// Chat 聊天 (SSE 流式)
//
// 区分两种调用语义（对齐原 mooc-manus Python 版本 agent_service.chat 的 if message 分支）：
//  1. 发送新消息：body 含 "message" 键（非空字符串），进入 chat 流程
//  2. 空流续读：body 不含 "message" 键，仅传 event_id 订阅当前 Run 的事件流
func (h *SessionHandler) Chat(c *gin.Context) {
	h.chatRun(c, c.Param("id"))
}

// chatRun 保留旧 Session Chat 的 HTTP/SSE 形状，但执行事实全部来自 Run。
func (h *SessionHandler) chatRun(c *gin.Context, sessionID string) {
	if h.run == nil || h.events == nil {
		response.FromError(c, apperr.FailedPrecondition("run execution is not configured"))
		return
	}
	req, err := parseChatRequest(c)
	if err != nil {
		response.FromError(c, err)
		return
	}
	startID := strings.TrimSpace(req.EventID)
	if startID == "" {
		startID = strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	}
	var runID string
	if req.Message != nil {
		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if key == "" {
			key = uuid.NewString()
		}
		ctx := c.Request.Context()
		if req.ModelID != "" {
			ctx = llm.WithModelID(ctx, req.ModelID)
		}
		created, createErr := h.run.Create(ctx, service.CreateApplicationRunInput{
			SessionID: sessionID, IdempotencyKey: key, Content: *req.Message, AttachmentIDs: req.Attachments,
		})
		if createErr != nil {
			response.FromError(c, createErr)
			return
		}
		runID = created.Run.ID
		setSSEHeaders(c)
		userPayload, marshalErr := sonic.Marshal(&model.MessageEvent{Type: model.EventTypeMessage, Role: model.RoleUser, Message: *req.Message})
		if marshalErr != nil {
			return
		}
		if err := writeSSEEvent(c, string(model.EventTypeMessage), "", userPayload); err != nil {
			return
		}
		taskPayload, marshalErr := sonic.Marshal(map[string]string{"task_id": runID})
		if marshalErr != nil {
			return
		}
		if err := writeSSEEvent(c, string(sseEventTaskID), "", taskPayload); err != nil {
			return
		}
	} else {
		active, lookupErr := h.run.GetActiveBySessionID(c.Request.Context(), sessionID)
		if lookupErr != nil {
			response.FromError(c, lookupErr)
			return
		}
		setSSEHeaders(c)
		if active == nil {
			heartbeatCtx, heartbeatCancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), sessionStreamTimeout)
			defer heartbeatCancel()
			h.idleHeartbeat(c, heartbeatCtx, sessionID)
			return
		}
		runID = active.ID
	}
	h.streamRunEvents(c, runID, startID)
}

// streamRunEvents 读取按 Run 隔离的 Redis Stream；游标语义与 RunHandler 保持一致。
func (h *SessionHandler) streamRunEvents(c *gin.Context, runID, startID string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), sessionStreamTimeout)
	defer cancel()
	if startID == "" {
		startID = "0-0"
	}
	for {
		events, err := h.events.Read(ctx, runID, startID)
		if err != nil {
			_ = writeSSEEvent(c, "stream_error", "", []byte(`{"message":"stream temporarily unavailable"}`))
			return
		}
		for _, event := range events {
			if event == nil {
				continue
			}
			startID = event.ID
			if err := writeSSEEvent(c, string(event.Type), event.ID, mergeEventMetadata(ctx, event)); err != nil {
				return
			}
			if event.Type == model.EventTypeDone || event.Type == model.EventTypeError || event.Type == model.EventTypeWait {
				return
			}
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-ctx.Done():
			return
		default:
		}
	}
}

// idleHeartbeat 空流续读但无活跃 Run：保持长连接，定期推送 SSE 心跳注释
// 防止前端超时断连。客户端断开或流超时（eventCtx 结束）时返回。
func (h *SessionHandler) idleHeartbeat(c *gin.Context, eventCtx context.Context, sessionID string) {
	logger.InfoContext(c.Request.Context(), "空流续读: session 无活跃 Run，保持长连接心跳",
		logger.String("session_id", sessionID))

	heartbeat := time.NewTicker(sessionHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
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

// callSandbox 校验 session 存在与沙箱可用性后执行沙箱调用，统一错误映射。
// ReadFile / ReadShell 共用。
func (h *SessionHandler) callSandbox(c *gin.Context, sessionID, action string, call func(ctx context.Context) (*model.ToolResult, error)) {
	// 校验 session 存在，避免对任意 session id / 路径的越权读取
	if _, err := h.service.GetSession(c.Request.Context(), sessionID); err != nil {
		response.FromError(c, apperr.NotFound("session not found: "+sessionID))
		return
	}
	if h.sandbox == nil {
		response.FromError(c, apperr.FailedPrecondition("沙箱服务未配置"))
		return
	}
	result, err := call(c.Request.Context())
	if err != nil {
		response.FromError(c, sandboxAppError(action, err))
		return
	}
	if result == nil {
		response.FromError(c, sandboxResultError(action, nil))
		return
	}
	if !result.Success {
		response.FromError(c, sandboxResultError(action, result))
		return
	}
	response.Success(c, result.Data)
}

func sandboxAppError(action string, err error) error {
	statusCode := sandbox.SandboxErrorStatus(err)
	message := action + ": " + err.Error()
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return apperr.BadRequest(message)
	case http.StatusNotFound:
		return apperr.NotFound(message)
	case http.StatusPreconditionFailed:
		return apperr.FailedPrecondition(message)
	case http.StatusRequestTimeout, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return apperr.Unavailable(message)
	default:
		return apperr.Internal(message)
	}
}

func sandboxResultError(action string, result *model.ToolResult) error {
	if result == nil {
		return apperr.Internal(action + ": 沙箱返回空结果")
	}
	if result.StatusCode == 0 {
		return apperr.Internal(result.Message)
	}
	return sandboxAppError(action, &sandbox.SandboxAPIError{
		StatusCode: result.StatusCode,
		Code:       result.StatusCode,
		Message:    result.Message,
	})
}

// ReadFile 查看沙箱文件内容（对齐原项目 POST /sessions/:id/file）
func (h *SessionHandler) ReadFile(c *gin.Context) {
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
	h.callSandbox(c, c.Param("id"), "读取文件失败", func(ctx context.Context) (*model.ToolResult, error) {
		return h.sandbox.ReadFile(ctx, req.Filepath, nil, nil, false, 0)
	})
}

// ReadShell 查看 Shell 输出（对齐原项目 POST /sessions/:id/shell）
func (h *SessionHandler) ReadShell(c *gin.Context) {
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
	h.callSandbox(c, c.Param("id"), "读取 Shell 输出失败", func(ctx context.Context) (*model.ToolResult, error) {
		return h.sandbox.ReadShellOutput(ctx, req.ShellSessionID, true)
	})
}

// Stop 停止会话
func (h *SessionHandler) Stop(c *gin.Context) {
	id := c.Param("id")
	if h.run == nil {
		response.FromError(c, apperr.FailedPrecondition("run execution is not configured"))
		return
	}
	active, err := h.run.GetActiveBySessionID(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if active != nil {
		if _, err := h.run.Cancel(c.Request.Context(), active.ID); err != nil {
			response.FromError(c, err)
			return
		}
	}
	response.Success(c, nil)
}

// mergeEventMetadata 将 model.Event 的 Data 业务 payload 与元数据（event_id、created_at）合并
// 对齐原 Python 版本的 BaseEventData 平铺结构。
//
// Run event publisher 创建事件时已把元数据平铺进 payload 并填充 Event.ID，
// 此处对这类事件零序列化直传；仅对旧格式/其他生产方（payload 无元数据）
// 的事件退化为解析重组。Data 解析失败时直接返回原始 Data，保证前端不卡死。
func mergeEventMetadata(ctx context.Context, event *model.Event) []byte {
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
		out, err := sonic.Marshal(map[string]interface{}{
			model.EventMetadataKeyEventID:   event.ID,
			model.EventMetadataKeyCreatedAt: createdAt.Unix(),
		})
		if err != nil {
			logger.ErrorContext(ctx, "序列化空事件元数据失败", logger.Err(err))
			return []byte(`{}`)
		}
		return out
	}

	var payload map[string]interface{}
	if err := sonic.Unmarshal(event.Data, &payload); err != nil {
		// 业务 payload 不是对象，无法平铺。直接返回原始 Data，由前端按 type 自行解析。
		return event.Data
	}
	if payload == nil {
		return event.Data
	}
	payload[model.EventMetadataKeyEventID] = event.ID
	payload[model.EventMetadataKeyCreatedAt] = createdAt.Unix()
	out, err := sonic.Marshal(payload)
	if err != nil {
		logger.ErrorContext(ctx, "序列化事件元数据失败", logger.Err(err))
		return event.Data
	}
	return out
}
