package model

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"
)

// TestRequestPolicy_Extra_JSONRoundtrip 验证 RequestPolicy.Extra 字段
// 不会被 base64 编码。
//
// 历史 Bug:
//
//	Extra 原为 map[string][]byte，map value 是 []byte 同样会被 sonic base64 编码。
//	改为 map[string]json.RawMessage 后解决。
func TestRequestPolicy_Extra_JSONRoundtrip(t *testing.T) {
	policy := RequestPolicy{
		ReasoningMode: ReasoningOff,
		Extra: map[string]json.RawMessage{
			"reasoning_effort": json.RawMessage(`"none"`),
			"temperature":      json.RawMessage(`0.7`),
			"tools":            json.RawMessage(`["search","calc"]`),
		},
	}

	out, err := sonic.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if contains(string(out), `"extra":{"reasoning_effort":"Im5vbmU`) {
		t.Errorf("Extra value was base64-encoded, JSON output:\n%s", out)
	}

	var back RequestPolicy
	if err := sonic.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if string(back.Extra["reasoning_effort"]) != `"none"` {
		t.Errorf("reasoning_effort lost: %s", back.Extra["reasoning_effort"])
	}
	if string(back.Extra["temperature"]) != `0.7` {
		t.Errorf("temperature lost: %s", back.Extra["temperature"])
	}
	if string(back.Extra["tools"]) != `["search","calc"]` {
		t.Errorf("tools lost: %s", back.Extra["tools"])
	}
}

// TestRequestPolicy_Extra_Omitempty 验证空 Extra 不写入 JSON。
func TestRequestPolicy_Extra_Omitempty(t *testing.T) {
	policy := RequestPolicy{ReasoningMode: ReasoningOff}
	out, err := sonic.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if contains(string(out), `"extra"`) {
		t.Errorf("empty Extra should be omitted, got: %s", out)
	}
}

// TestRequestPolicy_Extra_NestedObject 验证嵌套对象 value 也能正确存储。
func TestRequestPolicy_Extra_NestedObject(t *testing.T) {
	policy := RequestPolicy{
		ReasoningMode: ReasoningAuto,
		Extra: map[string]json.RawMessage{
			"response_format": json.RawMessage(`{"type":"json_object","schema":{"type":"object"}}`),
		},
	}

	out, err := sonic.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back struct {
		Extra map[string]json.RawMessage `json:"extra"`
	}
	if err := sonic.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	rf := string(back.Extra["response_format"])
	if rf != `{"type":"json_object","schema":{"type":"object"}}` {
		t.Errorf("nested object lost: %s", rf)
	}
}

func TestLLMModelAPIKey_InputOnly(t *testing.T) {
	var m LLMModel
	if err := sonic.Unmarshal([]byte(`{"api_key":"secret"}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.APIKey != "secret" {
		t.Fatalf("APIKey = %q, want secret", m.APIKey)
	}
	out, err := sonic.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if contains(string(out), "secret") || contains(string(out), "api_key") {
		t.Fatalf("API key leaked in response: %s", out)
	}
}

func TestLLMModelCapabilitiesApplyDefaultsWithoutOverwritingExplicitFalse(t *testing.T) {
	var m LLMModel
	input := []byte(`{"capabilities":{"supports_text":false,"supports_vision":true,"max_context_tokens":100000}}`)
	if err := sonic.Unmarshal(input, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Capabilities.SupportsText {
		t.Fatal("explicit supports_text=false was overwritten")
	}
	if !m.Capabilities.SupportsToolCalls || !m.Capabilities.SupportsStreaming {
		t.Fatalf("missing boolean defaults were not applied: %+v", m.Capabilities)
	}
	if !m.Capabilities.SupportsVision {
		t.Fatal("explicit supports_vision=true was lost")
	}
	if m.Capabilities.MaxContextTokens != 100000 || m.Capabilities.MaxOutputTokens == 0 {
		t.Fatalf("token defaults were not merged correctly: %+v", m.Capabilities)
	}
}
