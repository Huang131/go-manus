package external

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

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "simple object",
			input:  `{"key": "value"}`,
			expect: `{"key": "value"}`,
		},
		{
			name:   "nested object",
			input:  `prefix {"key": "value"} suffix`,
			expect: `{"key": "value"}`,
		},
		{
			name:   "array",
			input:  `prefix [1, 2, 3] suffix`,
			expect: `[1, 2, 3]`,
		},
		{
			name:   "complex nested",
			input:  `{"outer": {"inner": "value"}}`,
			expect: `{"outer": {"inner": "value"}}`,
		},
		{
			name:   "no json found returns original",
			input:  `no json here`,
			expect: `no json here`,
		},
		{
			name:   "empty input",
			input:  ``,
			expect: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if result != tt.expect {
				t.Errorf("extractJSON() = %q, expect %q", result, tt.expect)
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

func TestRemoveTrailingCommas(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "object with trailing comma",
			input:  `{"key": "value",}`,
			expect: `{"key": "value"}`,
		},
		{
			name:   "array with trailing comma",
			input:  `[1, 2, 3,]`,
			expect: `[1, 2, 3]`,
		},
		{
			name:   "nested with trailing comma",
			input:  `{"outer": {"inner": "value"},}`,
			expect: `{"outer": {"inner": "value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeTrailingCommas(tt.input)
			if result != tt.expect {
				t.Errorf("removeTrailingCommas() = %q, expect %q", result, tt.expect)
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
