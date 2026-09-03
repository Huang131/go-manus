//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"github.com/bytedance/sonic"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
)

// CleanupLLMModel 清理测试 LLM 模型
func CleanupLLMModel(t *testing.T, id string) {
	ctx, cancel := NewTestContext()
	defer cancel()
	_, err := testDB.Pool.Exec(ctx, "DELETE FROM llm_models WHERE id = $1", id)
	if err != nil {
		t.Logf("清理 llm model %s 失败: %v", id, err)
	}
}

// TestLLMModelAPI_CRUD 端到端测试 LLM 模型 CRUD 流程
func TestLLMModelAPI_CRUD(t *testing.T) {
	// 1. Create
	createBody := map[string]interface{}{
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
		"capabilities": map[string]interface{}{},
		"request_policy": map[string]interface{}{
			"reasoning_mode": "off",
		},
		"cost_policy": map[string]interface{}{
			"input_price_per_m_tokens":  3.0,
			"output_price_per_m_tokens": 15.0,
			"currency":                  "USD",
		},
	}
	bodyJSON, _ := sonic.Marshal(createBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/llm-models", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "create 应返回 200")
	var createResp map[string]interface{}
	sonic.Unmarshal(w.Body.Bytes(), &createResp)
	assertResponseCode(t, createResp, 0)

	created := createResp["data"].(map[string]interface{})
	modelID := created["id"].(string)
	assert.NotEmpty(t, modelID, "创建后应返回 id")
	defer CleanupLLMModel(t, modelID)
	assert.Equal(t, "integration-test-claude", created["name"])
	assert.Equal(t, "claude-3-5-sonnet-20241022", created["model_name"])

	// 2. Get
	getW := httptest.NewRecorder()
	getReq, _ := http.NewRequest("GET", "/api/llm-models/"+modelID, nil)
	testServer.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusOK, getW.Code)
	var getResp map[string]interface{}
	sonic.Unmarshal(getW.Body.Bytes(), &getResp)
	gotModel := getResp["data"].(map[string]interface{})
	assert.Equal(t, "integration-test-claude", gotModel["name"])
	assert.Equal(t, "anthropic", gotModel["provider"])

	// 3. List
	listW := httptest.NewRecorder()
	listReq, _ := http.NewRequest("GET", "/api/llm-models", nil)
	testServer.ServeHTTP(listW, listReq)
	assert.Equal(t, http.StatusOK, listW.Code)
	var listResp map[string]interface{}
	sonic.Unmarshal(listW.Body.Bytes(), &listResp)
	assertResponseCode(t, listResp, 0)
	listData := listResp["data"].(map[string]interface{})
	models := listData["models"].([]interface{})
	assert.GreaterOrEqual(t, len(models), 1, "列表中应至少包含刚创建的模型")

	// 4. Update
	updateBody := map[string]interface{}{
		"name":         "integration-test-claude-updated",
		"provider":     "anthropic",
		"base_url":     "https://api.anthropic.com",
		"model_name":   "claude-3-5-sonnet-20241022",
		"temperature":  0.5,
		"max_tokens":   8192,
		"is_default":   false,
		"is_enabled":   true,
		"capabilities": map[string]interface{}{},
		"request_policy": map[string]interface{}{
			"reasoning_mode": "off",
		},
		"cost_policy": map[string]interface{}{
			"input_price_per_m_tokens":  3.0,
			"output_price_per_m_tokens": 15.0,
			"currency":                  "USD",
		},
	}
	updateJSON, _ := sonic.Marshal(updateBody)
	updW := httptest.NewRecorder()
	updReq, _ := http.NewRequest("PUT", "/api/llm-models/"+modelID, bytes.NewReader(updateJSON))
	updReq.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(updW, updReq)
	assert.Equal(t, http.StatusOK, updW.Code, "update 应返回 200")

	// 验证 update 生效
	regetW := httptest.NewRecorder()
	regetReq, _ := http.NewRequest("GET", "/api/llm-models/"+modelID, nil)
	testServer.ServeHTTP(regetW, regetReq)
	var regetResp map[string]interface{}
	sonic.Unmarshal(regetW.Body.Bytes(), &regetResp)
	regetModel := regetResp["data"].(map[string]interface{})
	assert.Equal(t, "integration-test-claude-updated", regetModel["name"])
	assert.Equal(t, float64(8192), regetModel["max_tokens"], "max_tokens 应被更新")

	// 5. SetDefault
	defW := httptest.NewRecorder()
	defReq, _ := http.NewRequest("POST", "/api/llm-models/"+modelID+"/default", nil)
	testServer.ServeHTTP(defW, defReq)
	assert.Equal(t, http.StatusOK, defW.Code, "set default 应返回 200")

	// 6. GetDefault
	getDefW := httptest.NewRecorder()
	getDefReq, _ := http.NewRequest("GET", "/api/llm-models/default", nil)
	testServer.ServeHTTP(getDefW, getDefReq)
	assert.Equal(t, http.StatusOK, getDefW.Code)
	var getDefResp map[string]interface{}
	sonic.Unmarshal(getDefW.Body.Bytes(), &getDefResp)
	defaultModel := getDefResp["data"].(map[string]interface{})
	assert.Equal(t, modelID, defaultModel["id"], "default 模型应为刚设置的模型")
	assert.Equal(t, true, defaultModel["is_default"])

	// 7. 验证允许删除 default 模型（已开放业务规则：agent 启动会降级到第一个 enabled）
	// 此时 modelID 仍是 step 5 设置的 default
	delDefaultW := httptest.NewRecorder()
	delDefaultReq, _ := http.NewRequest("DELETE", "/api/llm-models/"+modelID, nil)
	testServer.ServeHTTP(delDefaultW, delDefaultReq)
	assert.Equal(t, http.StatusOK, delDefaultW.Code, "允许直接删除 default 模型（agent 启动会降级）")

	// 8. 验证 UnsetDefault 公开 API：创建下一个 default 模型，再取消默认
	// modelID 已删除，此时表中无 default，is_default=true 不会触发唯一索引冲突
	createBody2 := map[string]interface{}{
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
		"capabilities": map[string]interface{}{},
	}
	bodyJSON2, _ := sonic.Marshal(createBody2)
	createW2 := httptest.NewRecorder()
	createReq2, _ := http.NewRequest("POST", "/api/llm-models", bytes.NewReader(bodyJSON2))
	createReq2.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(createW2, createReq2)
	assert.Equal(t, http.StatusOK, createW2.Code)
	var createResp2 map[string]interface{}
	sonic.Unmarshal(createW2.Body.Bytes(), &createResp2)
	modelID2 := createResp2["data"].(map[string]interface{})["id"].(string)
	defer CleanupLLMModel(t, modelID2)

	// 8.1 调用 DELETE /api/llm-models/default 取消默认
	unsetW := httptest.NewRecorder()
	unsetReq, _ := http.NewRequest("DELETE", "/api/llm-models/default", nil)
	testServer.ServeHTTP(unsetW, unsetReq)
	assert.Equal(t, http.StatusOK, unsetW.Code, "unset default 应返回 200")

	// 8.2 验证：UnsetDefault 后再查 default，应降级返回一个 enabled 模型，
	// 且该模型 is_default=false（说明显式 default 已被清除，走的是降级逻辑）
	// 注意：DB 里有 seed 的 GPT-4（sort_order=0），降级会优先返回它，故不断言具体 id。
	getDefW2 := httptest.NewRecorder()
	getDefReq2, _ := http.NewRequest("GET", "/api/llm-models/default", nil)
	testServer.ServeHTTP(getDefW2, getDefReq2)
	assert.Equal(t, http.StatusOK, getDefW2.Code)
	var getDefResp2 map[string]interface{}
	sonic.Unmarshal(getDefW2.Body.Bytes(), &getDefResp2)
	fallback := getDefResp2["data"].(map[string]interface{})
	assert.Equal(t, false, fallback["is_default"], "UnsetDefault 后不应存在显式 default（降级返回的模型 is_default=false）")

	// 9. delete 非 default 模型（modelID2）应成功
	delW := httptest.NewRecorder()
	delReq, _ := http.NewRequest("DELETE", "/api/llm-models/"+modelID2, nil)
	testServer.ServeHTTP(delW, delReq)
	assert.Equal(t, http.StatusOK, delW.Code, "非 default 模型 delete 应返回 200")

	// 10. 验证已删除（modelID2 应已不存在）
	finalGetW := httptest.NewRecorder()
	finalGetReq, _ := http.NewRequest("GET", "/api/llm-models/"+modelID2, nil)
	testServer.ServeHTTP(finalGetW, finalGetReq)
	assert.NotEqual(t, http.StatusOK, finalGetW.Code, "已删除的模型不应能 get 到")
}

