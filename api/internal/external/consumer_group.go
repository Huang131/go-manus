package external

import (
	"context"
	"sync"
	"time"
)

// ConsumerGroup 消费者组接口
type ConsumerGroup interface {
	// Name 返回消费者组名称
	Name() string

	// StreamName 返回关联的流名称
	StreamName() string

	// Claim 认领一条消息（从 Pending 列表）
	// 返回: 消息ID, 消息内容, 错误
	Claim(ctx context.Context) (string, interface{}, error)

	// Ack 确认消息已被处理
	// messageID: 消息ID
	Ack(ctx context.Context, messageID string) error

	// Pending 获取 Pending 消息数量
	Pending(ctx context.Context) (int64, error)

	// Info 获取消费者组信息
	Info(ctx context.Context) (*ConsumerGroupInfo, error)

	// Close 关闭消费者组
	Close() error
}

// ConsumerGroupInfo 消费者组信息
type ConsumerGroupInfo struct {
	Name            string         `json:"name"`
	StreamName      string         `json:"stream_name"`
	Consumers       []ConsumerInfo `json:"consumers"`
	PendingCount    int64          `json:"pending_count"`
	LastDeliveredID string         `json:"last_delivered_id"`
	CreatedAt       time.Time      `json:"created_at"`
}

// ConsumerInfo 消费者信息
type ConsumerInfo struct {
	Name     string    `json:"name"`
	Pending  int64     `json:"pending"`
	LastSeen time.Time `json:"last_seen"`
	Active   bool      `json:"active"`
}

// DefaultConsumerGroup 默认消费者组实现
type DefaultConsumerGroup struct {
	name       string
	streamName string
	mq         MessageQueue
	consumerID string
	pendingIDs []string // 本地 Pending 消息ID缓存
	mu         sync.RWMutex
}

// ConsumerGroupConfig 消费者组配置
type ConsumerGroupConfig struct {
	Name       string `json:"name"`        // 消费者组名称
	StreamName string `json:"stream_name"` // 流名称
	ConsumerID string `json:"consumer_id"` // 消费者ID
}

// ConsumerGroupManager 消费者组管理器接口
type ConsumerGroupManager interface {
	// CreateGroup 创建消费者组
	CreateGroup(ctx context.Context, groupName string, streamName string) error

	// DeleteGroup 删除消费者组
	DeleteGroup(ctx context.Context, groupName string, streamName string) error

	// ListGroups 列出流的所有消费者组
	ListGroups(ctx context.Context, streamName string) ([]string, error)

	// GetGroupInfo 获取消费者组信息
	GetGroupInfo(ctx context.Context, groupName string, streamName string) (*ConsumerGroupInfo, error)

	// CreateConsumerGroup 创建消费者组实例
	CreateConsumerGroup(cfg *ConsumerGroupConfig) (ConsumerGroup, error)

	// GetOrCreateGroup 获取或创建消费者组
	GetOrCreateGroup(ctx context.Context, groupName string, streamName string) (ConsumerGroup, error)
}

// PendingMessage Pending 消息结构
type PendingMessage struct {
	ID             string    `json:"id"`
	ConsumerName   string    `json:"consumer_name"`
	DeliveryTime   time.Time `json:"delivery_time"`
	Redelivered    bool      `json:"redelivered"`
	IdleSince      time.Time `json:"idle_since"`
	TimesDelivered int       `json:"times_delivered"`
}

// GroupPendingInfo 消费者组 Pending 信息
type GroupPendingInfo struct {
	GroupName  string           `json:"group_name"`
	TotalCount int64            `json:"total_count"`
	Messages   []PendingMessage `json:"messages"`
}

// GetPendingMessages 获取消费者组的 Pending 消息
// groupName: 消费者组名称
// streamName: 流名称
// idleTime: 空闲时间阈值，只返回空闲时间超过这个值的消息
// count: 最大返回消息数
func GetPendingMessages(ctx context.Context, mq MessageQueue, groupName string, streamName string, idleTime time.Duration, count int64) (*GroupPendingInfo, error) {
	// 这个方法需要在具体的 MQ 实现中调用 XREADGROUP 命令
	// 这里提供接口，具体实现在 Redis 实现中
	return nil, nil
}

// DefaultConsumerGroupManager 默认消费者组管理器实现
type DefaultConsumerGroupManager struct {
	mq     MessageQueue
	mu     sync.RWMutex
	groups map[string]ConsumerGroup // 本地缓存的消费者组
}

// NewConsumerGroupManager 创建消费者组管理器
func NewConsumerGroupManager(mq MessageQueue) ConsumerGroupManager {
	return &DefaultConsumerGroupManager{
		mq:     mq,
		groups: make(map[string]ConsumerGroup),
	}
}

// Name 返回消费者组名称
func (cg *DefaultConsumerGroup) Name() string {
	return cg.name
}

// StreamName 返回关联的流名称
func (cg *DefaultConsumerGroup) StreamName() string {
	return cg.streamName
}

