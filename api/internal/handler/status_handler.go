package handler

import (
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
)

// StatusHandler 状态处理器
type StatusHandler struct {
	service service.StatusService
}

// NewStatusHandler 创建状态处理器
func NewStatusHandler(svc service.StatusService) *StatusHandler {
	return &StatusHandler{service: svc}
}

// Health 健康检查
func (h *StatusHandler) Health(c *gin.Context) {
	status, err := h.service.GetHealthStatus(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status)
}

// GetStatus 获取状态
func (h *StatusHandler) GetStatus(c *gin.Context) {
	status, err := h.service.GetHealthStatus(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, status)
}
