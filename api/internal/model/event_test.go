package model

import (
	"encoding/json"
	"testing"

	"github.com/bytedance/sonic"
)

// TestEvent_Data_JSONRoundtrip 验证 Event.Data 字段在 JSON 序列化/反序列化后
// 保持原始 JSON 结构（不出现 base64 编码）。
//
// 历史 Bug:
//
//	Event.Data 原为 []byte，被 sonic 序列化为 base64 字符串（JSON 没有 []byte 类型，
//	标准库与 sonic 都会做 base64 编码）。改为 json.RawMessage 后，sonic 直接嵌入
//	原始 JSON，不再二次编码。
func TestEvent_Data_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "user message object",
			input: `{"role":"user","message":"GitHub上最热门的存储库有哪些？"}`,
		},
		{
			name:  "title string",
			input: `{"title":"GitHub热门存储库"}`,
		},
		{
			name:  "step result with attachments",
			input: `{"step":{"id":"1","description":"查询GitHub","status":"completed","success":true,"attachments":null},"status":"finished"}`,
		},
		{
			name:  "empty object",
			input: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := Event{
				ID:   "test-id",
				Type: EventTypeMessage,
				Data: json.RawMessage(tt.input),
			}
			out, err := sonic.Marshal(event)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}

			var probe struct {
				Data json.RawMessage `json:"data"`
			}
			if err := sonic.Unmarshal(out, &probe); err != nil {
				t.Fatalf("unmarshal probe failed: %v", err)
			}

			got := string(probe.Data)
			if len(got) == 0 || got[0] != '{' {
				t.Errorf("Data should be embedded as JSON object, got: %s\nfull json: %s", got, out)
			}

			var back Event
			if err := sonic.Unmarshal(out, &back); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if string(back.Data) != tt.input {
				t.Errorf("roundtrip mismatch:\nwant: %s\ngot:  %s", tt.input, back.Data)
			}
		})
	}
}

// TestEvent_Data_NotBase64 防御性回归测试：明确验证 Data 不会被 base64 编码。
func TestEvent_Data_NotBase64(t *testing.T) {
	original := `{"role":"user","message":"hello"}`
	event := Event{
		Type: EventTypeMessage,
		Data: json.RawMessage(original),
	}

	out, err := sonic.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if contains(string(out), `"data":"eyJ`) {
		t.Errorf("Data was base64-encoded, JSON output:\n%s", out)
	}
}

// TestEvent_Data_PreservesStructure 验证 Data 在嵌套场景下的结构保真度。
func TestEvent_Data_PreservesStructure(t *testing.T) {
	data := json.RawMessage(`{
		"plan": {
			"id": "",
			"title": "GitHub热门存储库",
			"steps": [
				{"id": "1", "description": "查询GitHub", "status": "pending", "success": false, "attachments": null}
			]
		},
		"status": "running"
	}`)

	event := Event{
		Type: EventTypePlan,
		Data: data,
	}

	out, err := sonic.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back struct {
		Data struct {
			Plan struct {
				Title string `json:"title"`
				Steps []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				} `json:"steps"`
			} `json:"plan"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := sonic.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if back.Data.Plan.Title != "GitHub热门存储库" {
		t.Errorf("plan title lost: %s", back.Data.Plan.Title)
	}
	if len(back.Data.Plan.Steps) != 1 || back.Data.Plan.Steps[0].ID != "1" {
		t.Errorf("plan steps lost: %+v", back.Data.Plan.Steps)
	}
	if back.Data.Status != "running" {
		t.Errorf("status lost: %s", back.Data.Status)
	}
}

// TestEvent_Data_Nil 验证 nil Data 序列化为 null。
func TestEvent_Data_Nil(t *testing.T) {
	event := Event{Type: EventTypeMessage}
	out, err := sonic.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !contains(string(out), `"data":null`) {
		t.Errorf("nil Data should serialize as null, got: %s", out)
	}
}
