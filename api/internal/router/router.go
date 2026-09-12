package router

import (
	"github.com/gin-gonic/gin"

	"github.com/Huang131/go-manus/api/internal/handler"
)

// Handlers 所有 Handler 的容器
type Handlers struct {
	Session   *handler.SessionHandler
	File      *handler.FileHandler
	Status    *handler.StatusHandler
	AppConfig *handler.AppConfigHandler
	LLMModel  *handler.LLMModelHandler
}

// SetupRoutes 设置所有路由
func SetupRoutes(engine *gin.Engine, h *Handlers) {
	// 健康检查
	engine.GET("/health", h.Status.Health)

	// API 路由组
	api := engine.Group("/api")
	{
		// 状态模块
		api.GET("/status", h.Status.Health)

		// 会话模块
		sessions := api.Group("/sessions")
		{
			sessions.POST("", h.Session.Create)
			sessions.GET("/stream", h.Session.Stream)  // SSE 流式推送所有会话
			sessions.POST("/stream", h.Session.Stream) // 同时支持 POST（前端 createSSEStream 默认用 POST）
			sessions.GET("", h.Session.List)
			sessions.GET("/:id", h.Session.Get)
			sessions.POST("/:id/delete", h.Session.Delete) // 对齐原项目
			sessions.POST("/:id/rename", h.Session.RenameSession)
			sessions.POST("/:id/clear-unread-message-count", h.Session.ClearUnread) // 对齐原项目
			sessions.POST("/:id/chat", h.Session.Chat)
			sessions.POST("/:id/stop", h.Session.Stop)
			sessions.GET("/:id/files", h.Session.GetFiles)
			// 对齐原项目：/sessions/:id/file (POST 读沙箱文件) 和 /sessions/:id/shell (POST 读 shell 输出)
			sessions.POST("/:id/file", h.Session.ReadFile)
			sessions.POST("/:id/shell", h.Session.ReadShell)
			// VNC WebSocket 代理
			sessions.GET("/:id/vnc", handler.VNCProxy(h.Session.Service()))
		}

		// 文件模块
		files := api.Group("/files")
		{
			files.POST("", h.File.Upload) // POST 需要 session_id 参数
			files.GET("/:id", h.File.GetInfo)
			files.GET("/:id/download", h.File.Download)
		}

		// 配置模块（对齐原项目：/app-config/mcp-servers、/a2a-servers）
		appConfig := api.Group("/app-config")
		{
			appConfig.GET("/llm", h.AppConfig.GetLLMConfig)
			appConfig.POST("/llm", h.AppConfig.UpdateLLMConfig)
			appConfig.GET("/agent", h.AppConfig.GetAgentConfig)
			appConfig.POST("/agent", h.AppConfig.UpdateAgentConfig)
			appConfig.GET("/mcp-servers", h.AppConfig.GetMCPConfig)
			appConfig.POST("/mcp-servers", h.AppConfig.UpdateMCPConfig)
			appConfig.POST("/mcp-servers/:server_name/delete", h.AppConfig.DeleteMCPServer)
			appConfig.GET("/a2a-servers", h.AppConfig.GetA2AConfig)
			appConfig.POST("/a2a-servers", h.AppConfig.UpdateA2AConfig)
		}

		// 多模型管理
		llmModels := api.Group("/llm-models")
		{
			llmModels.GET("", h.LLMModel.List)
			llmModels.GET("/default", h.LLMModel.GetDefault)
			llmModels.DELETE("/default", h.LLMModel.UnsetDefault)
			llmModels.POST("", h.LLMModel.Create)
			llmModels.GET("/:id", h.LLMModel.Get)
			llmModels.PUT("/:id", h.LLMModel.Update)
			llmModels.DELETE("/:id", h.LLMModel.Delete)
			llmModels.POST("/:id/default", h.LLMModel.SetDefault)
		}
	}
}
