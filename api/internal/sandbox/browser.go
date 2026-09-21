package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/model"
)

// Browser 浏览器自动化接口（通过 Playwright CDP 控制沙箱 Chrome）
type Browser interface {
	// ViewPage 获取当前页面内容
	ViewPage(ctx context.Context, sessionID string) (*model.ToolResult, error)
	// Navigate 导航到指定 URL
	Navigate(ctx context.Context, sessionID, url string) (*model.ToolResult, error)
	// Restart 重启浏览器并访问 URL（复用现有 Chrome，由 Supervisor 管理）
	Restart(ctx context.Context, sessionID, url string) (*model.ToolResult, error)
	// Click 点击元素（通过索引或坐标）
	Click(ctx context.Context, sessionID string, index *int, x, y *float64) (*model.ToolResult, error)
	// Input 在输入框输入文本
	Input(ctx context.Context, sessionID, text string, pressEnter bool, index *int, x, y *float64) (*model.ToolResult, error)
	// MoveMouse 移动鼠标到指定坐标
	MoveMouse(ctx context.Context, sessionID string, x, y float64) (*model.ToolResult, error)
	// PressKey 模拟按键
	PressKey(ctx context.Context, sessionID, key string) (*model.ToolResult, error)
	// SelectOption 在下拉菜单中选择选项
	SelectOption(ctx context.Context, sessionID string, index, option int) (*model.ToolResult, error)
	// ScrollUp 向上滚动页面
	ScrollUp(ctx context.Context, sessionID string, toTop *bool) (*model.ToolResult, error)
	// ScrollDown 向下滚动页面
	ScrollDown(ctx context.Context, sessionID string, toBottom *bool) (*model.ToolResult, error)
	// Screenshot 对当前页面截图（返回 PNG 二进制）
	Screenshot(ctx context.Context, sessionID string, fullPage *bool) ([]byte, error)
	// ConsoleExec 在浏览器控制台执行 JavaScript
	ConsoleExec(ctx context.Context, sessionID, javascript string) (*model.ToolResult, error)
	// ConsoleView 获取控制台输出日志
	ConsoleView(ctx context.Context, sessionID string, maxLines *int) (*model.ToolResult, error)
}

// BrowserClient 浏览器客户端（复用 SandboxClient 执行 Playwright 脚本）
type BrowserClient struct {
	sandbox Sandbox
}

func NewBrowserClient(sandbox Sandbox) *BrowserClient {
	return &BrowserClient{sandbox: sandbox}
}

// browserExec 在沙箱内执行 Playwright 脚本，必要时等待异步进程完成
func (c *BrowserClient) browserExec(ctx context.Context, sessionID, script string) (*model.ToolResult, error) {
	result, err := c.sandbox.ExecCommand(ctx, sessionID, "", script)
	if err != nil || result == nil || !result.Success {
		return result, err
	}
	// 命令仍在后台运行时，等待进程结束后读取输出
	if data, ok := result.Data.(map[string]interface{}); !ok || data["status"] != "running" {
		return result, nil
	}
	seconds := 10
	if _, err = c.sandbox.WaitProcess(ctx, sessionID, &seconds); err != nil {
		return toolResultErr(err)
	}
	output, err := c.sandbox.ReadShellOutput(ctx, sessionID, false)
	if err != nil {
		return toolResultErr(err)
	}
	if output == nil || !output.Success {
		return output, fmt.Errorf("browser command output unavailable: %w", err)
	}
	return &model.ToolResult{Success: true, Message: output.Message, Data: map[string]interface{}{"output": output.Message}}, nil
}

// browserScript 生成 Playwright 脚本并通过 heredoc 执行
func browserScript(operation string) string {
	return fmt.Sprintf(playwrightScript, cdpAddress, operation)
}

// jsString 将 Go 字符串安全序列化为 JS 字符串字面量（JSON 编码防注入）
func jsString(s string) string {
	str, err := sonic.MarshalString(s)
	if err != nil {
		return `""`
	}
	return str
}

func (c *BrowserClient) ViewPage(ctx context.Context, sessionID string) (*model.ToolResult, error) {
	return c.browserExec(ctx, sessionID, browserScript(`console.log(await page.content());`))
}

func (c *BrowserClient) Navigate(ctx context.Context, sessionID, targetURL string) (*model.ToolResult, error) {
	script := fmt.Sprintf(`await page.goto(%s);`, jsString(targetURL))
	return c.browserExec(ctx, sessionID, browserScript(script))
}

func (c *BrowserClient) Restart(ctx context.Context, sessionID, targetURL string) (*model.ToolResult, error) {
	// Chrome 由 Supervisor 管理，重启只重置复用页面，避免 pkill 杀掉守护进程
	return c.Navigate(ctx, sessionID, targetURL)
}

