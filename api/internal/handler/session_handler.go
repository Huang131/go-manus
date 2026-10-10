package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
)

// SessionHandler 会话处理器
type SessionHandler struct {
	service service.SessionService
	sandbox sandbox.Sandbox
	run     service.RunApplicationService
}

// NewSessionHandler 创建会话处理器
func NewSessionHandler(svc service.SessionService, sandbox sandbox.Sandbox, run service.RunApplicationService) *SessionHandler {
	return &SessionHandler{
		service: svc,
		sandbox: sandbox,
		run:     run,
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

// mergeEventMetadata 将 model.Event 的 Data 业务 payload 与元数据（event_id、created_at）合并
// 对齐原 Python 版本的 BaseEventData 平铺结构。
//
// Redis Stream 游标只在读取时可知，因此所有对象 payload 都在此处补入相同的 event_id。
// 这样 SSE id 和 payload 去重键保持一致；Data 解析失败时直接返回原始 Data，保证前端不卡死。
func mergeEventMetadata(ctx context.Context, event *model.Event) []byte {
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
