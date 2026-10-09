package tools

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

type searchEngineStub struct {
	limit int
	query string
}

func (s *searchEngineStub) Invoke(_ context.Context, query string, _ *string, limit int) (*model.ToolResult, error) {
	s.limit = limit
	s.query = query
	return model.NewToolResult("ok"), nil
}

func TestSearchToolRejectsEmptyQuery(t *testing.T) {
	search := &searchEngineStub{}
	result, err := NewSearchTool(search, 6).Invoke(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Invoke() error = %v, want nil", err)
	}
	if result == nil || result.Success {
		t.Fatalf("Invoke() = %+v, want validation error", result)
	}
	if search.query != "" {
		t.Fatalf("search query = %q, want no backend call", search.query)
	}
}

func TestSearchToolForwardsConfiguredLimit(t *testing.T) {
	search := &searchEngineStub{}
	tool := NewSearchTool(search, 6)

	if _, err := tool.Invoke(context.Background(), map[string]interface{}{"query": "go"}); err != nil {
		t.Fatal(err)
	}
	if search.limit != 6 {
		t.Fatalf("search limit = %d, want 6", search.limit)
	}
}

func TestToolProviderReloadSearchLimitAffectsNewTools(t *testing.T) {
	search := &searchEngineStub{}
	provider := NewToolProvider(context.Background(), nil, nil, search, nil, nil)

	var searchTool *SearchTool
	for _, tool := range provider.Tools(9) {
		if candidate, ok := tool.(*SearchTool); ok {
			searchTool = candidate
			break
		}
	}
	if searchTool == nil {
		t.Fatal("search tool was not registered")
	}
	if _, err := searchTool.Invoke(context.Background(), map[string]interface{}{"query": "go"}); err != nil {
		t.Fatal(err)
	}
	if search.limit != 9 {
		t.Fatalf("search limit = %d, want 9", search.limit)
	}
}
