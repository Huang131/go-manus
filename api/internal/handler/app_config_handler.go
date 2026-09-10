package handler

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
)

// AppConfigHandler 应用配置处理器
type AppConfigHandler struct {
	service service.AppConfigService
}

// NewAppConfigHandler 创建应用配置处理器
func NewAppConfigHandler(svc service.AppConfigService) *AppConfigHandler {
	return &AppConfigHandler{service: svc}
}

// getConfig 通用配置读取：service 返回 nil 时兜底为空配置
func getConfig[T any](c *gin.Context, fn func(ctx context.Context) (*T, error)) {
	cfg, err := fn(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if cfg == nil {
		cfg = new(T)
	}
	response.Success(c, cfg)
}

// updateConfig 通用配置更新
func updateConfig[T any](c *gin.Context, fn func(ctx context.Context, cfg *T) error) {
	var cfg T
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.FromError(c, apperr.BadRequest(err.Error()))
		return
	}
	if err := fn(c.Request.Context(), &cfg); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetLLMConfig 获取 LLM 配置
func (h *AppConfigHandler) GetLLMConfig(c *gin.Context) {
	cfg, err := h.service.GetLLMConfig(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, model.NewLLMConfigResponse(cfg))
}

// UpdateLLMConfig 更新 LLM 配置
func (h *AppConfigHandler) UpdateLLMConfig(c *gin.Context) {
	var request model.LLMConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.FromError(c, apperr.BadRequest(err.Error()))
		return
	}
	if err := h.service.UpdateLLMConfig(c.Request.Context(), request.NewLLMConfig()); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetAgentConfig 获取 Agent 配置
func (h *AppConfigHandler) GetAgentConfig(c *gin.Context) {
	getConfig(c, h.service.GetAgentConfig)
}

// UpdateAgentConfig 更新 Agent 配置
func (h *AppConfigHandler) UpdateAgentConfig(c *gin.Context) {
	updateConfig(c, h.service.UpdateAgentConfig)
}

// GetMCPConfig 获取 MCP 配置
func (h *AppConfigHandler) GetMCPConfig(c *gin.Context) {
	getConfig(c, h.service.GetMCPConfig)
}

// UpdateMCPConfig 更新 MCP 配置
func (h *AppConfigHandler) UpdateMCPConfig(c *gin.Context) {
	updateConfig(c, h.service.UpdateMCPConfig)
}

// DeleteMCPServer 删除 MCP 服务器
func (h *AppConfigHandler) DeleteMCPServer(c *gin.Context) {
	serverName := c.Param("server_name")
	if err := h.service.DeleteMCPServer(c.Request.Context(), serverName); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetA2AConfig 获取 A2A 配置
func (h *AppConfigHandler) GetA2AConfig(c *gin.Context) {
	getConfig(c, h.service.GetA2AConfig)
}

// UpdateA2AConfig 更新 A2A 配置
func (h *AppConfigHandler) UpdateA2AConfig(c *gin.Context) {
	updateConfig(c, h.service.UpdateA2AConfig)
}
