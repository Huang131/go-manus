package search

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

// TestSearchEngineInterface 测试 SearchEngine 接口定义
func TestSearchEngineInterface(t *testing.T) {
	var _ SearchEngine = (*MockSearchEngine)(nil)
}

// MockSearchEngine 用于测试的 SearchEngine Mock 实现
type MockSearchEngine struct {
	searchResult *model.ToolResult
	searchErr    error
}

func (m *MockSearchEngine) Invoke(ctx context.Context, query string, dateRange *string, limit int) (*model.ToolResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	if m.searchResult != nil {
		return m.searchResult, nil
	}
	return &model.ToolResult{
		Success: true,
		Message: "Search results for: " + query,
	}, nil
}

// TestSearchEngine_Invoke 测试搜索引擎调用
func TestSearchEngine_Invoke(t *testing.T) {
	search := &MockSearchEngine{}

	result, err := search.Invoke(context.Background(), "test query", nil, 10)
	if err != nil {
		t.Errorf("Invoke() error = %v", err)
	}

	if result == nil {
		t.Error("Invoke() should return a result")
	}

	if !result.Success {
		t.Error("Invoke() should succeed")
	}
}

// TestSearchEngine_Invoke_WithError 测试搜索引擎错误
func TestSearchEngine_Invoke_WithError(t *testing.T) {
	search := &MockSearchEngine{
		searchErr: context.DeadlineExceeded,
	}

	result, err := search.Invoke(context.Background(), "test query", nil, 10)
	if err == nil {
		t.Error("Invoke() should return an error")
	}

	if result != nil {
		t.Error("Invoke() with error should return nil result")
	}
}
