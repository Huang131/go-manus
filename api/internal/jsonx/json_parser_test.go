package jsonx

import (
	"testing"
)

func TestRepairJSONParser_Parse(t *testing.T) {
	parser := NewRepairJSONParser()

	tests := []struct {
		name    string
		input   string
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:  "valid json",
			input: `{"success": true, "result": "test"}`,
			want: map[string]interface{}{
				"success": true,
				"result":  "test",
			},
			wantErr: false,
		},
		{
			name:  "json with trailing comma",
			input: `{"success": true, "result": "test",}`,
			want: map[string]interface{}{
				"success": true,
				"result":  "test",
			},
			wantErr: false,
		},
		{
			name:  "json in markdown code block",
			input: "```json\n{\"success\": true}\n```",
			want: map[string]interface{}{
				"success": true,
			},
			wantErr: false,
		},
		{
			name:  "json with single quotes",
			input: `{"success": true, "result": 'test'}`,
			want: map[string]interface{}{
				"success": true,
				"result":  "test",
			},
			wantErr: false,
		},
		{
			name:  "json with leading text",
			input: "Here is the result: {\"success\": true}",
			want: map[string]interface{}{
				"success": true,
			},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:  "invalid json but repairable",
			input: `{invalid json}`,
			// 当前实现会尽量修复 LLM 输出的异常 JSON，这类输入应按可修复样本处理。
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			err := parser.Parse(tt.input, &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Error("Parse() returned nil without error")
				return
			}
		})
	}
}

func TestRepairJSONParser_ParseWithRepairs(t *testing.T) {
	parser := NewRepairJSONParser()

	// 核心验证场景：
	// - 无修复成功（valid json / markdown 包裹的合法 json）→ WasRepaired=false，无 repair 记录
	// - 括号截断（字符串值闭合，仅缺结尾括号）→ complete_brackets
	// - tailing comma / 单引号 / 前后文字 → json-repair 兜底成功
	tests := []struct {
		name         string
		input        string
		wantErr      bool
		wantRepaired bool
		wantStrategy string
	}{
		{
			name:         "valid json needs no repair",
			input:        `{"success": true}`,
			wantErr:      false,
			wantRepaired: false,
			wantStrategy: "",
		},
		{
			name:         "markdown code block with valid json inside",
			input:        "```json\n{\"foo\": \"bar\"}\n```",
			wantErr:      false,
			wantRepaired: false, // 去掉 markdown 后直接可解析
			wantStrategy: "",
		},
		{
			name:         "truncated object repaired by bracket completion",
			input:        `{"foo":"bar"`, // 字符串值闭合，仅缺结尾 }
			wantErr:      false,
			wantRepaired: true,
			wantStrategy: "complete_brackets",
		},
		{
			name:         "nested truncated object repaired by bracket completion",
			input:        `{"outer":{"inner":"value"`, // 两层对象，缺两个 }
			wantErr:      false,
			wantRepaired: true,
			wantStrategy: "complete_brackets",
		},
		{
			name:         "trailing comma repaired by json-repair",
			input:        `{"foo":"bar",}`,
			wantErr:      false,
			wantRepaired: true,
			wantStrategy: "json_repair",
		},
		{
			name:         "single quotes repaired by json-repair",
			input:        `{'foo':'bar'}`,
			wantErr:      false,
			wantRepaired: true,
			wantStrategy: "json_repair",
		},
		{
			name:         "leading text repaired by json-repair",
			input:        `Result: {"foo":"bar"}`,
			wantErr:      false,
			wantRepaired: true,
			wantStrategy: "json_repair",
		},
		{
			name:         "markdown with single quotes repaired by json-repair",
			input:        "```json\n{'foo': 'bar'}\n```",
			wantErr:      false,
			wantRepaired: true,
			wantStrategy: "json_repair",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]interface{}
			repairs, err := parser.ParseWithRepairs(tt.input, &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWithRepairs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if repairs.WasRepaired != tt.wantRepaired {
				t.Errorf("WasRepaired = %v, want %v", repairs.WasRepaired, tt.wantRepaired)
			}

			if repairs.StrategyUsed != tt.wantStrategy {
				t.Errorf("StrategyUsed = %q, want %q", repairs.StrategyUsed, tt.wantStrategy)
			}

			// WasRepaired 与 Repairs 列表、StrategyUsed 三者必须自洽
			if tt.wantRepaired {
				if len(repairs.Repairs) == 0 {
					t.Errorf("WasRepaired=true 但 Repairs 为空")
				}
			} else {
				if len(repairs.Repairs) != 0 {
					t.Errorf("WasRepaired=false 但 Repairs 非空: %v", repairs.Repairs)
				}
			}
		})
	}
}

func TestRemoveMarkdownCodeBlocks(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "json code block",
			input:  "```json\n{\"key\": \"value\"}\n```",
			expect: "{\"key\": \"value\"}\n",
		},
		{
			name:   "plain code block",
			input:  "```\n{\"key\": \"value\"}\n```",
			expect: "{\"key\": \"value\"}\n",
		},
		{
			name:   "no code block",
			input:  `{"key": "value"}`,
			expect: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeMarkdownCodeBlocks(tt.input)
			if result != tt.expect {
				t.Errorf("removeMarkdownCodeBlocks() = %q, expect %q", result, tt.expect)
			}
		})
	}
}

func TestDefaultJSONParser(t *testing.T) {
	parser := NewDefaultJSONParser()

	input := `{"success": true, "result": "test"}`
	var got map[string]interface{}

	err := parser.Parse(input, &got)
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	if got["success"] != true {
		t.Errorf("Parse() got success = %v, want true", got["success"])
	}

	if got["result"] != "test" {
		t.Errorf("Parse() got result = %v, want test", got["result"])
	}
}

func TestIsTruncatedJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{name: "valid object", input: `{"key": "value"}`, expect: false},
		{name: "valid array", input: `[1, 2, 3]`, expect: false},
		{name: "truncated object", input: `{"key": "value"`, expect: true},
		{name: "truncated array", input: `[1, 2, 3`, expect: true},
		{name: "nested truncated", input: `{"outer": {"inner": "value"`, expect: true},
		{name: "truncated in string", input: `{"key": "value}"}`, expect: false}, // } 在字符串内
		{name: "empty string", input: ``, expect: false},
		{name: "no brackets", input: `"just a string"`, expect: false},
		{name: "starts with text", input: `text {"key": "value"}`, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTruncatedJSON(tt.input)
			if result != tt.expect {
				t.Errorf("isTruncatedJSON(%q) = %v, want %v", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCompleteTruncatedJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		// 完整对象/数组：无缺失，原样返回
		{name: "complete object", input: `{"a": 1}`, expect: `{"a": 1}`},
		{name: "complete array", input: `[1, 2, 3]`, expect: `[1, 2, 3]`},
		// 缺一个 }
		{name: "missing one brace", input: `{"a": 1`, expect: `{"a": 1}`},
		// 缺两个 }（嵌套）
		{name: "missing two braces", input: `{"x": {"a": 1`, expect: `{"x": {"a": 1}}`},
		// 缺一个 ]
		{name: "missing one bracket", input: `[1, 2, 3`, expect: `[1, 2, 3]`},
		// 对象内嵌数组，缺 ] 和 }
		{name: "mixed object and array", input: `[{"a": 1}`, expect: `[{"a": 1}]`},
		// 非截断场景（unquoted key），原样返回
		{name: "unquoted key not truncation", input: `{name: John}`, expect: `{name: John}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := completeTruncatedJSON(tt.input)
			if result != tt.expect {
				t.Errorf("completeTruncatedJSON(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}