// TestLLMModelAPI_CreateInvalidBody 测试非法请求体
func TestLLMModelAPI_CreateInvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/llm-models", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "非法 JSON 应返回非 200")
}

// TestLLMModelAPI_GetNotFound 测试不存在的 id
func TestLLMModelAPI_GetNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/llm-models/00000000-0000-0000-0000-000000000000", nil)
	testServer.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code, "不存在的 id 应返回非 200")
}

// TestLLMModelAPI_GetDefault_NoModel 测试无任何模型时获取 default
// 这是端到端的边界场景：清理后应回退到 "no model available" 错误
func TestLLMModelAPI_GetDefault_NoModel(t *testing.T) {
	// 先确保至少存在一个 enabled 或 default 的模型，否则一直返回错误
	// 此测试不强行破坏全局数据，只验证 API 不会 panic
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/llm-models/default", nil)
	testServer.ServeHTTP(w, req)

	// 有可能 200（存在默认模型）也可能非 200（无模型），但不应 panic 或 500
	assert.NotEqual(t, http.StatusInternalServerError, w.Code, "不应返回 500")
}

// TestFileAPI_FileTableConsistency 测试 file 表内一致性
// 由于 session.files JSONB 与 file 表的同步是个独立问题（见 TestFileAPI_GetSessionFiles），
// 这里单独验证 file 表的增删查能在真实 DB 上一致工作
func TestFileAPI_FileTableConsistency(t *testing.T) {
	// 1. 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)
	var sessionResp map[string]interface{}
	sonic.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 2. 上传文件
	fileContent := []byte("consistency check content")
	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "check.txt", fileContent)

	uploadW := httptest.NewRecorder()
	uploadReq, _ := http.NewRequest("POST", "/api/files", body)
	uploadReq.Header.Set("Content-Type", contentType)
	testServer.ServeHTTP(uploadW, uploadReq)
	assert.Equal(t, http.StatusOK, uploadW.Code)
	var uploadResp map[string]interface{}
	sonic.Unmarshal(uploadW.Body.Bytes(), &uploadResp)
	fileID := uploadResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupFile(t, fileID)

	// 3. 直接查 file 表验证
	ctx, cancel := NewTestContext()
	defer cancel()
	var name, fSessionID string
	var sizeBytes int64
	err := testDB.Pool.QueryRow(ctx,
		"SELECT filename, session_id, size FROM files WHERE id = $1", fileID).
		Scan(&name, &fSessionID, &sizeBytes)
	assert.NoError(t, err, "file 表应能查到")
	assert.Equal(t, "check.txt", name)
	assert.Equal(t, sessionID, fSessionID, "file.session_id 应与上传时一致")
	assert.Equal(t, int64(len(fileContent)), sizeBytes, "size 应等于实际内容长度")
}

