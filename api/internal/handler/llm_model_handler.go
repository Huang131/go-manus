package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/service"
	"github.com/Huang131/go-manus/api/pkg/response"
)

// LLMModelHandler 多模型 HTTP 处理器
type LLMModelHandler struct {
	svc service.LLMModelService
}

// NewLLMModelHandler 创建多模型处理器
func NewLLMModelHandler(svc service.LLMModelService) *LLMModelHandler {
	return &LLMModelHandler{svc: svc}
}

// List 列表
func (h *LLMModelHandler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if items == nil {
		items = []*model.LLMModel{}
	}
	models := make([]*model.LLMModelResponse, 0, len(items))
	for _, item := range items {
		models = append(models, model.NewLLMModelResponse(item))
	}
	response.Success(c, &model.LLMModelListResponse{Models: models})
}

// Get 单个
func (h *LLMModelHandler) Get(c *gin.Context) {
	id := c.Param("id")
	m, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, model.NewLLMModelResponse(m))
}

// GetDefault 取当前默认（agent 启动用）
func (h *LLMModelHandler) GetDefault(c *gin.Context) {
	m, err := h.svc.GetDefaultForAgent(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, model.NewLLMModelResponse(m))
}

// GetRuntimeHealth 读取模型的实时运行健康快照（路由器内存数据，重启归零）。
// 实时状态从这里取。模型无调用记录时返回零值，前端显示"暂无数据"。
func (h *LLMModelHandler) GetRuntimeHealth(c *gin.Context) {
	health, err := h.svc.GetRuntimeHealth(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, health)
}

// Create 新增
func (h *LLMModelHandler) Create(c *gin.Context) {
	var req model.LLMModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, err)
		return
	}
	m, err := h.svc.Create(c.Request.Context(), req.ToModel())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, model.NewLLMModelResponse(m))
}

// Update 更新
func (h *LLMModelHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.LLMModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, err)
		return
	}
	req.ID = id
	m, err := h.svc.Update(c.Request.Context(), req.ToModel())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, model.NewLLMModelResponse(m))
}

// Delete 删除
func (h *LLMModelHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// SetDefault 切换默认
func (h *LLMModelHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.SetDefault(c.Request.Context(), id); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// UnsetDefault 取消默认（agent 启动时会降级到第一个 enabled）
func (h *LLMModelHandler) UnsetDefault(c *gin.Context) {
	if err := h.svc.UnsetDefault(c.Request.Context()); err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, nil)
}

// TestDraft 测试尚未保存的模型配置。
func (h *LLMModelHandler) TestDraft(c *gin.Context) {
	var req model.LLMModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FromError(c, err)
		return
	}
	result, err := h.svc.Test(c.Request.Context(), req.ToModel())
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, result)
}

// TestSaved 测试数据库中已保存的模型配置，避免前端读取密钥。
func (h *LLMModelHandler) TestSaved(c *gin.Context) {
	m, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.FromError(c, err)
		return
	}
	result, err := h.svc.Test(c.Request.Context(), m)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.Success(c, result)
}
