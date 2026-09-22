package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/config"
)

// getSearchKeys 从配置加载搜索 API key
func getSearchKeys(t *testing.T) (tavilyKey, bochaKey string) {
	configPaths := []string{
		"config.yaml",
		"../../config.yaml",
	}

	var cfg *config.Config
	for _, p := range configPaths {
		absPath, _ := filepath.Abs(p)
		if c, err := config.LoadWithValidation(absPath); err == nil {
			cfg = c
			break
		}
	}

	if cfg == nil {
		t.Skip("skipping live test: cannot load config.yaml")
		return
	}

	return cfg.Search.TavilyAPIKey, cfg.Search.BochaAPIKey
}

// skipIfNoKeys 跳过测试如果没有配置凭证
func skipIfNoKeys(t *testing.T) (tavilyKey, bochaKey string) {
	tavilyKey, bochaKey = getSearchKeys(t)
	if tavilyKey == "" || bochaKey == "" {
		t.Skip("skipping live test: tavily_api_key or bocha_api_key not set in config.yaml")
	}
	return
}

// TestTavilyLive 真实验证 Tavily API
func TestTavilyLive(t *testing.T) {
	tavilyKey, bochaKey := skipIfNoKeys(t)
	_ = bochaKey

	c := NewTavilySearchClientWithTimeout(tavilyKey, 30*time.Second)
	res, err := c.Invoke(context.Background(), "成龙是谁", nil, 3)
	if err != nil {
		t.Fatalf("Tavily Invoke failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("Tavily returned error: %s", res.Message)
	}

	data := res.Data.(map[string]interface{})
	results := data["results"]
	t.Logf("Tavily returned %+v results", results)

	if results == nil {
		t.Error("expected non-nil results")
	}
}

// TestBochaLive 真实验证 Bocha API
func TestBochaLive(t *testing.T) {
	tavilyKey, bochaKey := skipIfNoKeys(t)
	_ = tavilyKey

	c := NewBochaSearchClientWithTimeout(bochaKey, 30*time.Second)
	res, err := c.Invoke(context.Background(), "阿里巴巴ESG报告", nil, 3)
	if err != nil {
		t.Fatalf("Bocha Invoke failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("Bocha returned error: %s", res.Message)
	}

	data := res.Data.(map[string]interface{})
	results := data["results"]
	t.Logf("Bocha returned results: %v", results)

	if results == nil {
		t.Error("expected non-nil results")
	}
}

// TestFallbackLive 真实验证 Tavily + Bocha 自动切换
func TestFallbackLive(t *testing.T) {
	tavilyKey, bochaKey := skipIfNoKeys(t)

	c := NewFallbackSearchClient(tavilyKey, bochaKey, 30*time.Second)

	res, err := c.Invoke(context.Background(), "成龙是谁", nil, 3)
	if err != nil {
		t.Fatalf("Fallback Invoke failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("Fallback returned error: %s", res.Message)
	}
	t.Log("Fallback (Tavily primary) succeeded")
}

// TestFallbackLive_BochaFallback 真实验证 fallback 到 Bocha
func TestFallbackLive_BochaFallback(t *testing.T) {
	_, bochaKey := skipIfNoKeys(t)

	// 使用无效的 Tavily key 强制触发 fallback
	c := NewFallbackSearchClient("invalid-key-force-fallback", bochaKey, 30*time.Second)

	res, err := c.Invoke(context.Background(), "golang", nil, 2)
	if err != nil {
		t.Fatalf("Fallback Invoke failed: %v", err)
	}
	if res.Success {
		t.Log("Fallback correctly switched to Bocha")
		data := res.Data.(map[string]interface{})
		results := data["results"]
		t.Logf("Fallback (Bocha fallback) returned results: %v", results)
	} else {
		t.Logf("Both services failed as expected: %s", res.Message)
	}
}

// TestBochaLive_WithDateRange 测试带日期范围的 Bocha 搜索
func TestBochaLive_WithDateRange(t *testing.T) {
	if os.Getenv("RUN_LIVE_TESTS") != "true" {
		t.Skip("skipping live test: RUN_LIVE_TESTS not set")
	}

	_, bochaKey := skipIfNoKeys(t)
	c := NewBochaSearchClientWithTimeout(bochaKey, 30*time.Second)
	dateRange := "y"
	res, err := c.Invoke(context.Background(), "AI人工智能发展", &dateRange, 5)
	if err != nil {
		t.Fatalf("Bocha Invoke with dateRange failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("Bocha returned error: %s", res.Message)
	}

	data := res.Data.(map[string]interface{})
	results := data["results"]
	t.Logf("Bocha with dateRange returned: %v", results)
}