// createLLMModelForTest 创建一个 enabled 的非 default 模型，返回 id。
// name 同时用于 name 与 model_name（拼接后缀保证唯一约束不冲突）。
func createLLMModelForTest(t *testing.T, name string) string {
	t.Helper()
	createBody := map[string]interface{}{
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
		"capabilities": map[string]interface{}{},
	}
	bodyJSON, _ := sonic.Marshal(createBody)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/llm-models", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	testServer.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "create "+name+" 应返回 200")
	var resp map[string]interface{}
	sonic.Unmarshal(w.Body.Bytes(), &resp)
	return resp["data"].(map[string]interface{})["id"].(string)
}

// TestLLMModelAPI_UpdateRuntimeHealth 覆盖 Issue #3：运行时健康上报无集成测试。
// handler 未暴露该端点（见 ISSUES_FOUND #3），故走 repo 层直接更新，再用 HTTP GET 验证。
func TestLLMModelAPI_UpdateRuntimeHealth(t *testing.T) {
	modelID := createLLMModelForTest(t, "health-check")
	defer CleanupLLMModel(t, modelID)

	repo := repository.NewLLMModelRepository(testDB)
	ctx, cancel := NewTestContext()
	defer cancel()
	health := model.RuntimeHealth{Status: "degraded", RecentFailures: 3, AverageLatencyMS: 1200}
	assert.NoError(t, repo.UpdateRuntimeHealth(ctx, modelID, health), "repo.UpdateRuntimeHealth 应成功")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/llm-models/"+modelID, nil)
	testServer.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	sonic.Unmarshal(w.Body.Bytes(), &resp)
	got := resp["data"].(map[string]interface{})["runtime_health"].(map[string]interface{})
	assert.Equal(t, "degraded", got["status"])
	assert.Equal(t, float64(3), got["recent_failures"])
	assert.Equal(t, float64(1200), got["average_latency_ms"])
}

