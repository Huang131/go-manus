package handler

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/agent"
	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/response"
	"github.com/gin-gonic/gin"
)

// AppConfigHandler 应用配置处理器
type AppConfigHandler struct {
	service  service.AppConfigService
	reloader configRuntimeReloader
}

type configRuntimeReloader interface {
	ReloadAgentConfig(*agent.AgentConfig)
	ReloadMCPConfig(context.Context, *agent.MCPConfig) error
	ReloadA2AConfig(context.Context, *agent.A2AConfig) error
}

// NewAppConfigHandler 创建应用配置处理器
func NewAppConfigHandler(svc service.AppConfigService, reloaders ...configRuntimeReloader) *AppConfigHandler {
	var reloader configRuntimeReloader
	if len(reloaders) > 0 {
		reloader = reloaders[0]
	}
	return &AppConfigHandler{service: svc, reloader: reloader}
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

// reloadOrFail 执行运行时重载，失败时统一映射为 Unavailable（label 前缀区分域名）。
func reloadOrFail(c *gin.Context, label string, reload func(*gin.Context) error) error {
	if err := reload(c); err != nil {
		return apperr.Unavailable(label + "运行时重载失败: " + err.Error())
	}
	return nil
}

// GetAgentConfig 获取 Agent 配置
func (h *AppConfigHandler) GetAgentConfig(c *gin.Context) {
	getConfig(c, h.service.GetAgentConfig)
}

// UpdateAgentConfig 更新 Agent 配置
func (h *AppConfigHandler) UpdateAgentConfig(c *gin.Context) {
	updateConfig(c, func(ctx context.Context, cfg *model.AgentConfig) error {
		if err := h.service.UpdateAgentConfig(ctx, cfg); err != nil {
			return err
		}
		if h.reloader != nil {
			h.reloader.ReloadAgentConfig(&agent.AgentConfig{
				MaxIterations:    cfg.MaxIterations,
				MaxRetries:       cfg.MaxRetries,
				MaxSearchResults: cfg.MaxSearchResults,
			})
		}
		return nil
	})
}

// GetMCPConfig 获取 MCP 配置
func (h *AppConfigHandler) GetMCPConfig(c *gin.Context) {
	getConfig(c, h.service.GetMCPConfig)
}

// UpdateMCPConfig 更新 MCP 配置
func (h *AppConfigHandler) UpdateMCPConfig(c *gin.Context) {
	updateConfig(c, func(ctx context.Context, cfg *model.MCPConfig) error {
		if err := h.service.UpdateMCPConfig(ctx, cfg); err != nil {
			return err
		}
		return reloadOrFail(c, "MCP", h.reloadMCP)
	})
}

// DeleteMCPServer 删除 MCP 服务器
func (h *AppConfigHandler) DeleteMCPServer(c *gin.Context) {
	serverName := c.Param("server_name")
	if err := h.service.DeleteMCPServer(c.Request.Context(), serverName); err != nil {
		response.FromError(c, err)
		return
	}
	if err := reloadOrFail(c, "MCP", h.reloadMCP); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// UpdateMCPServerEnabled 更新 MCP 服务启用状态。
func (h *AppConfigHandler) UpdateMCPServerEnabled(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, apperr.BadRequest(err.Error()))
		return
	}
	if err := h.service.UpdateMCPServerEnabled(c.Request.Context(), c.Param("server_name"), req.Enabled); err != nil {
		response.FromError(c, err)
		return
	}
	if err := reloadOrFail(c, "MCP", h.reloadMCP); err != nil {
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
	updateConfig(c, func(ctx context.Context, cfg *model.A2AConfig) error {
		if err := h.service.UpdateA2AConfig(ctx, cfg); err != nil {
			return err
		}
		return reloadOrFail(c, "A2A", h.reloadA2A)
	})
}

// DeleteA2AServer 删除 A2A 服务。
func (h *AppConfigHandler) DeleteA2AServer(c *gin.Context) {
	if err := h.service.DeleteA2AServer(c.Request.Context(), c.Param("id")); err != nil {
		response.FromError(c, err)
		return
	}
	if err := reloadOrFail(c, "A2A", h.reloadA2A); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// UpdateA2AServerEnabled 更新 A2A 服务启用状态。
func (h *AppConfigHandler) UpdateA2AServerEnabled(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, apperr.BadRequest(err.Error()))
		return
	}
	if err := h.service.UpdateA2AServerEnabled(c.Request.Context(), c.Param("id"), req.Enabled); err != nil {
		response.FromError(c, err)
		return
	}
	if err := reloadOrFail(c, "A2A", h.reloadA2A); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *AppConfigHandler) reloadMCP(c *gin.Context) error {
	if h.reloader == nil {
		return nil
	}
	cfg, err := h.service.GetMCPConfig(c.Request.Context())
	if err != nil {
		return err
	}
	runtimeCfg := &agent.MCPConfig{}
	if cfg != nil {
		runtimeCfg.Servers = make([]agent.MCPServer, 0, len(cfg.Servers))
		for _, server := range cfg.Servers {
			if !server.Enabled {
				continue
			}
			runtimeCfg.Servers = append(runtimeCfg.Servers, agent.MCPServer{Name: server.ServerName, Command: server.Command, Args: server.Args, Env: server.Env})
		}
	}
	return h.reloader.ReloadMCPConfig(c.Request.Context(), runtimeCfg)
}

func (h *AppConfigHandler) reloadA2A(c *gin.Context) error {
	if h.reloader == nil {
		return nil
	}
	cfg, err := h.service.GetA2AConfig(c.Request.Context())
	if err != nil {
		return err
	}
	runtimeCfg := &agent.A2AConfig{}
	if cfg != nil {
		runtimeCfg.Agents = make([]agent.A2AAgent, 0, len(cfg.Servers))
		for _, server := range cfg.Servers {
			if !server.Enabled {
				continue
			}
			runtimeCfg.Agents = append(runtimeCfg.Agents, agent.A2AAgent{Name: server.ID, URL: server.URL})
		}
	}
	return h.reloader.ReloadA2AConfig(c.Request.Context(), runtimeCfg)
}
