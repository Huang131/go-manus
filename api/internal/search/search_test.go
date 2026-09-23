//go:build external

package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/config"
)

// loadSearchKeys 从配置加载仅供 external smoke 使用的搜索 API key。
func loadSearchKeys(t *testing.T) (tavilyKey, bochaKey string) {
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

// requireExternalSearchTests 防止直接执行 external build tag 时意外调用真实服务。
func requireExternalSearchTests(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_EXTERNAL_TESTS") != "1" {
		t.Skip("skipping search external smoke: RUN_EXTERNAL_TESTS=1 is required")
	}
}

func requireTavilyKey(t *testing.T) string {
	t.Helper()
	requireExternalSearchTests(t)
	tavilyKey, _ := loadSearchKeys(t)
	if tavilyKey == "" {
		t.Skip("skipping Tavily external smoke: tavily_api_key is not configured")
	}
	return tavilyKey
}

func requireBochaKey(t *testing.T) string {
	t.Helper()
	requireExternalSearchTests(t)
	_, bochaKey := loadSearchKeys(t)
	if bochaKey == "" {
		t.Skip("skipping Bocha external smoke: bocha_api_key is not configured")
	}
	return bochaKey
}

func requireFallbackKeys(t *testing.T) (tavilyKey, bochaKey string) {
	t.Helper()
	requireExternalSearchTests(t)
	tavilyKey, bochaKey = loadSearchKeys(t)
	if tavilyKey == "" || bochaKey == "" {
		t.Skip("skipping fallback external smoke: tavily_api_key and bocha_api_key are required")
	}
	return tavilyKey, bochaKey
}

// TestTavilyLive 真实验证 Tavily API
func TestTavilyLive(t *testing.T) {
	tavilyKey := requireTavilyKey(t)

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
	bochaKey := requireBochaKey(t)

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
	tavilyKey, bochaKey := requireFallbackKeys(t)

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
	bochaKey := requireBochaKey(t)

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
	bochaKey := requireBochaKey(t)
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
