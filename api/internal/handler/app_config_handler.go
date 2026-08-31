package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/service"
	"github.com/mooc-manus/go-manus/api/pkg/response"
)

// AppConfigHandler 应用配置处理器
type AppConfigHandler struct {
	service service.AppConfigService
}

// NewAppConfigHandler 创建应用配置处理器
func NewAppConfigHandler(svc service.AppConfigService) *AppConfigHandler {
	return &AppConfigHandler{service: svc}
}

// GetLLMConfig 获取 LLM 配置
func (h *AppConfigHandler) GetLLMConfig(c *gin.Context) {
	cfg, err := h.service.GetLLMConfig(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, cfg)
}

// UpdateLLMConfig 更新 LLM 配置
func (h *AppConfigHandler) UpdateLLMConfig(c *gin.Context) {
	var cfg model.LLMConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	if err := h.service.UpdateLLMConfig(c.Request.Context(), &cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// GetAgentConfig 获取 Agent 配置
func (h *AppConfigHandler) GetAgentConfig(c *gin.Context) {
	cfg, err := h.service.GetAgentConfig(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, cfg)
}

// UpdateAgentConfig 更新 Agent 配置
func (h *AppConfigHandler) UpdateAgentConfig(c *gin.Context) {
	var cfg model.AgentConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	if err := h.service.UpdateAgentConfig(c.Request.Context(), &cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// GetMCPConfig 获取 MCP 配置
func (h *AppConfigHandler) GetMCPConfig(c *gin.Context) {
	cfg, err := h.service.GetMCPConfig(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, cfg)
}

// UpdateMCPConfig 更新 MCP 配置
func (h *AppConfigHandler) UpdateMCPConfig(c *gin.Context) {
	var cfg model.MCPConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	if err := h.service.UpdateMCPConfig(c.Request.Context(), &cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// DeleteMCPServer 删除 MCP 服务器
func (h *AppConfigHandler) DeleteMCPServer(c *gin.Context) {
	serverName := c.Param("server_name")
	if err := h.service.DeleteMCPServer(c.Request.Context(), serverName); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// GetA2AConfig 获取 A2A 配置
func (h *AppConfigHandler) GetA2AConfig(c *gin.Context) {
	cfg, err := h.service.GetA2AConfig(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, cfg)
}

// UpdateA2AConfig 更新 A2A 配置
func (h *AppConfigHandler) UpdateA2AConfig(c *gin.Context) {
	var cfg model.A2AConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	if err := h.service.UpdateA2AConfig(c.Request.Context(), &cfg); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, nil)
}
