package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/internal/service"
	"github.com/mooc-manus/go-manus/api/pkg/response"
)

// FileHandler 文件处理器
type FileHandler struct {
	service service.FileService
}

// NewFileHandler 创建文件处理器
func NewFileHandler(svc service.FileService) *FileHandler {
	return &FileHandler{service: svc}
}

// Upload 上传文件
func (h *FileHandler) Upload(c *gin.Context) {
	sessionID := c.PostForm("session_id")
	if sessionID == "" {
		response.Error(c, "session_id is required")
		return
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
