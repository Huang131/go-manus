package agent

import (
	"context"

	"github.com/mooc-manus/go-manus/api/internal/external"
	"github.com/mooc-manus/go-manus/api/internal/model"
)

// SearchTool 搜索工具
type SearchTool struct {
	searchEngine external.SearchEngine
}

// NewSearchTool 创建搜索工具
func NewSearchTool(searchEngine external.SearchEngine) *SearchTool {
	return &SearchTool{searchEngine: searchEngine}
}

// Name 返回工具名称
func (t *SearchTool) Name() string {
	return "search"
}

// Description 返回工具描述
func (t *SearchTool) Description() string {
	return "用于搜索互联网信息。可以搜索关键词获取相关网页结果。"
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

	return t.searchEngine.Invoke(ctx, query, dateRange)
}
