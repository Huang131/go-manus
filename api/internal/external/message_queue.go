package external

import (
	"context"
	"time"
)

// MessageQueue 消息队列接口
type MessageQueue interface {
	// Put 往消息队列中添加一条消息，返回消息ID
	Put(ctx context.Context, streamName string, message interface{}) (string, error)

	// Get 根据传递的开始ID和阻塞时间获取一条数据
	// startID: 起始消息ID，nil 表示从最新消息开始
	// blockMs: 阻塞毫秒数，nil 表示非阻塞
	// 返回: 消息ID, 消息内容
	Get(ctx context.Context, streamName string, startID string, blockMs *int) (string, interface{}, error)

	// GetBlocking 阻塞获取消息，支持 context 取消和超时
	// startID: 起始消息ID，空字符串表示从最新消息开始
	// timeout: 单次阻塞超时，建议 3-5 秒
	// 返回: 消息ID, 消息内容
	// 注意: context 取消时会立即返回 context.Canceled
	GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error)

	// GetRange 获取指定范围内的消息
	// startID: 起始消息ID，"" 表示从最早消息开始
	// endID: 结束消息ID，"" 表示到最新消息
	// limit: 最大返回消息数，0 表示不限制
	// 返回: 消息列表
	GetRange(ctx context.Context, streamName string, startID, endID string, limit int64) ([]*Message, error)

	// GetLatestID 获取最新消息的ID
	// 返回: 最新消息ID，如果没有消息则返回空字符串
	GetLatestID(ctx context.Context, streamName string) (string, error)

	// Pop 获取并移除消息队列中的第一条消息
	Pop(ctx context.Context, streamName string) (string, interface{}, error)

	// Clear 清空消息队列中的所有消息
	Clear(ctx context.Context, streamName string) error

	// IsEmpty 判断消息队列是否为空
	IsEmpty(ctx context.Context, streamName string) (bool, error)

	// Size 获取消息队列的长度
	Size(ctx context.Context, streamName string) (int64, error)

	// DeleteMessage 根据传递的消息ID删除队列中指定的消息
	DeleteMessage(ctx context.Context, streamName string, messageID string) error

	// Subscribe 订阅消息，返回一个 channel，消息会异步推送
	// bufferSize: channel 缓冲区大小
	Subscribe(ctx context.Context, streamName string, bufferSize int) (<-chan *Message, func())

	// Close 关闭消息队列
	Close() error
}

// Message 消息结构
type Message struct {
	ID     string
	Data   interface{}
	Stream string
}
