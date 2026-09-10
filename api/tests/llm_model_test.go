//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/stretchr/testify/assert"
)

// CleanupLLMModel 清理测试 LLM 模型
func CleanupLLMModel(t *testing.T, id string) {
	t.Helper()
	ctx, cancel := NewTestContext()
	defer cancel()
	_, err := testDB.Pool.Exec(ctx, "DELETE FROM llm_models WHERE id = $1", id)
	if err != nil {
		t.Logf("清理 llm model %s 失败: %v", id, err)
	}
}

// createLLMModelForTest 创建一个 enabled 的非 default 模型，返回 id
func createLLMModelForTest(t *testing.T, name string) string {
	t.Helper()
	createBody := map[string]any{
		"name":         name,
		"provider":     "openai",
		"base_url":     "https://api.openai.com/v1",
		"api_key":      "test-key",
		"model_name":   name + "-model",
		"temperature":  0.7,
		"max_tokens":   4096,
		"is_default":   false,
		"is_enabled":   true,
		"sort_order":   100,
		"capabilities": map[string]any{},
	}
	w := postJSON(t, "/api/llm-models", createBody)
	assert.Equal(t, http.StatusOK, w.Code, "create "+name+" 应返回 200")

	data := parseResponseDataAsMap(t, w)
	return data["id"].(string)
}

// TestLLMModelAPI_CRUD 端到端测试 LLM 模型 CRUD 流程
func TestLLMModelAPI_CRUD(t *testing.T) {
	// 1. Create
	createBody := map[string]any{
		"name":         "integration-test-claude",
		"provider":     "anthropic",
		"base_url":     "https://api.anthropic.com",
		"api_key":      "test-key",
		"model_name":   "claude-3-5-sonnet-20241022",
		"temperature":  0.7,
		"max_tokens":   4096,
		"tags":         []string{"tools", "long_ctx"},
		"is_default":   false,
		"is_enabled":   true,
		"sort_order":   1,
		"capabilities": map[string]any{},
		"request_policy": map[string]any{
			"reasoning_mode": "off",
		},
		"cost_policy": map[string]any{
			"input_price_per_m_tokens":  3.0,
			"output_price_per_m_tokens": 15.0,
			"currency":                  "USD",
		},
	}
	w := postJSON(t, "/api/llm-models", createBody)
	assert.Equal(t, http.StatusOK, w.Code, "create 应返回 200")

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	created := parseResponseDataAsMap(t, w)
	modelID := created["id"].(string)
	defer CleanupLLMModel(t, modelID)
	assert.NotEmpty(t, modelID, "创建后应返回 id")
	assert.Equal(t, "integration-test-claude", created["name"])
	assert.Equal(t, "claude-3-5-sonnet-20241022", created["model_name"])

	// 2. Get
	w = getJSON(t, "/api/llm-models/"+modelID)
	assert.Equal(t, http.StatusOK, w.Code)

	getResp := parseResponse(t, w)
	assert.Equal(t, 0, getResp.Code)
	gotModel := parseResponseDataAsMap(t, w)
	assert.Equal(t, "integration-test-claude", gotModel["name"])
	assert.Equal(t, "anthropic", gotModel["provider"])

	// 3. List
	listW := getJSON(t, "/api/llm-models")
	assert.Equal(t, http.StatusOK, listW.Code)

	listResp := parseResponse(t, listW)
	assert.Equal(t, 0, listResp.Code)
	listData := parseResponseDataAsMap(t, listW)
	models := listData["models"].([]any)
	assert.GreaterOrEqual(t, len(models), 1, "列表中应至少包含刚创建的模型")

	// 4. Update
	updateBody := map[string]any{
		"name":         "integration-test-claude-updated",
		"provider":     "anthropic",
		"base_url":     "https://api.anthropic.com",
		"model_name":   "claude-3-5-sonnet-20241022",
		"temperature":  0.5,
		"max_tokens":   8192,
		"is_default":   false,
		"is_enabled":   true,
		"capabilities": map[string]any{},
		"request_policy": map[string]any{
			"reasoning_mode": "off",
		},
		"cost_policy": map[string]any{
			"input_price_per_m_tokens":  3.0,
			"output_price_per_m_tokens": 15.0,
			"currency":                  "USD",
		},
	}
	w = putJSON(t, "/api/llm-models/"+modelID, updateBody)
	assert.Equal(t, http.StatusOK, w.Code, "update 应返回 200")

	// 验证 update 生效
	w = getJSON(t, "/api/llm-models/"+modelID)
	_ = parseResponse(t, w)
	regetModel := parseResponseDataAsMap(t, w)
	assert.Equal(t, "integration-test-claude-updated", regetModel["name"])
	assert.Equal(t, float64(8192), regetModel["max_tokens"], "max_tokens 应被更新")

	// 5. SetDefault
	w = postJSON(t, "/api/llm-models/"+modelID+"/default", nil)
	assert.Equal(t, http.StatusOK, w.Code, "set default 应返回 200")

	// 6. GetDefault
	getDefW := getJSON(t, "/api/llm-models/default")
	assert.Equal(t, http.StatusOK, getDefW.Code)

	_ = parseResponse(t, getDefW)
	defaultModel := parseResponseDataAsMap(t, getDefW)
	assert.Equal(t, modelID, defaultModel["id"], "default 模型应为刚设置的模型")
	assert.Equal(t, true, defaultModel["is_default"])

	// 7. 验证允许删除 default 模型（已开放业务规则：agent 启动会降级到第一个 enabled）
	// 此时 modelID 仍是 step 5 设置的 default
	w = deleteRequest(t, "/api/llm-models/"+modelID)
	assert.Equal(t, http.StatusOK, w.Code, "允许直接删除 default 模型（agent 启动会降级）")

	// 8. 验证 UnsetDefault 公开 API：创建下一个 default 模型，再取消默认
	// modelID 已删除，此时表中无 default，is_default=true 不会触发唯一索引冲突
	createBody2 := map[string]any{
		"name":         "integration-test-unset",
		"provider":     "openai",
		"base_url":     "https://api.openai.com/v1",
		"api_key":      "test-key-2",
		"model_name":   "gpt-4o-mini",
		"temperature":  0.7,
		"max_tokens":   4096,
		"is_default":   true,
		"is_enabled":   true,
		"sort_order":   2,
		"capabilities": map[string]any{},
	}
	createW2 := postJSON(t, "/api/llm-models", createBody2)
	assert.Equal(t, http.StatusOK, createW2.Code)

	createResp2 := parseResponse(t, createW2)
	modelID2 := createResp2.Data.(map[string]any)["id"].(string)
	defer CleanupLLMModel(t, modelID2)

	// 8.1 调用 DELETE /api/llm-models/default 取消默认
	unsetW := deleteRequest(t, "/api/llm-models/default")
	assert.Equal(t, http.StatusOK, unsetW.Code, "unset default 应返回 200")

	// 8.2 验证：UnsetDefault 后再查 default，应降级返回一个 enabled 模型，
	// 且该模型 is_default=false（说明显式 default 已被清除，走的是降级逻辑）
	// 注意：DB 里有 seed 的 GPT-4（sort_order=0），降级会优先返回它，故不断言具体 id。
	getDefW2 := getJSON(t, "/api/llm-models/default")
	assert.Equal(t, http.StatusOK, getDefW2.Code)

	getDefResp2 := parseResponse(t, getDefW2)
	fallback := getDefResp2.Data.(map[string]any)
	assert.Equal(t, false, fallback["is_default"], "UnsetDefault 后不应存在显式 default（降级返回的模型 is_default=false）")

	// 9. delete 非 default 模型（modelID2）应成功
	w = deleteRequest(t, "/api/llm-models/"+modelID2)
	assert.Equal(t, http.StatusOK, w.Code, "非 default 模型 delete 应返回 200")

	// 10. 验证已删除（modelID2 应已不存在）
	finalGetW := getJSON(t, "/api/llm-models/"+modelID2)
	assert.NotEqual(t, http.StatusOK, finalGetW.Code, "已删除的模型不应能 get 到")
}

