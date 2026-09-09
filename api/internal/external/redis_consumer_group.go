package external

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Huang131/go-manus/api/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	consumerGroupCreateTimeout = 5 * time.Second
	consumerGroupBlockTimeout  = 3 * time.Second
	consumerActiveWindow       = 5 * time.Minute
)

// RedisConsumerGroup Redis 消费者组实现
type RedisConsumerGroup struct {
	name       string
	streamName string
	client     *redis.Client
	consumerID string
	pendingIDs []string
	mu         sync.RWMutex
}

// NewRedisConsumerGroup 创建 Redis 消费者组
func NewRedisConsumerGroup(client *redis.Client, groupName string, streamName string, consumerID string) (*RedisConsumerGroup, error) {
	cg := &RedisConsumerGroup{
		name:       groupName,
		streamName: streamName,
		client:     client,
		consumerID: consumerID,
		pendingIDs: make([]string, 0),
	}

	// 确保消费者组存在
	ctx, cancel := context.WithTimeout(context.Background(), consumerGroupCreateTimeout)
	defer cancel()

	// 尝试创建消费者组（如果已存在会忽略错误）
	err := client.XGroupCreateMkStream(ctx, streamName, groupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		logger.Warn("创建消费者组失败", logger.Err(err))
	}

	return cg, nil
}

// Name 返回消费者组名称
func (cg *RedisConsumerGroup) Name() string {
	return cg.name
}

// StreamName 返回关联的流名称
func (cg *RedisConsumerGroup) StreamName() string {
	return cg.streamName
}

// Claim 认领一条消息
func (cg *RedisConsumerGroup) Claim(ctx context.Context) (string, interface{}, error) {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	// 先检查 Pending 消息
	if len(cg.pendingIDs) > 0 {
		id := cg.pendingIDs[0]
		cg.pendingIDs = cg.pendingIDs[1:]

		// 获取消息内容
		messages, err := cg.client.XRange(ctx, cg.streamName, id, id).Result()
		if err != nil || len(messages) == 0 {
			// 消息不存在，继续尝试获取新消息
			return cg.claimFromStream(ctx)
		}

		return id, messages[0].Values, nil
	}

	return cg.claimFromStream(ctx)
}

// claimFromStream 从流中认领新消息
func (cg *RedisConsumerGroup) claimFromStream(ctx context.Context) (string, interface{}, error) {
	// 使用 XREADGROUP 读取新消息
	streams, err := cg.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    cg.name,
		Consumer: cg.consumerID,
		Streams:  []string{cg.streamName, ">"},
		Count:    1,
		Block:    consumerGroupBlockTimeout,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			// 没有新消息
			return "", nil, nil
		}
		return "", nil, err
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return "", nil, nil
	}

	msg := streams[0].Messages[0]
	return msg.ID, msg.Values, nil
}

// Ack 确认消息已被处理
func (cg *RedisConsumerGroup) Ack(ctx context.Context, messageID string) error {
	// 使用 XACK 命令确认消息
	return cg.client.XAck(ctx, cg.streamName, cg.name, messageID).Err()
}

// Pending 获取 Pending 消息数量
func (cg *RedisConsumerGroup) Pending(ctx context.Context) (int64, error) {
	// 使用 XPENDING 命令获取 Pending 信息
	pending, err := cg.client.XPending(ctx, cg.streamName, cg.name).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

// GetPendingMessages 获取 Pending 消息详情
func (cg *RedisConsumerGroup) GetPendingMessages(ctx context.Context, idleTime time.Duration, count int64) ([]PendingMessage, error) {
	// 使用 XPENDING 命令获取 Pending 消息详情
	pending, err := cg.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: cg.streamName,
		Group:  cg.name,
		Start:  "-",
		End:    "+",
		Count:  count,
		Idle:   idleTime,
	}).Result()

	if err != nil {
		return nil, err
	}

	messages := make([]PendingMessage, 0, len(pending))
	for _, p := range pending {
		// XPendingExt 返回：ID, Consumer, Idle (duration)
		messages = append(messages, PendingMessage{
			ID:           p.ID,
			ConsumerName: p.Consumer,
			IdleSince:    time.Now().Add(-p.Idle),
		})
	}

	return messages, nil
}

// ClaimPending 认领 Pending 消息
func (cg *RedisConsumerGroup) ClaimPending(ctx context.Context, messageID string, minIdleTime time.Duration) error {
	// 使用 XCLAIM 命令认领 Pending 消息
	_, err := cg.client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   cg.streamName,
		Group:    cg.name,
		Consumer: cg.consumerID,
		MinIdle:  minIdleTime,
		Messages: []string{messageID},
	}).Result()

	return err
}

// Info 获取消费者组信息
func (cg *RedisConsumerGroup) Info(ctx context.Context) (*ConsumerGroupInfo, error) {
	// 使用 XINFO GROUPS 命令获取消费者组信息
	groups, err := cg.client.XInfoGroups(ctx, cg.streamName).Result()
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.Name == cg.name {
			info := &ConsumerGroupInfo{
				Name:            g.Name,
				StreamName:      cg.streamName,
				PendingCount:    g.Pending,
				LastDeliveredID: g.LastDeliveredID,
				CreatedAt:       time.Now(),
				Consumers:       make([]ConsumerInfo, 0),
			}

			// 获取消费者列表
			consumers, err := cg.client.XInfoConsumers(ctx, cg.streamName, cg.name).Result()
			if err == nil {
				for _, c := range consumers {
					// XInfoConsumer 返回: Name, Pending, Idle (duration)
					info.Consumers = append(info.Consumers, ConsumerInfo{
						Name:     c.Name,
						Pending:  c.Pending,
						LastSeen: time.Now().Add(-c.Idle),
						Active:   c.Idle < consumerActiveWindow,
					})
				}
			}

			return info, nil
		}
	}

	return nil, fmt.Errorf("消费者组 %s 不存在", cg.name)
}

