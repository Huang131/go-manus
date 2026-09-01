package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/service"
	"github.com/mooc-manus/go-manus/api/pkg/response"
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
		response.Error(c, err.Error())
		return
	}
	if items == nil {
		items = []*model.LLMModel{}
	}
	response.Success(c, &model.LLMModelListResponse{Models: items})
}

// Get 单个
func (h *LLMModelHandler) Get(c *gin.Context) {
	id := c.Param("id")
	m, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, m)
}

// GetDefault 取当前默认（agent 启动用）
func (h *LLMModelHandler) GetDefault(c *gin.Context) {
	m, err := h.svc.GetByID(c.Request.Context(), "")
	_ = m
	_ = err
	// 直接走 service 接口
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	for _, it := range items {
		if it.IsDefault {
			response.Success(c, it)
			return
		}
	}
	// 降级：取第一个 enabled
	for _, it := range items {
		if it.IsEnabled {
			response.Success(c, it)
			return
		}
	}
	response.Error(c, "no model available")
}

// Create 新增
func (h *LLMModelHandler) Create(c *gin.Context) {
	var req model.LLMModel
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err.Error())
		return
	}
	m, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, m)
}

// Update 更新
func (h *LLMModelHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.LLMModel
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err.Error())
		return
	}
	req.ID = id
	m, err := h.svc.Update(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, m)
}

// Delete 删除
func (h *LLMModelHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, nil)
}

// SetDefault 切换默认
func (h *LLMModelHandler) SetDefault(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.SetDefault(c.Request.Context(), id); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, nil)
}