// TestLLMModelAPI_CreateInvalidBody 测试非法请求体
func TestLLMModelAPI_CreateInvalidBody(t *testing.T) {
	// 直接使用 doRequest 发送非法 JSON
	w := doRequest(t, "POST", "/api/llm-models", []byte("not json"), "application/json")
	assert.NotEqual(t, http.StatusOK, w.Code, "非法 JSON 应返回非 200")
}

// TestLLMModelAPI_GetNotFound 测试不存在的 id
func TestLLMModelAPI_GetNotFound(t *testing.T) {
	w := getJSON(t, "/api/llm-models/00000000-0000-0000-0000-000000000000")
	assert.NotEqual(t, http.StatusOK, w.Code, "不存在的 id 应返回非 200")
}

// TestLLMModelAPI_GetDefault_NoModel 测试无任何模型时获取 default
// 这是端到端的边界场景：清理后应回退到 "no model available" 错误
func TestLLMModelAPI_GetDefault_NoModel(t *testing.T) {
	// 先确保至少存在一个 enabled 或 default 的模型，否则一直返回错误
	// 此测试不强行破坏全局数据，只验证 API 不会 panic
	w := getJSON(t, "/api/llm-models/default")

	// 有可能 200（存在默认模型）也可能非 200（无模型），但不应 panic 或 500
	assert.NotEqual(t, http.StatusInternalServerError, w.Code, "不应返回 500")
}

