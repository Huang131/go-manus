package external

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"go.uber.org/zap"
)

// 消息队列相关常量
const (
	defaultBufferSize   = 100 // 默认 channel 缓冲区大小
	defaultBlockTimeout = 3   // 默认阻塞超时（秒）
	maxBlockTimeout     = 5   // 最大阻塞超时（秒）
	lockExpireSec       = 10  // 分布式锁默认过期时间（秒）
	lockRetryInterval   = 100 // 锁重试间隔（毫秒）
	pollInterval        = 100 // 轮询间隔（毫秒）
)

// RedisStreamMessageQueue 基于 Redis Stream 的消息队列
type RedisStreamMessageQueue struct {
	mu            sync.RWMutex
	client        *redis.Client
	lockExpireSec int
}

// NewRedisStreamMessageQueue 创建 Redis Stream 消息队列
func NewRedisStreamMessageQueue(client *redis.Client) *RedisStreamMessageQueue {
	return &RedisStreamMessageQueue{
		client:        client,
		lockExpireSec: 10,
	}
}

// Put 往消息队列中添加一条消息
func (q *RedisStreamMessageQueue) Put(ctx context.Context, streamName string, message interface{}) (string, error) {
	// 序列化消息
	data, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	result, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"data": string(data),
		},
	}).Result()

	if err != nil {
		logger.Error("添加消息到队列失败",
			zap.String("stream", streamName),
			zap.Error(err))
		return "", fmt.Errorf("failed to add message: %w", err)
	}

	logger.Debug("添加消息到队列成功",
		zap.String("stream", streamName),
		zap.String("message_id", result))

	return result, nil
}

// Get 获取一条消息
func (q *RedisStreamMessageQueue) Get(ctx context.Context, streamName string, startID string, blockMs *int) (string, interface{}, error) {
	// 如果 startID 为空，从最新消息开始
	if startID == "" {
		startID = "$"
	}

	// 构建读取参数
	args := &redis.XReadArgs{
		Streams: []string{streamName, startID},
		Count:   1,
	}

	// 如果指定了阻塞时间
	if blockMs != nil && *blockMs > 0 {
		args.Block = time.Duration(*blockMs) * time.Millisecond
	}

	// 读取消息
	result, err := q.client.XRead(ctx, args).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil, nil // 超时无消息
		}
		logger.Error("从队列获取消息失败",
			zap.String("stream", streamName),
			zap.Error(err))
		return "", nil, fmt.Errorf("failed to get message: %w", err)
	}

	// 检查结果
	if len(result) == 0 || len(result[0].Messages) == 0 {
		return "", nil, nil
	}

	// 获取第一条消息
	msg := result[0].Messages[0]
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		return "", nil, fmt.Errorf("invalid message format")
	}

	// 反序列化消息
	var data interface{}
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		// 如果反序列化失败，返回原始字符串
		data = dataStr
	}

	return msg.ID, data, nil
}

// GetBlocking 阻塞获取消息，支持 context 取消和超时
// timeout: 单次阻塞超时，建议 3-5 秒，最大不超过 5 秒
func (q *RedisStreamMessageQueue) GetBlocking(ctx context.Context, streamName string, startID string, timeout ...time.Duration) (string, interface{}, error) {
	// 默认超时使用常量
	blockTimeout := time.Duration(defaultBlockTimeout) * time.Second
	if len(timeout) > 0 && timeout[0] > 0 {
		blockTimeout = timeout[0]
	}
	// 防止超时设置过大（使用常量限制）
	if blockTimeout > time.Duration(maxBlockTimeout)*time.Second {
		blockTimeout = time.Duration(maxBlockTimeout) * time.Second
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
			if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
				data = dataStr
			}

			return msg.ID, data, nil
		}
	}
}

// Pop 获取并移除消息队列中的第一条消息
func (q *RedisStreamMessageQueue) Pop(ctx context.Context, streamName string) (string, interface{}, error) {
	lockKey := fmt.Sprintf("lock:%s:pop", streamName)

	// 获取分布式锁
	lockValue, err := q.acquireLock(ctx, lockKey, 5*time.Second)
	if err != nil {
		return "", nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if lockValue == "" {
		return "", nil, nil // 获取锁失败
	}

	// 确保释放锁
	defer q.releaseLock(ctx, lockKey, lockValue)

	// 从流中获取第一条消息
	messages, err := q.client.XRange(ctx, streamName, "-", "+").Result()
	if err != nil {
		logger.Error("获取队列消息失败",
			zap.String("stream", streamName),
			zap.Error(err))
		return "", nil, fmt.Errorf("failed to range messages: %w", err)
	}

	if len(messages) == 0 {
		return "", nil, nil
	}

	// 获取第一条消息
	msg := messages[0]

	// 删除消息
	if err := q.client.XDel(ctx, streamName, msg.ID).Err(); err != nil {
		logger.Error("删除队列消息失败",
			zap.String("stream", streamName),
			zap.String("message_id", msg.ID),
			zap.Error(err))
		return "", nil, fmt.Errorf("failed to delete message: %w", err)
	}

	// 解析消息内容
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		return msg.ID, nil, nil
	}

	var data interface{}
	if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
		data = dataStr
	}

	return msg.ID, data, nil
}

// Clear 清空消息队列
func (q *RedisStreamMessageQueue) Clear(ctx context.Context, streamName string) error {
	// 使用 DEL 命令删除整个流
	if err := q.client.Del(ctx, streamName).Err(); err != nil {
		logger.Error("清空队列失败",
			zap.String("stream", streamName),
			zap.Error(err))
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
			zap.String("stream", streamName),
			zap.Error(err))
		return 0, fmt.Errorf("failed to get stream length: %w", err)
	}
	return size, nil
}

