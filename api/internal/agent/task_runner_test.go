package agent

import (
	"encoding/json"
	"testing"
)

func TestSafeMarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "正常序列化字符串",
			input:   map[string]interface{}{"key": "value"},
			wantErr: false,
		},
		{
			name:    "序列化嵌套结构",
			input:   map[string]interface{}{"key": map[string]interface{}{"nested": "value"}},
			wantErr: false,
		},
		{
			name:    "序列化 nil 值",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "序列化数组",
			input:   []int{1, 2, 3},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeMarshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("safeMarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// 验证结果可以反序列化
				var decoded interface{}
				if err := json.Unmarshal(result, &decoded); err != nil {
					t.Errorf("safeMarshal() result = %s, is not valid JSON", string(result))
				}
			}
		})
	}
}

func TestPopRetryConfig(t *testing.T) {
	// 测试重试配置常量的合理性
	if popRetryBaseDelay <= 0 {
		t.Errorf("popRetryBaseDelay should be positive, got %v", popRetryBaseDelay)
	}

	if popRetryMaxDelay <= 0 {
		t.Errorf("popRetryMaxDelay should be positive, got %v", popRetryMaxDelay)
	}

	if popRetryMaxDelay < popRetryBaseDelay {
		t.Errorf("popRetryMaxDelay (%v) should be >= popRetryBaseDelay (%v)", popRetryMaxDelay, popRetryBaseDelay)
	}

	if popRetryMaxCount <= 0 {
		t.Errorf("popRetryMaxCount should be positive, got %d", popRetryMaxCount)
	}
}

func TestMustMarshal_BackwardCompatibility(t *testing.T) {
	// 测试 mustMarshal 向后兼容性
	result := mustMarshal(map[string]interface{}{"key": "value"})
	if len(result) == 0 {
		t.Error("mustMarshal() should return non-empty result for valid input")
	}

	// mustMarshal 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("mustMarshal() panicked: %v", r)
		}
	}()

	// 测试无效输入（channel 无法序列化）
	chanInput := make(chan int)
	_ = mustMarshal(chanInput) // 应该返回 "null"
}