// Claim 认领一条消息（从 Pending 列表）
func (cg *DefaultConsumerGroup) Claim(ctx context.Context) (string, interface{}, error) {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	// 如果有本地缓存的 Pending 消息，优先处理
	if len(cg.pendingIDs) > 0 {
		id := cg.pendingIDs[0]
		cg.pendingIDs = cg.pendingIDs[1:]

		// 尝试获取消息内容
		messages, err := cg.mq.GetRange(ctx, cg.streamName, id, id, 1)
		if err != nil || len(messages) == 0 {
			// 消息可能已被其他消费者处理，继续获取下一条
			return cg.claimNext(ctx)
		}

		return id, messages[0].Data, nil
	}

	return cg.claimNext(ctx)
}

// claimNext 认领下一条消息
func (cg *DefaultConsumerGroup) claimNext(ctx context.Context) (string, interface{}, error) {
	// 尝试从流中获取新消息
	// 这里需要调用 XREADGROUP 命令，具体实现在 Redis 实现中
	// 目前暂时使用阻塞获取
	return cg.mq.GetBlocking(ctx, cg.streamName, "", time.Second*3)
}

// Ack 确认消息已被处理
func (cg *DefaultConsumerGroup) Ack(ctx context.Context, messageID string) error {
	// 从 Pending 列表中移除（ACK 命令由具体 MQ 实现处理）
	// 目前暂时使用删除消息的方式
	return cg.mq.DeleteMessage(ctx, cg.streamName, messageID)
}

// Pending 获取 Pending 消息数量
func (cg *DefaultConsumerGroup) Pending(ctx context.Context) (int64, error) {
	// 调用 XPENDING 命令获取 Pending 消息数量
	// 具体实现在 Redis 实现中
	return cg.mq.Size(ctx, cg.streamName)
}

// Info 获取消费者组信息
func (cg *DefaultConsumerGroup) Info(ctx context.Context) (*ConsumerGroupInfo, error) {
	size, err := cg.Pending(ctx)
	if err != nil {
		return nil, err
	}

	return &ConsumerGroupInfo{
		Name:       cg.name,
		StreamName: cg.streamName,
		Consumers: []ConsumerInfo{
			{
				Name:     cg.consumerID,
				Pending:  size,
				LastSeen: time.Now(),
				Active:   true,
			},
		},
		PendingCount: size,
		CreatedAt:    time.Now(),
	}, nil
}

// Close 关闭消费者组
func (cg *DefaultConsumerGroup) Close() error {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.pendingIDs = nil
	return nil
}

// CreateGroup 创建消费者组（接口方法，需要在具体 MQ 实现中实现）
func (m *DefaultConsumerGroupManager) CreateGroup(ctx context.Context, groupName string, streamName string) error {
	// 需要在 Redis 等具体实现中调用 XGROUP CREATE 命令
	return nil
}

// DeleteGroup 删除消费者组（接口方法，需要在具体 MQ 实现中实现）
func (m *DefaultConsumerGroupManager) DeleteGroup(ctx context.Context, groupName string, streamName string) error {
	// 需要在 Redis 等具体实现中调用 XGROUP DESTROY 命令
	return nil
}

// ListGroups 列出流的所有消费者组（接口方法，需要在具体 MQ 实现中实现）
func (m *DefaultConsumerGroupManager) ListGroups(ctx context.Context, streamName string) ([]string, error) {
	// 需要在 Redis 等具体实现中调用 XINFO GROUPS 命令
	return nil, nil
}

// GetGroupInfo 获取消费者组信息（接口方法，需要在具体 MQ 实现中实现）
func (m *DefaultConsumerGroupManager) GetGroupInfo(ctx context.Context, groupName string, streamName string) (*ConsumerGroupInfo, error) {
	// 需要在 Redis 等具体实现中调用 XINFO GROUPS 命令
	return nil, nil
}

// CreateConsumerGroup 创建消费者组实例
func (m *DefaultConsumerGroupManager) CreateConsumerGroup(cfg *ConsumerGroupConfig) (ConsumerGroup, error) {
	if cfg.Name == "" || cfg.StreamName == "" {
		return nil, nil
	}

	key := cfg.StreamName + ":" + cfg.Name

	m.mu.Lock()
	defer m.mu.Unlock()

	if cg, ok := m.groups[key]; ok {
		return cg, nil
	}

	consumerID := cfg.ConsumerID
	if consumerID == "" {
		consumerID = "consumer-" + time.Now().Format("20060102150405")
	}

	cg := &DefaultConsumerGroup{
		name:       cfg.Name,
		streamName: cfg.StreamName,
		mq:         m.mq,
		consumerID: consumerID,
		pendingIDs: make([]string, 0),
	}

	m.groups[key] = cg
	return cg, nil
}

// GetOrCreateGroup 获取或创建消费者组
func (m *DefaultConsumerGroupManager) GetOrCreateGroup(ctx context.Context, groupName string, streamName string) (ConsumerGroup, error) {
	return m.CreateConsumerGroup(&ConsumerGroupConfig{
		Name:       groupName,
		StreamName: streamName,
	})
}
