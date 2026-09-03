package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/internal/service"
	"github.com/mooc-manus/go-manus/api/pkg/response"
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
		response.Error(c, "session_id is required")
		return
	}

	// Issue #5：先校验 session 存在性，避免上传到不存在的 session
	if h.sessionSvc != nil {
		if _, err := h.sessionSvc.GetSession(c.Request.Context(), sessionID); err != nil {
			response.Error(c, "session not found: "+sessionID)
			return
		}
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, err.Error())
		return
	}

	src, err := file.Open()
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	defer src.Close()

	result, err := h.service.UploadFile(
		c.Request.Context(),
		sessionID,
		file.Filename,
		src,
		file.Size,
		file.Header.Get("Content-Type"),
	)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, result)
}

// GetInfo 获取文件信息
func (h *FileHandler) GetInfo(c *gin.Context) {
	id := c.Param("id")
	file, err := h.service.GetFileInfo(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, file)
}

// Download 下载文件
func (h *FileHandler) Download(c *gin.Context) {
	id := c.Param("id")
	file, reader, err := h.service.DownloadFile(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "attachment; filename="+file.Filename)
	c.DataFromReader(http.StatusOK, file.Size, file.MimeType, reader, nil)
}