// Close 关闭消费者组
func (cg *RedisConsumerGroup) Close() error {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.pendingIDs = nil
	return nil
}

// RedisConsumerGroupManager Redis 消费者组管理器
type RedisConsumerGroupManager struct {
	client *redis.Client
	mu     sync.RWMutex
	groups map[string]*RedisConsumerGroup
}

// NewRedisConsumerGroupManager 创建 Redis 消费者组管理器
func NewRedisConsumerGroupManager(client *redis.Client) *RedisConsumerGroupManager {
	return &RedisConsumerGroupManager{
		client: client,
		groups: make(map[string]*RedisConsumerGroup),
	}
}

// CreateGroup 创建消费者组
func (m *RedisConsumerGroupManager) CreateGroup(ctx context.Context, groupName string, streamName string) error {
	// 使用 XGROUP CREATE 命令创建消费者组
	err := m.client.XGroupCreateMkStream(ctx, streamName, groupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// DeleteGroup 删除消费者组
func (m *RedisConsumerGroupManager) DeleteGroup(ctx context.Context, groupName string, streamName string) error {
	// 使用 XGROUP DESTROY 命令删除消费者组
	return m.client.Do(ctx, "XGROUP", "DESTROY", streamName, groupName).Err()
}

// ListGroups 列出流的所有消费者组
func (m *RedisConsumerGroupManager) ListGroups(ctx context.Context, streamName string) ([]string, error) {
	groups, err := m.client.XInfoGroups(ctx, streamName).Result()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(groups))
	for i, g := range groups {
		names[i] = g.Name
	}
	return names, nil
}

// GetGroupInfo 获取消费者组信息
func (m *RedisConsumerGroupManager) GetGroupInfo(ctx context.Context, groupName string, streamName string) (*ConsumerGroupInfo, error) {
	groups, err := m.client.XInfoGroups(ctx, streamName).Result()
	if err != nil {
		return nil, err
	}

	for _, g := range groups {
		if g.Name == groupName {
			info := &ConsumerGroupInfo{
				Name:            g.Name,
				StreamName:      streamName,
				PendingCount:    g.Pending,
				LastDeliveredID: g.LastDeliveredID,
				CreatedAt:       time.Now(),
				Consumers:       make([]ConsumerInfo, 0),
			}

			// 获取消费者列表
			consumers, err := m.client.XInfoConsumers(ctx, streamName, groupName).Result()
			if err == nil {
				for _, c := range consumers {
					info.Consumers = append(info.Consumers, ConsumerInfo{
						Name:     c.Name,
						Pending:  c.Pending,
						LastSeen: time.Now(), // 默认值，实际活跃状态依赖调用方判断
						Active:   true,       // 根据 Idle 时间判断
					})
				}
			}

			return info, nil
		}
	}

	return nil, fmt.Errorf("消费者组 %s 不存在", groupName)
}

// CreateConsumerGroup 创建消费者组实例
func (m *RedisConsumerGroupManager) CreateConsumerGroup(cfg *ConsumerGroupConfig) (ConsumerGroup, error) {
	if cfg.Name == "" || cfg.StreamName == "" {
		return nil, fmt.Errorf("消费者组名称和流名称不能为空")
	}

	key := cfg.StreamName + ":" + cfg.Name

	m.mu.Lock()
	defer m.mu.Unlock()

	if cg, ok := m.groups[key]; ok {
		return cg, nil
	}

	consumerID := cfg.ConsumerID
	if consumerID == "" {
		consumerID = "consumer-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	cg, err := NewRedisConsumerGroup(m.client, cfg.Name, cfg.StreamName, consumerID)
	if err != nil {
		return nil, err
	}

	m.groups[key] = cg
	return cg, nil
}

// GetOrCreateGroup 获取或创建消费者组
func (m *RedisConsumerGroupManager) GetOrCreateGroup(ctx context.Context, groupName string, streamName string) (ConsumerGroup, error) {
	// 先尝试创建组
	m.CreateGroup(ctx, groupName, streamName)

	// 创建消费者组实例
	return m.CreateConsumerGroup(&ConsumerGroupConfig{
		Name:       groupName,
		StreamName: streamName,
	})
}

// RemoveGroup 从管理器中移除消费者组
func (m *RedisConsumerGroupManager) RemoveGroup(groupName string, streamName string) {
	key := streamName + ":" + groupName

	m.mu.Lock()
	defer m.mu.Unlock()

	if cg, ok := m.groups[key]; ok {
		cg.Close()
		delete(m.groups, key)
	}
}

// GetGroup 获取已创建的消费者组
func (m *RedisConsumerGroupManager) GetGroup(groupName string, streamName string) ConsumerGroup {
	key := streamName + ":" + groupName

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.groups[key]
}

// MessageToJSON 将消息转换为 JSON
func MessageToJSON(msg *Message) (string, error) {
	data, err := sonic.Marshal(msg.Data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// JSONToMessage 从 JSON 恢复消息
func JSONToMessage(jsonStr string) (interface{}, error) {
	var data interface{}
	err := sonic.Unmarshal([]byte(jsonStr), &data)
	return data, err
}
