package agent

import "context"

// Task 任务接口
type Task interface {
	// InputStream 返回输入流
	InputStream() Stream
	// OutputStream 返回输出流
	OutputStream() Stream
}

// Stream 消息流接口
type Stream interface {
	// Put 放入消息
	Put(ctx context.Context, data string) (string, error)
	// Pop 取出消息
	Pop(ctx context.Context) (string, string, error)
	// IsEmpty 检查是否为空
	IsEmpty(ctx context.Context) (bool, error)
	// Len 返回队列长度
	Len(ctx context.Context) (int, error)
}
