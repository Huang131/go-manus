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
