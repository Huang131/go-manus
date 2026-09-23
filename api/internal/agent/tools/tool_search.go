package tools

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/search"
)

// SearchTool 搜索工具
type SearchTool struct {
	searchEngine search.SearchEngine
	limit        int
}

// NewSearchTool 创建搜索工具
func NewSearchTool(searchEngine search.SearchEngine, limit int) *SearchTool {
	return &SearchTool{searchEngine: searchEngine, limit: limit}
}

// Name 返回工具名称
func (t *SearchTool) Name() string {
	return ToolNameSearch
}

// Description 返回工具描述
func (t *SearchTool) Description() string {
	return "搜索网络信息。"
}

// Parameters 返回工具参数定义
func (t *SearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "搜索关键词",
			},
			"date_range": map[string]interface{}{
				"type":        "string",
				"description": "日期范围: d (今天), w (本周), m (本月), y (今年)",
			},
		},
		"required": []string{"query"},
	}
}

// ReadOnly 搜索只读取外部信息，不修改本地或远端状态。
func (t *SearchTool) ReadOnly() bool {
	return true
}

// Invoke 调用工具
func (t *SearchTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	query := ""
	if v, ok := params["query"].(string); ok {
		query = v
	}

	var dateRange *string
	if v, ok := params["date_range"].(string); ok && v != "" {
		dateRange = &v
	}

	return t.searchEngine.Invoke(ctx, query, dateRange, t.limit)
}
