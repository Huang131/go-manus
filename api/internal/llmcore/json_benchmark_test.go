package llmcore

import (
	stdjson "encoding/json"
	"testing"

	"github.com/bytedance/sonic"
)

// TestStruct 用于测试的结构体
type TestStruct struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	BaseURL   string                 `json:"base_url"`
	APIKey    string                 `json:"api_key"`
	ModelName string                 `json:"model_name"`
	Enabled   bool                   `json:"enabled"`
	Tags      []string               `json:"tags"`
	Extra     map[string]interface{} `json:"extra"`
}

// LargeStruct 大结构体测试
type LargeStruct struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	BaseURL       string                 `json:"base_url"`
	APIKey        string                 `json:"api_key"`
	ModelName     string                 `json:"model_name"`
	Provider      string                 `json:"provider"`
	Capabilities  []string               `json:"capabilities"`
	Tags          []string               `json:"tags"`
	Enabled       bool                   `json:"enabled"`
	IsDefault     bool                   `json:"is_default"`
	Temperature   float64                `json:"temperature"`
	MaxTokens     int                    `json:"max_tokens"`
	Extra         map[string]interface{} `json:"extra"`
	Headers       map[string]string      `json:"headers"`
	Timeout       int                    `json:"timeout"`
	RetryCount    int                    `json:"retry_count"`
	RetryDelay    int                    `json:"retry_delay"`
	FallbackModel string                 `json:"fallback_model"`
	Streaming     bool                   `json:"streaming"`
	Vision        bool                   `json:"vision"`
	LongContext   bool                   `json:"long_context"`
	Tools         []string               `json:"tools"`
	CustomParams  map[string]string      `json:"custom_params"`
}

var testData = TestStruct{
	ID:        "test-id-123456",
	Name:      "Test Model",
	BaseURL:   "https://api.test.com/v1",
	APIKey:    "sk-test-key-1234567890",
	ModelName: "gpt-4o",
	Enabled:   true,
	Tags:      []string{"vision", "tools", "long_ctx", "streaming"},
	Extra: map[string]interface{}{
		"reasoning_effort": "none",
		"custom_param":     "value",
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	},
}

var largeTestData = LargeStruct{
	ID:            "large-test-id",
	Name:          "Large Test Model",
	BaseURL:       "https://api.test.com/v1",
	APIKey:        "sk-test-key-1234567890",
	ModelName:     "gpt-4o-32k",
	Provider:      "openai",
	Capabilities:  []string{"vision", "tools", "streaming"},
	Tags:          []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
	Enabled:       true,
	IsDefault:     true,
	Temperature:   0.7,
	MaxTokens:     4096,
	Extra:         map[string]interface{}{"key": "value"},
	Headers:       map[string]string{"X-Custom": "header"},
	Timeout:       60,
	RetryCount:    3,
	RetryDelay:    1000,
	FallbackModel: "gpt-3.5-turbo",
	Streaming:     true,
	Vision:        true,
	LongContext:   true,
	Tools:         []string{"tool1", "tool2", "tool3"},
	CustomParams:  map[string]string{"param1": "value1"},
}

// ==================== Marshal Benchmark ====================

func Benchmark_StdLib_Marshal_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(testData)
	}
}

func Benchmark_Sonic_Marshal_Small(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(testData)
	}
}

func Benchmark_StdLib_Marshal_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = stdjson.Marshal(largeTestData)
	}
}

func Benchmark_Sonic_Marshal_Large(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = sonic.Marshal(largeTestData)
	}
}

// ==================== Unmarshal Benchmark ====================

var smallJSON = []byte(`{"id":"test-id","name":"Test","base_url":"https://api.test.com","api_key":"key","model_name":"gpt-4","enabled":true,"tags":["v","t"],"extra":{"k":"v"}}`)
var largeJSON = []byte(`{"id":"large","name":"Large","base_url":"https://api.test.com","api_key":"key","model_name":"gpt-4","provider":"openai","capabilities":["v","t"],"tags":["t1"],"enabled":true,"is_default":true,"temperature":0.7,"max_tokens":4096,"extra":{"k":"v"},"headers":{"h":"v"},"timeout":60,"retry_count":3,"retry_delay":1000,"fallback_model":"gpt-3.5","streaming":true,"vision":true,"long_context":true,"tools":["t1"],"custom_params":{"p":"v"}}`)

func Benchmark_StdLib_Unmarshal_Small(b *testing.B) {
	var result TestStruct
	for i := 0; i < b.N; i++ {
		_ = stdjson.Unmarshal(smallJSON, &result)
	}
}

func Benchmark_Sonic_Unmarshal_Small(b *testing.B) {
	var result TestStruct
	for i := 0; i < b.N; i++ {
		_ = sonic.Unmarshal(smallJSON, &result)
	}
}

func Benchmark_StdLib_Unmarshal_Large(b *testing.B) {
	var result LargeStruct
	for i := 0; i < b.N; i++ {
		_ = stdjson.Unmarshal(largeJSON, &result)
	}
}

func Benchmark_Sonic_Unmarshal_Large(b *testing.B) {
	var result LargeStruct
	for i := 0; i < b.N; i++ {
		_ = sonic.Unmarshal(largeJSON, &result)
	}
}

// ==================== Round-trip Benchmark ====================

func Benchmark_StdLib_Roundtrip_Small(b *testing.B) {
	var result TestStruct
	for i := 0; i < b.N; i++ {
		data, _ := stdjson.Marshal(testData)
		_ = stdjson.Unmarshal(data, &result)
	}
}

func Benchmark_Sonic_Roundtrip_Small(b *testing.B) {
	var result TestStruct
	for i := 0; i < b.N; i++ {
		data, _ := sonic.Marshal(testData)
		_ = sonic.Unmarshal(data, &result)
	}
}

func Benchmark_StdLib_Roundtrip_Large(b *testing.B) {
	var result LargeStruct
	for i := 0; i < b.N; i++ {
		data, _ := stdjson.Marshal(largeTestData)
		_ = stdjson.Unmarshal(data, &result)
	}
}

func Benchmark_Sonic_Roundtrip_Large(b *testing.B) {
	var result LargeStruct
	for i := 0; i < b.N; i++ {
		data, _ := sonic.Marshal(largeTestData)
		_ = sonic.Unmarshal(data, &result)
	}
}
