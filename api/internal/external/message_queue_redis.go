package external

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// 消息队列相关常量
const (
	defaultBlockTimeout = 3 * time.Second
	maxBlockTimeout     = 5 * time.Second
	// 保留足够的增量事件，避免短时断线时 token 前缀被过早裁剪。
	streamMaxLen             = 10000
	streamRetention          = 24 * time.Hour
	completedStreamRetention = 30 * time.Minute
)

// CompletedStreamRetention 返回任务完成后 SSE 续读所需的保留窗口。
func CompletedStreamRetention() time.Duration {
	return completedStreamRetention
}

// RedisStreamMessageQueue 基于 Redis Stream 的消息队列
type RedisStreamMessageQueue struct {
	client *redis.Client
}

// NewRedisStreamMessageQueue 创建 Redis Stream 消息队列
func NewRedisStreamMessageQueue(client *redis.Client) *RedisStreamMessageQueue {
	return &RedisStreamMessageQueue{client: client}
}

// Put 往消息队列中添加一条消息
func (q *RedisStreamMessageQueue) Put(ctx context.Context, streamName string, message interface{}) (string, error) {
	// 序列化消息
	data, err := sonic.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	result, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		MaxLen: streamMaxLen,
		Approx: true,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if err != nil {
		logger.Error("添加消息到队列失败",
			logger.String("stream", streamName),
			logger.Err(err))
		return "", fmt.Errorf("failed to add message: %w", err)
	}
	// 每次写入刷新运行期 TTL，保证超长任务不会在执行过程中丢失事件流。
	if err := q.client.Expire(ctx, streamName, streamRetention).Err(); err != nil {
		logger.WarnContext(ctx, "设置消息流过期时间失败",
			logger.String("stream", streamName),
			logger.Err(err))
	}

	logger.Debug("添加消息到队列成功",
		logger.String("stream", streamName),
		logger.String("message_id", result))

	return result, nil
}

// SetRetention 设置消息流保留时间，由任务生命周期在完成或取消时调用。
func (q *RedisStreamMessageQueue) SetRetention(ctx context.Context, streamName string, retention time.Duration) error {
	if retention <= 0 {
		return fmt.Errorf("stream retention must be positive")
	}
	if err := q.client.Expire(ctx, streamName, retention).Err(); err != nil {
		logger.WarnContext(ctx, "设置消息流保留时间失败",
			logger.String("stream", streamName),
			logger.Err(err))
		return fmt.Errorf("set stream retention: %w", err)
	}
	return nil
}

// GetBlocking 阻塞获取消息，支持 context 取消和超时
// timeout: 单次阻塞超时，建议 3-5 秒，最大不超过 5 秒
func (q *RedisStreamMessageQueue) GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error) {
	// 默认超时使用常量
	blockTimeout := defaultBlockTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		blockTimeout = timeout[0]
	}
	// 防止超时设置过大（使用常量限制）
	if blockTimeout > maxBlockTimeout {
		blockTimeout = maxBlockTimeout
	}

	// startID 为空时从最新消息开始
	if startID == "" {
		startID = "$"
	}

	for {
		// 使用短超时 BLOCK，定期检查 context
		result, err := q.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{streamName, startID},
			Count:   1,
			Block:   blockTimeout,
		}).Result()

		// 关键：先检查 context 是否已取消
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		default:
		}

		// redis.Nil 表示超时（BLOCK 时间到但无新消息）
		if err == redis.Nil {
			// 超时但 context 仍有效，继续等待
			continue
		}

		if err != nil {
			// 其他错误（如连接断开、context 取消等）
			return "", nil, fmt.Errorf("XRead failed: %w", err)
		}

		// 处理消息
		if len(result) > 0 && len(result[0].Messages) > 0 {
			msg := result[0].Messages[0]
			dataStr, ok := msg.Values["data"].(string)
			if !ok {
				return "", nil, fmt.Errorf("invalid message format")
			}

			var data interface{}
			if err := sonic.Unmarshal([]byte(dataStr), &data); err != nil {
				data = dataStr
			}

			return msg.ID, data, nil
		}
	}
}

// GetBlockingBatch 阻塞读取一批消息，返回严格位于 startID 之后的事件。
// COUNT 限制单次响应大小，避免 token 增量积压时产生过大的 SSE 批次。
func (q *RedisStreamMessageQueue) GetBlockingBatch(ctx context.Context, streamName string, startID string, count int, timeout ...time.Duration) ([]StreamMessage, error) {
	blockTimeout := defaultBlockTimeout
	if len(timeout) > 0 && timeout[0] > 0 {
		blockTimeout = timeout[0]
	}
	if blockTimeout > maxBlockTimeout {
		blockTimeout = maxBlockTimeout
	}
	if startID == "" {
		startID = "$"
	}
	if count <= 0 {
		count = 1
	}

	for {
		result, err := q.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{streamName, startID},
			Count:   int64(count),
			Block:   blockTimeout,
		}).Result()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("XRead batch failed: %w", err)
		}

		messages := make([]StreamMessage, 0, count)
		for _, stream := range result {
			for _, msg := range stream.Messages {
				dataStr, ok := msg.Values["data"].(string)
				if !ok {
					return nil, fmt.Errorf("invalid message format")
				}
				var data interface{}
				if err := sonic.Unmarshal([]byte(dataStr), &data); err != nil {
					data = dataStr
				}
				messages = append(messages, StreamMessage{ID: msg.ID, Data: data})
			}
		}
		return messages, nil
	}
}

// Clear 清空消息队列
func (q *RedisStreamMessageQueue) Clear(ctx context.Context, streamName string) error {
	// 使用 DEL 命令删除整个流
	if err := q.client.Del(ctx, streamName).Err(); err != nil {
		logger.Error("清空队列失败",
			logger.String("stream", streamName),
			logger.Err(err))
		return fmt.Errorf("failed to clear stream: %w", err)
	}
	return nil
}

// IsEmpty 判断消息队列是否为空
func (q *RedisStreamMessageQueue) IsEmpty(ctx context.Context, streamName string) (bool, error) {
	size, err := q.Size(ctx, streamName)
	if err != nil {
		return false, err
	}
	return size == 0, nil
}

// Size 获取消息队列长度
func (q *RedisStreamMessageQueue) Size(ctx context.Context, streamName string) (int64, error) {
	size, err := q.client.XLen(ctx, streamName).Result()
	if err != nil {
		logger.Error("获取队列长度失败",
			logger.String("stream", streamName),
			logger.Err(err))
		return 0, fmt.Errorf("failed to get stream length: %w", err)
	}
	return size, nil
}