// TestLLMModelAPI_UpdateRuntimeHealth 测试运行时健康上报
// handler 未暴露该端点，故走 repo 层直接更新，再用 HTTP GET 验证
func TestLLMModelAPI_UpdateRuntimeHealth(t *testing.T) {
	modelID := createLLMModelForTest(t, "health-check")
	defer CleanupLLMModel(t, modelID)

	repo := repository.NewLLMModelRepository(testDB)
	ctx, cancel := NewTestContext()
	defer cancel()
	health := model.RuntimeHealth{Status: "degraded", RecentFailures: 3, AverageLatencyMS: 1200}
	assert.NoError(t, repo.UpdateRuntimeHealth(ctx, modelID, health), "repo.UpdateRuntimeHealth 应成功")

	w := getJSON(t, "/api/llm-models/"+modelID)
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	got := resp.Data.(map[string]any)["runtime_health"].(map[string]any)
	assert.Equal(t, "degraded", got["status"])
	assert.Equal(t, float64(3), got["recent_failures"])
	assert.Equal(t, float64(1200), got["average_latency_ms"])
}

// TestLLMModelAPI_SetDefault_Migration 测试 default 切换会清掉旧 default
// 先 set A，再 set B，A.is_default 应自动变 false，B.is_default 应为 true
func TestLLMModelAPI_SetDefault_Migration(t *testing.T) {
	idA := createLLMModelForTest(t, "migration-a")
	defer CleanupLLMModel(t, idA)
	idB := createLLMModelForTest(t, "migration-b")
	defer CleanupLLMModel(t, idB)

	// set A 为 default
	w := postJSON(t, "/api/llm-models/"+idA+"/default", nil)
	assert.Equal(t, http.StatusOK, w.Code, "set A default 应返回 200")

	// get A 验证 is_default=true
	getAW := getJSON(t, "/api/llm-models/"+idA)
	getAResp := parseResponse(t, getAW)
	assert.Equal(t, true, getAResp.Data.(map[string]any)["is_default"], "A 应为 default")

	// set B 为 default（切换）
	w = postJSON(t, "/api/llm-models/"+idB+"/default", nil)
	assert.Equal(t, http.StatusOK, w.Code, "set B default 应返回 200")

	// get B 验证 is_default=true
	getBW := getJSON(t, "/api/llm-models/"+idB)
	getBResp := parseResponse(t, getBW)
	assert.Equal(t, true, getBResp.Data.(map[string]any)["is_default"], "B 应为 default")

	// get A 验证 is_default 已自动清空
	getA2W := getJSON(t, "/api/llm-models/"+idA)
	getA2Resp := parseResponse(t, getA2W)
	assert.Equal(t, false, getA2Resp.Data.(map[string]any)["is_default"], "切到 B 后 A 应不再是 default")
}

// TestLLMModelAPI_SetDefault_Concurrent 测试并发 set default 最终只能有 1 个 default
// 唯一索引 uq_llm_models_default 兜底，事务竞争下也保证最终一致性
func TestLLMModelAPI_SetDefault_Concurrent(t *testing.T) {
	const n = 10
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := createLLMModelForTest(t, fmt.Sprintf("concurrent-%d", i))
		ids = append(ids, id)
	}
	// 在并发 set default 完成后统一清理，避免 defer 在循环里延后执行的坑
	defer func() {
		for _, id := range ids {
			CleanupLLMModel(t, id)
		}
	}()

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(modelID string) {
			defer wg.Done()
			w := postJSON(t, "/api/llm-models/"+modelID+"/default", nil)
			// 竞争失败只能映射为业务冲突，不能把数据库错误暴露成 500。
			if w.Code != http.StatusOK && w.Code != http.StatusConflict {
				t.Errorf("unexpected response for id=%s: status=%d, body=%s", modelID, w.Code, w.Body.String())
			}
		}(id)
	}
	wg.Wait()

	// 统计最终 is_default=true 的数量
	ctx, cancel := NewTestContext()
	defer cancel()
	var defaultCount int
	err := testDB.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM llm_models WHERE is_default = TRUE").Scan(&defaultCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, defaultCount, "并发 set default 后只能有 1 个 is_default=true")
}

// TestLLMModelAPI_List 测试模型列表
func TestLLMModelAPI_List(t *testing.T) {
	// 创建一个模型用于测试
	modelID := createLLMModelForTest(t, "list-test")
	defer CleanupLLMModel(t, modelID)

	// 获取列表
	w := getJSON(t, "/api/llm-models")
	assert.Equal(t, http.StatusOK, w.Code)

	resp := parseResponse(t, w)
	assert.Equal(t, 0, resp.Code)

	data := resp.Data.(map[string]any)
	models := data["models"].([]any)
	assert.GreaterOrEqual(t, len(models), 1, "列表应至少包含刚创建的模型")
}

// 防止 fmt 包未使用
var _ = fmt.Sprintf
