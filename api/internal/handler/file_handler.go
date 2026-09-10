package handler

import (
	"net/http"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
)

// FileHandler 文件处理器
type FileHandler struct {
	service    service.FileService
	sessionSvc service.SessionService
}

// NewFileHandler 创建文件处理器
// sessionSvc 用于校验上传时携带的 session_id 是否存在；传 nil 则跳过校验（保留兼容）。
func NewFileHandler(svc service.FileService, sessionSvc service.SessionService) *FileHandler {
	return &FileHandler{service: svc, sessionSvc: sessionSvc}
}

// Upload 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	sessionID := c.PostForm("session_id")
	if sessionID == "" {
		response.FromError(c, apperr.BadRequest("session_id is required"))
		return
	}

	// Issue #5：先校验 session 存在性，避免上传到不存在的 session
	if h.sessionSvc != nil {
		if _, err := h.sessionSvc.GetSession(c.Request.Context(), sessionID); err != nil {
			response.FromError(c, apperr.NotFound("session not found: "+sessionID))
			return
		}
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.FromError(c, apperr.BadRequest(err.Error()))
		return
	}

	src, err := file.Open()
	if err != nil {
		response.FromError(c, apperr.Internal(err.Error()))
		return
	}
	defer func() {
		if err := src.Close(); err != nil {
			logger.WarnContext(c.Request.Context(), "关闭上传文件失败", logger.Err(err))
		}
	}()

	result, err := h.service.UploadFile(
		c.Request.Context(),
		sessionID,
		file.Filename,
		src,
		file.Size,
		file.Header.Get("Content-Type"),
	)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, result)
}

// GetInfo 获取文件信息
func (h *FileHandler) GetInfo(c *gin.Context) {
	id := c.Param("id")
	file, err := h.service.GetFileInfo(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, file)
}

// Download 下载文件
func (h *FileHandler) Download(c *gin.Context) {
	id := c.Param("id")
	file, reader, err := h.service.DownloadFile(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	if reader == nil {
		response.FromError(c, service.ErrStorageUnavailable)
		return
	}
	defer func() {
		if err := reader.Close(); err != nil {
			logger.WarnContext(c.Request.Context(), "关闭下载文件失败",
				logger.String("file_id", id),
				logger.Err(err))
		}
	}()

	c.Header("Content-Disposition", "attachment; filename="+file.Filename)
	c.DataFromReader(http.StatusOK, file.Size, file.MimeType, reader, nil)
}
