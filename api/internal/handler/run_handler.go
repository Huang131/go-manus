package handler

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RunHandler 暴露 Run 生命周期和实时事件 API。
type RunHandler struct {
	app    service.RunApplicationService
	events service.RunEventReader
}

func NewRunHandler(app service.RunApplicationService, events service.RunEventReader) *RunHandler {
	return &RunHandler{app: app, events: events}
}

type createRunRequest struct {
	Message     string   `json:"message"`
	Attachments []string `json:"attachments,omitempty"`
	ModelID     string   `json:"model_id,omitempty"`
}

type submitRunInputRequest struct {
	IdempotencyKey   string   `json:"idempotency_key"`
	ReplyToMessageID string   `json:"reply_to_message_id"`
	Message          string   `json:"message"`
	Attachments      []string `json:"attachments,omitempty"`
	ModelID          string   `json:"model_id,omitempty"`
}

// Create 创建一次顶层 Run；幂等键优先读取请求头，也接受 JSON 字段。
func (h *RunHandler) Create(c *gin.Context) {
	if h == nil || h.app == nil {
		response.FromError(c, apperr.FailedPrecondition("run service is not configured"))
		return
	}
	var req createRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, apperr.BadRequest("invalid run request"))
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		response.FromError(c, apperr.BadRequest("message is required"))
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if key == "" {
		key = uuid.NewString()
	}
	ctx := c.Request.Context()
	if strings.TrimSpace(req.ModelID) != "" {
		ctx = llm.WithModelID(ctx, strings.TrimSpace(req.ModelID))
	}
	run, err := h.app.Create(ctx, service.CreateApplicationRunInput{
		SessionID: c.Param("sessionId"), IdempotencyKey: key,
		Content: req.Message, AttachmentIDs: req.Attachments,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, run.Run)
}

// SubmitInput 提交 waiting_input Run 的回答。
func (h *RunHandler) SubmitInput(c *gin.Context) {
	if h == nil || h.app == nil {
		response.FromError(c, apperr.FailedPrecondition("run service is not configured"))
		return
	}
	var req submitRunInputRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, apperr.BadRequest("invalid run input request"))
		return
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" {
		key = strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	}
	if key == "" {
		response.FromError(c, apperr.BadRequest("idempotency key is required"))
		return
	}
	ctx := c.Request.Context()
	if strings.TrimSpace(req.ModelID) != "" {
		ctx = llm.WithModelID(ctx, strings.TrimSpace(req.ModelID))
	}
	result, err := h.app.SubmitInput(ctx, c.Param("runId"), service.SubmitInputRequest{
		IdempotencyKey: key, ReplyToMessageID: req.ReplyToMessageID,
		Content: req.Message, Attachments: req.Attachments,
	})
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, result.Run)
}

func (h *RunHandler) Get(c *gin.Context) {
	if h == nil || h.app == nil {
		response.FromError(c, apperr.FailedPrecondition("run service is not configured"))
		return
	}
	run, err := h.app.Get(c.Request.Context(), c.Param("runId"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, run)
}

// ListBySession 返回会话历史 Run 和持久化消息，供刷新后的 UI 重建基础时间线。
func (h *RunHandler) ListBySession(c *gin.Context) {
	if h == nil || h.app == nil {
		response.FromError(c, apperr.FailedPrecondition("run service is not configured"))
		return
	}
	limit, offset, err := parseRunPage(c)
	if err != nil {
		response.FromError(c, err)
		return
	}
	page, err := h.app.ListBySessionID(c.Request.Context(), c.Param("sessionId"), limit, offset)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, page)
}

func parseRunPage(c *gin.Context) (int, int, error) {
	limit, offset := 50, 0
	for name, target := range map[string]*int{"limit": &limit, "offset": &offset} {
		raw := strings.TrimSpace(c.Query(name))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 || (name == "limit" && value == 0) {
			return 0, 0, apperr.BadRequest("invalid run pagination")
		}
		*target = value
	}
	if limit > 100 {
		return 0, 0, apperr.BadRequest("run limit must be at most 100")
	}
	return limit, offset, nil
}

func (h *RunHandler) Cancel(c *gin.Context) {
	if h == nil || h.app == nil {
		response.FromError(c, apperr.FailedPrecondition("run service is not configured"))
		return
	}
	run, err := h.app.Cancel(c.Request.Context(), c.Param("runId"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, run)
}

// Events 按 Last-Event-ID 读取 Run 的短期实时事件流。
func (h *RunHandler) Events(c *gin.Context) {
	if h == nil || h.app == nil || h.events == nil {
		response.FromError(c, apperr.FailedPrecondition("run event stream is not configured"))
		return
	}
	runID := c.Param("runId")
	if _, err := h.app.Get(c.Request.Context(), runID); err != nil {
		response.FromError(c, err)
		return
	}
	setSSEHeaders(c)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 30*time.Minute)
	defer cancel()
	startID := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	if startID == "" {
		startID = strings.TrimSpace(c.Query("last_event_id"))
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
