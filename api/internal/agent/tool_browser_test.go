package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
)

// browserStub 记录各动作收到的入参，用于验证 BrowserTool 的转发与参数解析，
// 而不只是验证"方法被调用过"。
type browserStub struct {
	url        string
	fullPage   bool
	target     sandbox.BrowserTarget
	input      sandbox.BrowserInput
	key        string
	direction  string
	toEnd      bool
	javascript string
	maxLines   *int
	screenshot []byte
}

func (b *browserStub) Navigate(ctx context.Context, url string) (*model.ToolResult, error) {
	b.url = url
	return model.NewToolResult("ok"), nil
}

func (b *browserStub) Snapshot(ctx context.Context) (*model.ToolResult, error) {
	return model.NewToolResult("ok"), nil
}

func (b *browserStub) Screenshot(ctx context.Context, fullPage bool) ([]byte, error) {
	b.fullPage = fullPage
	return b.screenshot, nil
}

func (b *browserStub) Click(ctx context.Context, target sandbox.BrowserTarget) (*model.ToolResult, error) {
	b.target = target
	return model.NewToolResult("ok"), nil
}

func (b *browserStub) Input(ctx context.Context, req sandbox.BrowserInput) (*model.ToolResult, error) {
	b.input = req
	return model.NewToolResult("ok"), nil
}

func (b *browserStub) PressKey(ctx context.Context, key string) (*model.ToolResult, error) {
	b.key = key
	return model.NewToolResult("ok"), nil
}

func (b *browserStub) Scroll(ctx context.Context, direction string, toEnd bool) (*model.ToolResult, error) {
	b.direction, b.toEnd = direction, toEnd
	return model.NewToolResult("ok"), nil
}

func (b *browserStub) ConsoleExec(ctx context.Context, javascript string) (*model.ToolResult, error) {
	b.javascript = javascript
	return model.NewToolResult("ok"), nil
}

func (b *browserStub) ConsoleView(ctx context.Context, maxLines *int) (*model.ToolResult, error) {
	b.maxLines = maxLines
	return model.NewToolResult("ok"), nil
}

