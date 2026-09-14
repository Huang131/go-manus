//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// senseNovaAPIKey 在本地运行集成测试时替换为真实密钥。
// 提交代码时保持占位值，避免把密钥写入仓库或测试日志。
var senseNovaAPIKey = "replace-with-local-sensenova-key"

// senseNovaRunImageTests 默认关闭，改为 true 后才会调用图像生成接口并消耗额度。
var senseNovaRunImageTests = false

const senseNovaBaseURL = "https://token.sensenova.cn/v1"

var senseNovaChatModels = []string{
	"sensenova-6.8-flash-lite",
	"deepseek-v4-pro",
	"deepseek-v4-flash",
	"glm-5.2",
	"kimi-k3",
}

type senseNovaChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			Reasoning        string `json:"reasoning"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func senseNovaReasoningEffort(modelName string) string {
	if modelName == "kimi-k3" {
		return "low"
	}
	return "none"
}

func requireSenseNovaKey(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(senseNovaAPIKey) == "" || strings.HasPrefix(senseNovaAPIKey, "replace-with-") {
		t.Skip("将 senseNovaAPIKey 替换为本地密钥后运行 SenseNova 集成测试")
	}
}

func senseNovaRequest(t *testing.T, path string, payload any) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("序列化 SenseNova 请求失败: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, senseNovaBaseURL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("创建 SenseNova 请求失败: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+senseNovaAPIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SenseNova 请求失败: %v", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取 SenseNova 响应失败: %v", err)
	}
	return resp.StatusCode, respBody
}

func TestSenseNova_ListModels(t *testing.T) {
	requireSenseNovaKey(t)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, senseNovaBaseURL+"/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+senseNovaAPIKey)
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("请求模型列表失败: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("模型列表返回 HTTP %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("模型列表不是合法 JSON: %v", err)
	}
	if len(result.Data) == 0 {
		t.Fatal("模型列表为空")
	}
	t.Logf("SenseNova 返回 %d 个模型", len(result.Data))
}

func TestSenseNova_ChatModelsAreOpenAICompatible(t *testing.T) {
	requireSenseNovaKey(t)
	for _, modelName := range senseNovaChatModels {
		modelName := modelName
		t.Run(modelName, func(t *testing.T) {
			status, body := senseNovaRequest(t, "/chat/completions", map[string]any{
				"model":            modelName,
				"messages":         []map[string]string{{"role": "user", "content": "请只回复数字 8"}},
				"stream":           false,
				"reasoning_effort": senseNovaReasoningEffort(modelName),
			})
			if status == http.StatusTooManyRequests {
				t.Skipf("模型触发限流（429），不判定为协议不兼容")
			}
			if status != http.StatusOK {
				t.Fatalf("模型返回 HTTP %d: %s", status, string(body))
			}
			var result senseNovaChatResponse
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("响应不是合法 JSON: %v", err)
			}
			if len(result.Choices) == 0 {
				t.Fatalf("响应缺少 choices: %s", string(body))
			}
			choice := result.Choices[0]
			if choice.Message.Role != "assistant" {
				t.Errorf("message.role = %q, want assistant", choice.Message.Role)
			}
			if strings.TrimSpace(choice.Message.Content) == "" {
				t.Errorf("message.content 为空，finish_reason=%s（reasoning 字段长度=%d）", choice.FinishReason, len(choice.Message.Reasoning)+len(choice.Message.ReasoningContent))
			}
			t.Logf("model=%s response_model=%s finish_reason=%s content=%q", modelName, result.Model, choice.FinishReason, choice.Message.Content)
		})
	}
}

func TestSenseNova_StructuredJSONForPlannerCompatibleModels(t *testing.T) {
	requireSenseNovaKey(t)
	for _, modelName := range []string{"sensenova-6.8-flash-lite", "deepseek-v4-pro", "deepseek-v4-flash", "glm-5.2"} {
		modelName := modelName
		t.Run(modelName, func(t *testing.T) {
			status, body := senseNovaRequest(t, "/chat/completions", map[string]any{
				"model":            modelName,
				"messages":         []map[string]string{{"role": "user", "content": "请严格输出 JSON：{\"answer\":8}"}},
				"response_format":  map[string]string{"type": "json_object"},
				"reasoning_effort": "none",
			})
			if status == http.StatusTooManyRequests {
				t.Skip("触发限流")
			}
			if status != http.StatusOK {
				t.Fatalf("模型返回 HTTP %d: %s", status, string(body))
			}
			var result senseNovaChatResponse
			if err := json.Unmarshal(body, &result); err != nil || len(result.Choices) == 0 {
				t.Fatalf("响应格式不兼容: %v, body=%s", err, string(body))
			}
			content := result.Choices[0].Message.Content
			var structured map[string]any
			if err := json.Unmarshal([]byte(content), &structured); err != nil {
				t.Fatalf("content 不是合法 JSON: %v, content=%q", err, content)
			}
			t.Logf("model=%s structured=%v", modelName, structured)
		})
	}
}

func TestSenseNova_ImageModels(t *testing.T) {
	requireSenseNovaKey(t)
	if !senseNovaRunImageTests {
		t.Skip("图像模型测试默认关闭，将 senseNovaRunImageTests 改为 true 后运行")
	}
	for _, tc := range []struct {
		name  string
		model string
		size  string
	}{
		{name: "u1.5-lite", model: "sensenova-u1.5-lite", size: "512x512"},
		{name: "u1-fast", model: "sensenova-u1-fast", size: "2048x2048"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := senseNovaRequest(t, "/images/generations", map[string]any{
				"model":           tc.model,
				"prompt":          "一只在白色背景上的橙色猫咪，简洁插画",
				"size":            tc.size,
				"n":               1,
				"response_format": "b64_json",
				"watermark":       true,
			})
			if status == http.StatusTooManyRequests {
				t.Skip("触发限流")
			}
			if status != http.StatusOK {
				t.Fatalf("图像模型返回 HTTP %d: %s", status, string(body))
			}
			var result struct {
				Data []struct {
					URL     string `json:"url"`
					B64JSON string `json:"b64_json"`
				} `json:"data"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatalf("图像响应不是合法 JSON: %v", err)
			}
			if len(result.Data) != 1 || (result.Data[0].URL == "" && result.Data[0].B64JSON == "") {
				t.Fatalf("图像响应缺少 data/url/b64_json: %s", string(body))
			}
		})
	}
}
