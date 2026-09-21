package sandbox

import (
	"context"
	"fmt"

	"github.com/Huang131/go-manus/api/internal/model"
)

// 滚动方向取值，与沙箱侧 BrowserScrollRequest 的约束保持一致
const (
	ScrollDirectionUp   = "up"
	ScrollDirectionDown = "down"
)

// BrowserTarget 元素定位目标：编号、CSS 选择器、坐标三选一。
//
// 编号来自 Snapshot 返回的元素列表，与沙箱侧定位使用同一套选择器规则，
// 因此编号语义不会漂移；选择器和坐标用于兜底。
type BrowserTarget struct {
	Index    *int
	Selector string
	X        *float64
	Y        *float64
}

// BrowserInput 文本输入参数：定位目标 + 文本 + 是否回车
type BrowserInput struct {
	BrowserTarget
	Text       string
	PressEnter bool
}

// Browser 浏览器自动化接口。
//
// 每个方法对应沙箱侧一个浏览器动作端点和一次独立子进程调用，
// 页面状态保存在沙箱 Chrome 内，接口本身不持有会话状态。
type Browser interface {
	// Navigate 导航到指定 URL
	Navigate(ctx context.Context, url string) (*model.ToolResult, error)
	// Snapshot 获取带编号的可交互元素列表，编号可直接用于 Click/Input
	Snapshot(ctx context.Context) (*model.ToolResult, error)
	// Screenshot 截图，返回 PNG 原始字节
	Screenshot(ctx context.Context, fullPage bool) ([]byte, error)
	// Click 点击元素
	Click(ctx context.Context, target BrowserTarget) (*model.ToolResult, error)
	// Input 向输入框写入文本
	Input(ctx context.Context, req BrowserInput) (*model.ToolResult, error)
	// PressKey 模拟按键
	PressKey(ctx context.Context, key string) (*model.ToolResult, error)
	// Scroll 滚动页面，direction 取 ScrollDirectionUp/Down，toEnd 表示直达端点
	Scroll(ctx context.Context, direction string, toEnd bool) (*model.ToolResult, error)
	// ConsoleExec 在页面上下文执行 JavaScript
	ConsoleExec(ctx context.Context, javascript string) (*model.ToolResult, error)
	// ConsoleView 读取历史控制台日志
	ConsoleView(ctx context.Context, maxLines *int) (*model.ToolResult, error)
}

// BrowserClient 浏览器客户端：复用 SandboxClient 的传输层调用浏览器端点
type BrowserClient struct {
	client *SandboxClient
}

func NewBrowserClient(client *SandboxClient) *BrowserClient {
	return &BrowserClient{client: client}
}

func (c *BrowserClient) Navigate(ctx context.Context, url string) (*model.ToolResult, error) {
	return c.client.postTool(ctx, "browser/navigate", &browserNavigateRequest{URL: url})
}

func (c *BrowserClient) Snapshot(ctx context.Context) (*model.ToolResult, error) {
	return c.client.postTool(ctx, "browser/snapshot", &struct{}{})
}

// Screenshot 先让沙箱把 PNG 落到文件，再下载二进制。
//
// 早期实现把截图 base64 塞进 shell stdout，会被输出上限截断；
// 走文件通道后截图大小不受任何中间环节限制。
func (c *BrowserClient) Screenshot(ctx context.Context, fullPage bool) ([]byte, error) {
	var data browserScreenshotData
	if _, err := c.client.postJSON(ctx, "browser/screenshot", &browserScreenshotRequest{FullPage: fullPage}, &data); err != nil {
		return nil, err
	}
	if data.Path == "" {
		return nil, fmt.Errorf("sandbox screenshot returned empty path")
	}
	return c.client.downloadBytes(ctx, data.Path)
}

func (c *BrowserClient) Click(ctx context.Context, target BrowserTarget) (*model.ToolResult, error) {
	return c.client.postTool(ctx, "browser/click", target.request())
}

func (c *BrowserClient) Input(ctx context.Context, req BrowserInput) (*model.ToolResult, error) {
	body := &browserInputRequest{
		Index:      req.Index,
		Selector:   req.Selector,
		X:          req.X,
		Y:          req.Y,
		Text:       req.Text,
		PressEnter: req.PressEnter,
	}
	return c.client.postTool(ctx, "browser/input", body)
}

func (c *BrowserClient) PressKey(ctx context.Context, key string) (*model.ToolResult, error) {
	return c.client.postTool(ctx, "browser/press-key", &browserPressKeyRequest{Key: key})
}

func (c *BrowserClient) Scroll(ctx context.Context, direction string, toEnd bool) (*model.ToolResult, error) {
	return c.client.postTool(ctx, "browser/scroll", &browserScrollRequest{
		Direction: direction,
		ToEnd:     toEnd,
	})
}

func (c *BrowserClient) ConsoleExec(ctx context.Context, javascript string) (*model.ToolResult, error) {
	return c.client.postTool(ctx, "browser/console-exec", &browserConsoleExecRequest{Javascript: javascript})
}

func (c *BrowserClient) ConsoleView(ctx context.Context, maxLines *int) (*model.ToolResult, error) {
	return c.client.postTool(ctx, "browser/console-view", &browserConsoleViewRequest{MaxLines: maxLines})
}

// request 把定位目标转换为请求体
func (t BrowserTarget) request() *browserTargetRequest {
	return &browserTargetRequest{
		Index:    t.Index,
		Selector: t.Selector,
		X:        t.X,
		Y:        t.Y,
	}
}
