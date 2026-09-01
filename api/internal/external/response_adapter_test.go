package external

import "testing"

func TestNormalizeLLMResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   *LLMResponse
		want    string
		wantRaw string
	}{
		{
			name: "prefer content",
			input: &LLMResponse{
				ID:         "1",
				Content:    "  hello  ",
				RawContent: "raw hello",
			},
			want:    "hello",
			wantRaw: "raw hello",
		},
		{
			name: "fallback reasoning",
			input: &LLMResponse{
				ID:               "2",
				Content:          "",
				ReasoningContent: "  reasoning text  ",
			},
			want:    "reasoning text",
			wantRaw: "reasoning text",
		},
		{
			name: "preserve tool use",
			input: &LLMResponse{
				ID:      "3",
				Content: "ok",
				ToolUse: []map[string]interface{}{
					{
						"id": "tool-1",
					},
				},
			},
			want:    "ok",
			wantRaw: "ok",
		},
		{
			name:  "nil",
			input: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLLMResponse(tt.input)
			if tt.input == nil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got.Content != tt.want {
				t.Fatalf("content=%q want %q", got.Content, tt.want)
			}
			if got.RawContent != tt.wantRaw {
				t.Fatalf("raw=%q want %q", got.RawContent, tt.wantRaw)
			}
			if tt.name == "preserve tool use" && len(got.ToolUse) != 1 {
				t.Fatalf("tool use lost: %+v", got.ToolUse)
			}
		})
	}
}
