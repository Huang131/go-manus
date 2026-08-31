package external

import (
	"github.com/mooc-manus/go-manus/api/internal/model"
)

// Browser 浏览器接口
type Browser interface {
	// ViewPage 获取当前浏览器的页面内容
	ViewPage(sessionID string) (*model.ToolResult, error)

	// Navigate 使用浏览器导航到指定 URL
	Navigate(sessionID, url string) (*model.ToolResult, error)

	// Restart 重启浏览器并访问指定 URL
	Restart(sessionID, url string) (*model.ToolResult, error)

	// Click 通过索引或坐标点击元素
	Click(sessionID string, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error)

	// Input 在输入框中输入文本
	Input(sessionID, text string, pressEnter bool, index *int, coordinateX, coordinateY *float64) (*model.ToolResult, error)

	// MoveMouse 移动鼠标到指定坐标
	MoveMouse(sessionID string, coordinateX, coordinateY float64) (*model.ToolResult, error)

	// PressKey 模拟按键
	PressKey(sessionID, key string) (*model.ToolResult, error)

	// SelectOption 在下拉菜单中选择选项
	SelectOption(sessionID string, index, option int) (*model.ToolResult, error)

	// ScrollUp 向上滚动浏览器
	ScrollUp(sessionID string, toTop *bool) (*model.ToolResult, error)

	// ScrollDown 向下滚动浏览器
	ScrollDown(sessionID string, toDown *bool) (*model.ToolResult, error)

	// Screenshot 对当前页面截图
	Screenshot(sessionID string, fullPage *bool) ([]byte, error)

	// ConsoleExec 在浏览器控制台执行 JavaScript
	ConsoleExec(sessionID, javascript string) (*model.ToolResult, error)

	// ConsoleView 获取控制台输出
	ConsoleView(sessionID string, maxLines *int) (*model.ToolResult, error)
}