// DeleteMessage 删除指定消息
func (q *RedisStreamMessageQueue) DeleteMessage(ctx context.Context, streamName string, messageID string) error {
	if err := q.client.XDel(ctx, streamName, messageID).Err(); err != nil {
		logger.Error("删除消息失败",
			zap.String("stream", streamName),
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}

// Close 关闭消息队列
func (q *RedisStreamMessageQueue) Close() error {
	// Redis 客户端由外部管理，这里不需要关闭
	return nil
}

// Subscribe 订阅消息，返回一个 channel 用于接收消息
// 使用 context 控制生命周期，避免 goroutine 泄漏
func (q *RedisStreamMessageQueue) Subscribe(ctx context.Context, streamName string, bufferSize int) (<-chan *Message, func()) {
	if bufferSize <= 0 {
		bufferSize = defaultBufferSize
	}

	msgChan := make(chan *Message, bufferSize)

	// 使用 context 创建取消函数
	// 创建派生 context，这样调用者可以独立控制订阅的生命周期
	ctx, cancel := context.WithCancel(ctx)

	// 取消订阅函数：同时关闭 context
	cleanup := func() {
		cancel()
		// channel 会在 goroutine 退出时自动关闭
	}

	// 启动后台 goroutine 持续读取消息
	go func() {
		defer close(msgChan)
		defer cancel() // 确保 goroutine 结束时取消 context

		lastID := "$" // 从最新消息开始
		ticker := time.NewTicker(time.Duration(pollInterval) * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// context 被取消，正常退出
				logger.Debug("订阅已取消", zap.String("stream", streamName))
				return
			case <-ticker.C:
				// 获取新消息（非阻塞）
				id, data, err := q.Get(ctx, streamName, lastID, nil)
				if err != nil {
					// 检查是否是 context 取消导致的错误
					select {
					case <-ctx.Done():
						return
					default:
						logger.Warn("订阅获取消息失败",
							zap.String("stream", streamName),
							zap.Error(err))
						continue
					}
				}

				if id == "" || data == nil {
					continue
				}

				// 发送消息到 channel
				msg := &Message{
					ID:     id,
					Data:   data,
					Stream: streamName,
				}

				select {
				case msgChan <- msg:
					lastID = id // 更新起始位置
				default:
					// channel 满了，跳过这条消息
					logger.Warn("消息 channel 已满，跳过消息",
						zap.String("stream", streamName),
						zap.String("message_id", id))
				}
			}
		}
	}()

	return msgChan, cleanup
}

// acquireLock 获取分布式锁
func (q *RedisStreamMessageQueue) acquireLock(ctx context.Context, lockKey string, timeout time.Duration) (string, error) {
	lockValue := uuid.New().String()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// 尝试设置锁
		ok, err := q.client.SetNX(ctx, lockKey, lockValue, time.Duration(q.lockExpireSec)*time.Second).Result()
		if err != nil {
			return "", err
		}

		if ok {
			return lockValue, nil
		}

		// 等待后重试
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return "", nil // 超时
}

// GetRange 获取指定范围内的消息
func (q *RedisStreamMessageQueue) GetRange(ctx context.Context, streamName string, startID, endID string, limit int64) ([]*Message, error) {
	// 处理起始ID
	if startID == "" {
		startID = "-"
	}
	// 处理结束ID
	if endID == "" {
		endID = "+"
	}

	// 使用 XRange 获取范围内的消息
	messages, err := q.client.XRange(ctx, streamName, startID, endID).Result()
	if err != nil {
		logger.Error("获取范围消息失败",
			zap.String("stream", streamName),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get range messages: %w", err)
	}

	// 应用限制
	if limit > 0 && int64(len(messages)) > limit {
		messages = messages[:limit]
	}

	// 转换为 Message 结构
	result := make([]*Message, 0, len(messages))
	for _, msg := range messages {
		dataStr, ok := msg.Values["data"].(string)
		if !ok {
			continue
		}

		var data interface{}
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			data = dataStr
		}

		result = append(result, &Message{
			ID:     msg.ID,
			Data:   data,
			Stream: streamName,
		})
	}

	logger.Debug("获取范围消息成功",
		zap.String("stream", streamName),
		zap.String("start_id", startID),
		zap.String("end_id", endID),
		zap.Int("count", len(result)))

	return result, nil
}

// GetLatestID 获取最新消息的ID
func (q *RedisStreamMessageQueue) GetLatestID(ctx context.Context, streamName string) (string, error) {
	// 使用 XRevRange 获取最新消息（倒序取第一个）
	// + 表示最大 ID，- 表示最小 ID，所以是获取所有消息然后倒序
	messages, err := q.client.XRevRange(ctx, streamName, "+", "-").Result()
	if err != nil {
		logger.Error("获取最新消息ID失败",
			zap.String("stream", streamName),
			zap.Error(err))
		return "", fmt.Errorf("failed to get latest message ID: %w", err)
	}

	if len(messages) == 0 {
		return "", nil
	}

	logger.Debug("获取最新消息ID成功",
		zap.String("stream", streamName),
		zap.String("latest_id", messages[0].ID))

	return messages[0].ID, nil
}

// releaseLock 释放分布式锁
func (q *RedisStreamMessageQueue) releaseLock(ctx context.Context, lockKey string, lockValue string) bool {
	// 使用 Lua 脚本确保原子性释放
	script := redis.NewScript(`
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`)

	result, err := script.Run(ctx, q.client, []string{lockKey}, lockValue).Int()
	if err != nil {
		logger.Warn("释放分布式锁失败",
			zap.String("lock_key", lockKey),
			zap.Error(err))
		return false
	}

	return result == 1
}
