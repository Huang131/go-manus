package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/internal/service"
	"github.com/mooc-manus/go-manus/api/pkg/response"
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
		response.Error(c, err.Error())
		return
	}
	response.Success(c, status)
}

// GetStatus 获取状态
func (h *StatusHandler) GetStatus(c *gin.Context) {
	status, err := h.service.GetHealthStatus(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, status)
}
