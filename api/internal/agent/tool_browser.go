package agent

import (
	"context"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"
)

// BrowserTool 浏览器工具
type BrowserTool struct {
	browser external.Browser
}

// NewBrowserTool 创建浏览器工具
func NewBrowserTool(browser external.Browser) *BrowserTool {
	return &BrowserTool{browser: browser}
}

// Name 返回工具名称
func (t *BrowserTool) Name() string {
	return ToolNameBrowser
}

// Description 返回工具描述
func (t *BrowserTool) Description() string {
	return "用于控制浏览器。可以访问网页、点击元素、输入文本、滚动页面、截图等。"
}

// Parameters 返回工具参数定义
func (t *BrowserTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "操作类型: navigate, view, screenshot, click, input, scroll_up, scroll_down, press_key",
				"enum":        []string{BrowserActionNavigate, BrowserActionView, BrowserActionScreenshot, BrowserActionClick, BrowserActionInput, BrowserActionScrollUp, BrowserActionScrollDown, BrowserActionPressKey},
			},
			"session_id": map[string]interface{}{
				"type":        "string",
				"description": "浏览器会话 ID",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "要访问的 URL (navigate 操作)",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "输入文本 (input 操作)",
			},
			"press_enter": map[string]interface{}{
				"type":        "boolean",
				"description": "是否按回车键 (input 操作)",
			},
			"index": map[string]interface{}{
				"type":        "integer",
				"description": "元素索引 (click/input 操作)",
			},
			"coordinate_x": map[string]interface{}{
				"type":        "number",
				"description": "X 坐标 (click 操作)",
			},
			"coordinate_y": map[string]interface{}{
				"type":        "number",
				"description": "Y 坐标 (click 操作)",
			},
			"key": map[string]interface{}{
				"type":        "string",
				"description": "按键标识 (press_key 操作)",
			},
			"to_top": map[string]interface{}{
				"type":        "boolean",
				"description": "是否滚动到顶部 (scroll_up 操作)",
			},
			"to_down": map[string]interface{}{
				"type":        "boolean",
				"description": "是否滚动到底部 (scroll_down 操作)",
			},
			"full_page": map[string]interface{}{
				"type":        "boolean",
				"description": "是否整页截图 (screenshot 操作)",
			},
		},
		"required": []string{"action", "session_id"},
	}
}

// ReadOnly 浏览器工具可能触发导航/点击/输入，默认视为有副作用。
func (t *BrowserTool) ReadOnly() bool {
	return false
}

// Invoke 调用工具
func (t *BrowserTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	action, _ := params["action"].(string)
	sessionID, _ := params["session_id"].(string)

	switch action {
	case BrowserActionNavigate:
		url := ""
		if v, ok := params["url"].(string); ok {
			url = v
		}
		return t.browser.Navigate(sessionID, url)

	case BrowserActionView:
		return t.browser.ViewPage(sessionID)

	case BrowserActionScreenshot:
		var fullPage *bool
		if v, ok := params["full_page"].(bool); ok {
			fullPage = &v
		}
		data, err := t.browser.Screenshot(sessionID, fullPage)
		if err != nil {
			return model.NewToolError(err.Error()), err
		}
		return model.NewToolResult(map[string]interface{}{
			"screenshot_data": data,
		}), nil

	case BrowserActionClick:
		var index *int
		if v, ok := params["index"].(float64); ok {
			n := int(v)
			index = &n
		}
		var coordX, coordY *float64
		if v, ok := params["coordinate_x"].(float64); ok {
			coordX = &v
		}
		if v, ok := params["coordinate_y"].(float64); ok {
			coordY = &v
		}
		return t.browser.Click(sessionID, index, coordX, coordY)

	case BrowserActionInput:
		text := ""
		if v, ok := params["text"].(string); ok {
			text = v
		}
		pressEnter := false
		if v, ok := params["press_enter"].(bool); ok {
			pressEnter = v
		}
		var index *int
		if v, ok := params["index"].(float64); ok {
			n := int(v)
			index = &n
		}
		var coordX, coordY *float64
		if v, ok := params["coordinate_x"].(float64); ok {
			coordX = &v
		}
		if v, ok := params["coordinate_y"].(float64); ok {
			coordY = &v
		}
		return t.browser.Input(sessionID, text, pressEnter, index, coordX, coordY)

	case BrowserActionScrollUp:
		var toTop *bool
		if v, ok := params["to_top"].(bool); ok {
			toTop = &v
		}
		return t.browser.ScrollUp(sessionID, toTop)

	case BrowserActionScrollDown:
		var toDown *bool
		if v, ok := params["to_down"].(bool); ok {
			toDown = &v
		}
		return t.browser.ScrollDown(sessionID, toDown)

	case BrowserActionPressKey:
		key := ""
		if v, ok := params["key"].(string); ok {
			key = v
		}
		return t.browser.PressKey(sessionID, key)

	default:
		return model.NewToolError("unknown action: " + action), nil
	}
}