// TestLLMModelAPI_SetDefault_Migration 覆盖 Issue #4：验证 default 切换会清掉旧 default。
// 先 set A，再 set B，A.is_default 应自动变 false，B.is_default 应为 true。
func TestLLMModelAPI_SetDefault_Migration(t *testing.T) {
	idA := createLLMModelForTest(t, "migration-a")
	defer CleanupLLMModel(t, idA)
	idB := createLLMModelForTest(t, "migration-b")
	defer CleanupLLMModel(t, idB)

	// set A 为 default
	setAW := httptest.NewRecorder()
	setAReq, _ := http.NewRequest("POST", "/api/llm-models/"+idA+"/default", nil)
	testServer.ServeHTTP(setAW, setAReq)
	assert.Equal(t, http.StatusOK, setAW.Code, "set A default 应返回 200")

	// get A 验证 is_default=true
	getAW := httptest.NewRecorder()
	getAReq, _ := http.NewRequest("GET", "/api/llm-models/"+idA, nil)
	testServer.ServeHTTP(getAW, getAReq)
	var getAResp map[string]interface{}
	sonic.Unmarshal(getAW.Body.Bytes(), &getAResp)
	assert.Equal(t, true, getAResp["data"].(map[string]interface{})["is_default"], "A 应为 default")

	// set B 为 default（切换）
	setBW := httptest.NewRecorder()
	setBReq, _ := http.NewRequest("POST", "/api/llm-models/"+idB+"/default", nil)
	testServer.ServeHTTP(setBW, setBReq)
	assert.Equal(t, http.StatusOK, setBW.Code, "set B default 应返回 200")

	// get B 验证 is_default=true
	getBW := httptest.NewRecorder()
	getBReq, _ := http.NewRequest("GET", "/api/llm-models/"+idB, nil)
	testServer.ServeHTTP(getBW, getBReq)
	var getBResp map[string]interface{}
	sonic.Unmarshal(getBW.Body.Bytes(), &getBResp)
	assert.Equal(t, true, getBResp["data"].(map[string]interface{})["is_default"], "B 应为 default")

	// get A 验证 is_default 已自动清空
	getA2W := httptest.NewRecorder()
	getA2Req, _ := http.NewRequest("GET", "/api/llm-models/"+idA, nil)
	testServer.ServeHTTP(getA2W, getA2Req)
	var getA2Resp map[string]interface{}
	sonic.Unmarshal(getA2W.Body.Bytes(), &getA2Resp)
	assert.Equal(t, false, getA2Resp["data"].(map[string]interface{})["is_default"], "切到 B 后 A 应不再是 default")
}

// TestLLMModelAPI_SetDefault_Concurrent 覆盖 Issue #6：并发 set default 最终只能有 1 个 default。
// 唯一索引 uq_llm_models_default 兜底，事务竞争下也保证最终一致性。
func TestLLMModelAPI_SetDefault_Concurrent(t *testing.T) {
	const n = 10
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := createLLMModelForTest(t, fmt.Sprintf("concurrent-%d", i))
		ids = append(ids, id)
		defer CleanupLLMModel(t, id)
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/llm-models/"+id+"/default", nil)
			testServer.ServeHTTP(w, req)
			// 并发下可能有个别因唯一约束竞争返回错误，属预期，不断言单次结果
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

// 防止 time 包未使用
var _ = time.Second
