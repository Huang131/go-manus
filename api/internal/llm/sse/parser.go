package sse

import (
	"bufio"
	"context"
	"io"
	"strings"
)

// Package sse 提供 Server-Sent Events (SSE) 的解析能力
// 专注于 LLM streaming 场景，只实现必要的字段解析
//
// 不支持的 SSE 规范字段：
//
//   - retry:  自动重连间隔，控制客户端重连策略
//     LLM streaming 是请求-响应模式，无法从中间 token 恢复
//     断开后需重新发起完整请求，重连无意义
//
//   - id:    事件 ID，配合 Last-Event-ID 实现断点续传
//     LLM 输出是连续流式文本，无法从中间位置恢复
//     如果断开，重连也只能获取后续增量，无法补全已丢失部分
//
//   - comment 行（: 开头）：可忽略，不影响业务逻辑

// Frame 是解析后的 SSE 事件帧
type Frame struct {
	Event string // 事件类型，无 event: 行时为空字符串
	Data  []byte // data: 行的内容（已去除前缀）
}

// StreamHandler 每解析一个帧调用一次
// 返回 true 继续解析，返回 false 停止
type StreamHandler func(ctx context.Context, frame Frame) bool

// ParserConfig 解析器配置
type ParserConfig struct {
	InitialBufSize int // Scanner 初始 buffer 大小
	MaxLineSize    int // Scanner 最大行大小
}

// 默认配置
var DefaultConfig = ParserConfig{
	InitialBufSize: 4096,
	MaxLineSize:    1024 * 1024,
}

// Parse 从 io.Reader 解析 SSE 流
// 直到 EOF 或 handler 返回 false
// 注意：handler 不接收 context，无法检测取消；如需取消支持，请使用 ParseWithContext
func Parse(r io.Reader, cfg ParserConfig, handler func(frame Frame) bool) error {
	return ParseWithContext(context.Background(), r, cfg, func(_ context.Context, frame Frame) bool {
		return handler(frame)
	})
}

// ParseWithContext 与 Parse 相同，但 handler 接收 context 参数
// 可通过 ctx.Done() 检测取消
func ParseWithContext(ctx context.Context, r io.Reader, cfg ParserConfig, handler StreamHandler) error {
	// 配置默认值处理（值拷贝，不修改 DefaultConfig）
	if cfg.InitialBufSize <= 0 {
		cfg.InitialBufSize = DefaultConfig.InitialBufSize
	}
	if cfg.MaxLineSize <= 0 {
		cfg.MaxLineSize = DefaultConfig.MaxLineSize
	}

	// 创建 Scanner（行解析器）
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, cfg.InitialBufSize), cfg.MaxLineSize)

	// 状态变量
	var pendingData []byte  // 累积的多行 data
	var currentEvent string // 当前的 event 类型

	// 主循环：逐行解析
	for scanner.Scan() {
		// context 取消检测（每行检查一次）
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()               // 自动按 \n 分行
		line = strings.TrimRight(line, "\r") // 处理 CRLF

		// 空行是帧分隔符
		if line == "" {
			if len(pendingData) > 0 {
				frame := Frame{
					Event: currentEvent,
					Data:  pendingData,
				}
				if !handler(ctx, frame) {
					return nil
				}
				pendingData = nil
				currentEvent = ""
			}
			continue
		}

		// 解析 event: 行
		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		// 跳过 comment 行
		if strings.HasPrefix(line, ":") {
			continue
		}

		// 解析 data: 行
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

			// 多行 data 合并：SSE 规范，多个 data: 行用 \n 连接
			if pendingData != nil {
				pendingData = append(pendingData, '\n')
			}
			pendingData = append(pendingData, data...)
			continue
		}

		// 遇到非 data 行，说明 pending data 帧结束
		if len(pendingData) > 0 {
			frame := Frame{
				Event: currentEvent,
				Data:  pendingData,
			}
			if !handler(ctx, frame) {
				return nil
			}
			pendingData = nil
			currentEvent = ""
		}
	}

	// 处理最后一个 pending data
	if len(pendingData) > 0 {
		frame := Frame{
			Event: currentEvent,
			Data:  pendingData,
		}
		handler(ctx, frame)
	}

	return scanner.Err()
}