// TestBrowserToolScreenshotKeepsImageOutOfLLMData 回归：截图必须只进 Display。
// 截图字节一旦出现在 Data 里就会随 tool 消息回灌模型，把上下文撑爆。
func TestBrowserToolScreenshotKeepsImageOutOfLLMData(t *testing.T) {
	stub := &browserStub{screenshot: []byte("png-bytes")}
	result, err := NewBrowserTool(stub).Invoke(context.Background(), map[string]interface{}{
		"action": "screenshot", "full_page": true,
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if !stub.fullPage {
		t.Error("Invoke() should forward full_page=true")
	}

	display, ok := result.Display[browserDisplayScreenshot].(string)
	if !ok || !strings.HasPrefix(display, "data:image/png;base64,") {
		t.Fatalf("display screenshot = %v, want png data URI", result.Display)
	}
	if strings.Contains(result.LLMJSON(), "base64") {
		t.Fatalf("LLMJSON leaked screenshot data: %s", result.LLMJSON())
	}
}

// TestBrowserToolRoutesActionsToBrowser 表格驱动验证各 action 的参数解析与转发。
func TestBrowserToolRoutesActionsToBrowser(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
		verify func(t *testing.T, b *browserStub)
	}{
		{
			name:   "navigate",
			params: map[string]interface{}{"action": "navigate", "url": "https://example.com"},
			verify: func(t *testing.T, b *browserStub) {
				if b.url != "https://example.com" {
					t.Errorf("Navigate url = %q", b.url)
				}
			},
		},
		{
			name:   "click by index",
			params: map[string]interface{}{"action": "click", "index": 3.0},
			verify: func(t *testing.T, b *browserStub) {
				if b.target.Index == nil || *b.target.Index != 3 {
					t.Errorf("Click index = %v, want 3", b.target.Index)
				}
			},
		},
		{
			name:   "click by coordinates",
			params: map[string]interface{}{"action": "click", "coordinate_x": 12.5, "coordinate_y": 30.0},
			verify: func(t *testing.T, b *browserStub) {
				if b.target.X == nil || *b.target.X != 12.5 || b.target.Y == nil || *b.target.Y != 30 {
					t.Errorf("Click coordinates = (%v, %v)", b.target.X, b.target.Y)
				}
			},
		},
		{
			name: "input",
			params: map[string]interface{}{
				"action": "input", "index": 1.0, "text": "golang", "press_enter": false,
			},
			verify: func(t *testing.T, b *browserStub) {
				if b.input.Text != "golang" || b.input.Index == nil || *b.input.Index != 1 {
					t.Errorf("Input = %+v", b.input)
				}
				if b.input.PressEnter {
					t.Error("Input press_enter should stay false")
				}
			},
		},
		{
			name:   "scroll defaults to down",
			params: map[string]interface{}{"action": "scroll"},
			verify: func(t *testing.T, b *browserStub) {
				if b.direction != sandbox.ScrollDirectionDown || b.toEnd {
					t.Errorf("Scroll = (%q, %v), want (down, false)", b.direction, b.toEnd)
				}
			},
		},
		{
			name:   "scroll to end up",
			params: map[string]interface{}{"action": "scroll", "direction": "up", "to_end": true},
			verify: func(t *testing.T, b *browserStub) {
				if b.direction != sandbox.ScrollDirectionUp || !b.toEnd {
					t.Errorf("Scroll = (%q, %v), want (up, true)", b.direction, b.toEnd)
				}
			},
		},
		{
			name:   "press key",
			params: map[string]interface{}{"action": "press_key", "key": "Escape"},
			verify: func(t *testing.T, b *browserStub) {
				if b.key != "Escape" {
					t.Errorf("PressKey = %q", b.key)
				}
			},
		},
		{
			name:   "console exec",
			params: map[string]interface{}{"action": "console_exec", "javascript": "document.title"},
			verify: func(t *testing.T, b *browserStub) {
				if b.javascript != "document.title" {
					t.Errorf("ConsoleExec = %q", b.javascript)
				}
			},
		},
		{
			name:   "console view default lines",
			params: map[string]interface{}{"action": "console_view"},
			verify: func(t *testing.T, b *browserStub) {
				if b.maxLines != nil {
					t.Errorf("ConsoleView maxLines = %v, want nil (沙箱侧取默认值)", *b.maxLines)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &browserStub{}
			result, err := NewBrowserTool(stub).Invoke(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Invoke() error = %v", err)
			}
			if result == nil || !result.Success {
				t.Fatalf("Invoke() = %+v, want success", result)
			}
			tt.verify(t, stub)
		})
	}
}

// TestBrowserToolRejectsInvalidParams 参数缺失/非法时必须返回可展示的工具错误，
// 而不是把空参数透传给沙箱再拿回一个 500。
func TestBrowserToolRejectsInvalidParams(t *testing.T) {
	tests := []struct {
		name   string
		params map[string]interface{}
	}{
		{name: "unknown action", params: map[string]interface{}{"action": "hover"}},
		{name: "navigate without url", params: map[string]interface{}{"action": "navigate"}},
		{name: "click without target", params: map[string]interface{}{"action": "click"}},
		{name: "click with half coordinates", params: map[string]interface{}{"action": "click", "coordinate_x": 1.0}},
		{name: "input without text", params: map[string]interface{}{"action": "input", "index": 1.0}},
		{name: "press key without key", params: map[string]interface{}{"action": "press_key"}},
		{name: "scroll with bad direction", params: map[string]interface{}{"action": "scroll", "direction": "left"}},
		{name: "console exec without javascript", params: map[string]interface{}{"action": "console_exec"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewBrowserTool(&browserStub{}).Invoke(context.Background(), tt.params)
			if err != nil {
				t.Fatalf("Invoke() error = %v, want nil (参数错误属于业务结果)", err)
			}
			if result == nil || result.Success || result.Message == "" {
				t.Fatalf("Invoke() = %+v, want failure with message", result)
			}
		})
	}
}