func (c *BrowserClient) Click(ctx context.Context, sessionID string, index *int, x, y *float64) (*model.ToolResult, error) {
	var script string
	if index != nil {
		script = fmt.Sprintf(`await page.locator('a').nth(%d).click();`, *index)
	} else if x != nil && y != nil {
		script = fmt.Sprintf(`await page.mouse.click(%f, %f);`, *x, *y)
	} else {
		return model.NewToolError("either index or coordinates required"), fmt.Errorf("either index or coordinates required")
	}
	return c.browserExec(ctx, sessionID, browserScript(script))
}

func (c *BrowserClient) Input(ctx context.Context, sessionID, text string, pressEnter bool, index *int, x, y *float64) (*model.ToolResult, error) {
	var script string
	if index != nil {
		if pressEnter {
			script = fmt.Sprintf(`await page.locator('input').nth(%d).fill(%s); await page.keyboard.press('Enter');`, *index, jsString(text))
		} else {
			script = fmt.Sprintf(`await page.locator('input').nth(%d).fill(%s);`, *index, jsString(text))
		}
	} else if x != nil && y != nil {
		if pressEnter {
			script = fmt.Sprintf(`await page.mouse.click(%f, %f); await page.keyboard.type(%s); await page.keyboard.press('Enter');`, *x, *y, jsString(text))
		} else {
			script = fmt.Sprintf(`await page.mouse.click(%f, %f); await page.keyboard.type(%s);`, *x, *y, jsString(text))
		}
	} else {
		return model.NewToolError("either index or coordinates required"), fmt.Errorf("either index or coordinates required")
	}
	return c.browserExec(ctx, sessionID, browserScript(script))
}

func (c *BrowserClient) MoveMouse(ctx context.Context, sessionID string, x, y float64) (*model.ToolResult, error) {
	script := fmt.Sprintf(`await page.mouse.move(%f, %f);`, x, y)
	return c.browserExec(ctx, sessionID, browserScript(script))
}

func (c *BrowserClient) PressKey(ctx context.Context, sessionID, key string) (*model.ToolResult, error) {
	script := fmt.Sprintf(`await page.keyboard.press(%s);`, jsString(key))
	return c.browserExec(ctx, sessionID, browserScript(script))
}

func (c *BrowserClient) SelectOption(ctx context.Context, sessionID string, selectIndex, optionIndex int) (*model.ToolResult, error) {
	script := fmt.Sprintf(`await page.locator('select').nth(%d).selectOption({ index: %d });`, selectIndex, optionIndex)
	return c.browserExec(ctx, sessionID, browserScript(script))
}

func (c *BrowserClient) ScrollUp(ctx context.Context, sessionID string, toTop *bool) (*model.ToolResult, error) {
	var script string
	if toTop != nil && *toTop {
		script = `await page.evaluate(() => window.scrollTo(0, 0));`
	} else {
		script = `await page.evaluate(() => window.scrollBy(0, -window.innerHeight));`
	}
	return c.browserExec(ctx, sessionID, browserScript(script))
}

func (c *BrowserClient) ScrollDown(ctx context.Context, sessionID string, toBottom *bool) (*model.ToolResult, error) {
	var script string
	if toBottom != nil && *toBottom {
		script = `await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));`
	} else {
		script = `await page.evaluate(() => window.scrollBy(0, window.innerHeight));`
	}
	return c.browserExec(ctx, sessionID, browserScript(script))
}

// Screenshot 对当前页面截图
func (c *BrowserClient) Screenshot(ctx context.Context, sessionID string, fullPage *bool) ([]byte, error) {
	var script string
	if fullPage != nil && *fullPage {
		script = `const screenshot = await page.screenshot({ fullPage: true }); console.log(Buffer.from(screenshot).toString('base64'));`
	} else {
		script = `const screenshot = await page.screenshot(); console.log(Buffer.from(screenshot).toString('base64'));`
	}
	result, err := c.browserExec(ctx, sessionID, browserScript(script))
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("screenshot failed: %s", result.Message)
	}
	output := result.Message
	if data, ok := result.Data.(map[string]interface{}); ok {
		if val, ok := data["output"].(string); ok {
			output = val
		}
	}
	output = strings.TrimSpace(output)
	decoded, err := base64.StdEncoding.DecodeString(output)
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}
	return decoded, nil
}

// ConsoleExec 在浏览器控制台执行 JavaScript
func (c *BrowserClient) ConsoleExec(ctx context.Context, sessionID, javascript string) (*model.ToolResult, error) {
	script := fmt.Sprintf(`await page.evaluate(() => { %s });`, javascript)
	return c.browserExec(ctx, sessionID, browserScript(script))
}

// ConsoleView 获取控制台输出
func (c *BrowserClient) ConsoleView(ctx context.Context, sessionID string, maxLines *int) (*model.ToolResult, error) {
	cmd := "cat /tmp/browser/console.log"
	if maxLines != nil {
		cmd = fmt.Sprintf("head -n %d /tmp/browser/console.log", *maxLines)
	}
	return c.sandbox.ExecCommand(ctx, sessionID, "", cmd)
}
