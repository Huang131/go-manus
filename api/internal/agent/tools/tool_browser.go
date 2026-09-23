package tools

import (
	"context"
	"strings"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
)

// BrowserTool 浏览器工具。
//
// 浏览器页面状态保存在沙箱内的 Chrome 里，工具本身无会话概念：
// 模型先 snapshot 拿到带编号的可交互元素，再用编号 click/input，
// 编号语义由沙箱侧统一生成，避免模型凭 HTML 猜元素位置。
type BrowserTool struct {
	browser sandbox.Browser
}

// NewBrowserTool 创建浏览器工具
func NewBrowserTool(browser sandbox.Browser) *BrowserTool {
	return &BrowserTool{browser: browser}
}

// Name 返回工具名称
func (t *BrowserTool) Name() string {
	return ToolNameBrowser
}

// Description 返回工具描述
func (t *BrowserTool) Description() string {
	return "控制浏览器。先 snapshot 获取元素编号，再按编号 click/input；支持滚动、按键、执行 JS、读日志、截图。"
}

// Parameters 返回工具参数定义
func (t *BrowserTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型",
				"enum": []string{
					BrowserActionNavigate, BrowserActionSnapshot, BrowserActionScreenshot,
					BrowserActionClick, BrowserActionInput, BrowserActionPressKey,
					BrowserActionScroll, BrowserActionConsoleExec, BrowserActionConsoleView,
				},
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "要访问的 URL",
			},
			"index": map[string]interface{}{
				"type":        "integer",
				"description": "元素编号（取自最近 snapshot）",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS 选择器兜底定位",
			},
			"coordinate_x": map[string]interface{}{
				"type":        "number",
				"description": "X 坐标兜底定位",
			},
			"coordinate_y": map[string]interface{}{
				"type":        "number",
				"description": "Y 坐标兜底定位",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "要输入的文本",
			},
			"press_enter": map[string]interface{}{
				"type":        "boolean",
				"description": "输入后是否按回车",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"description": "按键标识，如 Enter/Tab",
			},
			"direction": map[string]interface{}{
				"type":        "string",
				"description": "滚动方向",
				"enum":        []string{sandbox.ScrollDirectionUp, sandbox.ScrollDirectionDown},
			},
			"to_end": map[string]interface{}{
				"type":        "boolean",
				"description": "是否直达顶部/底部",
			},
			"full_page": map[string]interface{}{
				"type":        "boolean",
				"description": "是否整页截图",
			},
			"javascript": map[string]interface{}{
				"type":        "string",
				"description": "要执行的 JS 表达式",
			},
			"max_lines": map[string]interface{}{
				"type":        "integer",
				"description": "返回日志行数",
			},
		},
		// session_id 不暴露给模型：沙箱浏览器无需会话标识，由沙箱侧统一管理。
		"required": []string{"action"},
	}
}

// ReadOnly 浏览器工具可能触发导航/点击/输入，默认视为有副作用。
func (t *BrowserTool) ReadOnly() bool {
	return false
}

// Invoke 调用工具
func (t *BrowserTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	action, toolErr := requiredToolString(params, "action")
	if toolErr != nil {
		return toolErr, nil
	}

	switch action {
	case BrowserActionNavigate:
		url, toolErr := requiredToolString(params, "url")
		if toolErr != nil {
			return toolErr, nil
		}
		return t.browser.Navigate(ctx, url)

	case BrowserActionSnapshot:
		return t.browser.Snapshot(ctx)

	case BrowserActionScreenshot:
		data, err := t.browser.Screenshot(ctx, optionalToolBool(params, "full_page"))
		if err != nil {
			return model.NewToolError(err.Error()), err
		}
		// 截图以二进制产物挂载：运行期落存储后 Display 里只会留下文件引用，
		// 于是 LLM 上下文、SSE 事件与事件库都不会出现 base64。
		return model.NewToolResult(map[string]interface{}{"bytes": len(data)}).
				WithArtifact(BrowserScreenshotArtifact, model.ToolArtifact{
					Filename: BrowserScreenshotFilename,
					MimeType: BrowserScreenshotMimeType,
					Data:     data,
				}), nil

	case BrowserActionClick:
		target, toolErr := browserTargetFromParams(params)
		if toolErr != nil {
			return toolErr, nil
		}
		return t.browser.Click(ctx, target)

	case BrowserActionInput:
		text, toolErr := requiredToolString(params, "text")
		if toolErr != nil {
			return toolErr, nil
		}
		target, toolErr := browserTargetFromParams(params)
		if toolErr != nil {
			return toolErr, nil
		}
		return t.browser.Input(ctx, sandbox.BrowserInput{
			BrowserTarget: target,
			Text:          text,
			PressEnter:    optionalToolBool(params, "press_enter"),
		})

	case BrowserActionPressKey:
		key, toolErr := requiredToolString(params, "key")
		if toolErr != nil {
			return toolErr, nil
		}
		return t.browser.PressKey(ctx, key)

	case BrowserActionScroll:
		direction := optionalToolString(params, "direction")
		if direction == "" {
			direction = sandbox.ScrollDirectionDown
		}
		if direction != sandbox.ScrollDirectionUp && direction != sandbox.ScrollDirectionDown {
			return model.NewToolError("direction 只能是 up 或 down"), nil
		}
		return t.browser.Scroll(ctx, direction, optionalToolBool(params, "to_end"))

	case BrowserActionConsoleExec:
		javascript, toolErr := requiredToolString(params, "javascript")
		if toolErr != nil {
			return toolErr, nil
		}
		return t.browser.ConsoleExec(ctx, javascript)

	case BrowserActionConsoleView:
		return t.browser.ConsoleView(ctx, optionalToolInt(params, "max_lines"))

	default:
		return model.NewToolError("unknown action: " + action), nil
	}
}

// browserTargetFromParams 解析元素定位参数，要求三种定位方式至少给出一种。
func browserTargetFromParams(params map[string]interface{}) (sandbox.BrowserTarget, *model.ToolResult) {
	target := sandbox.BrowserTarget{
		Index:    optionalToolInt(params, "index"),
		Selector: strings.TrimSpace(optionalToolString(params, "selector")),
		X:        optionalToolFloat(params, "coordinate_x"),
		Y:        optionalToolFloat(params, "coordinate_y"),
	}
	if target.Index == nil && target.Selector == "" && (target.X == nil || target.Y == nil) {
		return target, model.NewToolError("需要 index、selector 或完整坐标之一来定位元素")
	}
	return target, nil
}
