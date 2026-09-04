package agent

import (
	"github.com/bytedance/sonic"
	"strings"
	"sync"
	"time"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/pkg/logger"
	"go.uber.org/zap"
)

// Memory 记忆接口
type Memory interface {
	// Add 添加消息到记忆
	Add(msg *model.Message) error
	// GetMessages 获取消息列表
	GetMessages() []*model.Message
	// GetLastN 获取最后 N 条消息
	GetLastN(n int) []*model.Message
	// GetLastMessage 获取最后一条消息
	GetLastMessage() *model.Message
	// RollbackLast 删除最后一条消息
	RollbackLast() error
	// Size 返回记忆大小 (token 数估算)
	Size() int
	// Clear 清空记忆
	Clear()
	// Compact 压缩记忆（保留最近消息）
	Compact(keepCount int) error
	// SmartCompact 智能压缩（移除长输出工具结果）
	SmartCompact() error
	// EstimateTokens 估算消息的 token 数
	EstimateTokens(msg *model.Message) int
}

// 压缩策略类型
type CompressionStrategy int

const (
	StrategyLight      CompressionStrategy = iota // 轻度压缩：移除长输出工具结果
	StrategyMedium                                // 中度压缩：保留最近对话，摘要中间部分
	StrategyAggressive                            // 激进压缩：只保留系统提示和最近对话
)

// 消息重要性权重
var messageImportanceWeights = map[string]float64{
	"system":    1.5, // 系统消息最重要
	"user":      1.2, // 用户消息很重要
	"assistant": 1.0, // 助手消息
	"tool":      0.6, // 工具结果相对不重要
}

// 需要压缩的工具名称列表（长输出工具）
var compressibleTools = map[string]bool{
	"browser_view":     true, // 浏览器访问结果通常很长
	"browser_navigate": true,
	"search":           true, // 搜索结果可能很长
}

// preservedKeywords 重要关键词列表，包含这些关键词的消息权重会增加
var preservedKeywords = []string{
	"error",      // 错误信息
	"warning",    // 警告信息
	"critical",   // 关键信息
	"important",  // 重要信息
	"decision",   // 决策相关
	"conclusion", // 结论
	"result",     // 结果
}

// SimpleMemory 简单记忆实现
type SimpleMemory struct {
	mu       sync.RWMutex
	messages []*model.Message
	maxSize  int
}

// NewSimpleMemory 创建简单记忆
func NewSimpleMemory(maxSize int) *SimpleMemory {
	return &SimpleMemory{
		messages: make([]*model.Message, 0),
		maxSize:  maxSize,
	}
}

func (m *SimpleMemory) Add(msg *model.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *SimpleMemory) GetMessages() []*model.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*model.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *SimpleMemory) GetLastN(n int) []*model.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if n >= len(m.messages) {
		return m.messages
	}
	result := make([]*model.Message, n)
	copy(result, m.messages[len(m.messages)-n:])
	return result
}

func (m *SimpleMemory) GetLastMessage() *model.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.messages) == 0 {
		return nil
	}
	return m.messages[len(m.messages)-1]
}

func (m *SimpleMemory) RollbackLast() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return nil
	}
	m.messages = m.messages[:len(m.messages)-1]
	return nil
}

func (m *SimpleMemory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, msg := range m.messages {
		data, _ := sonic.Marshal(msg)
		total += len(data)
	}
	// 简单估算: 1 token ≈ 4 字符
	return total / 4
}

func (m *SimpleMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]*model.Message, 0)
}

func (m *SimpleMemory) Compact(keepCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.messages) <= keepCount {
		return nil
	}

	// 保留最后 keepCount 条消息
	m.messages = m.messages[len(m.messages)-keepCount:]
	return nil
}

// EstimateTokens 估算消息的 token 数
func (m *SimpleMemory) EstimateTokens(msg *model.Message) int {
	data, _ := sonic.Marshal(msg)
	// 简单估算: 1 token ≈ 4 字符
	return len(data) / 4
}

// SmartCompact 智能压缩
// 参考 Python 版本的 Memory.compact() 方法
// 移除长输出工具结果和 reasoning_content
func (m *SimpleMemory) SmartCompact() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	removedCount := 0
	for _, msg := range m.messages {
		// 1. 如果是 tool 角色的消息，检查是否是长输出工具
		if msg.Role == "tool" && len(msg.ToolCalls) > 0 {
			tc := msg.ToolCalls[0]
			if compressibleTools[tc.Function.Name] {
				// 替换内容为 "(removed)"
				originalLen := len(msg.Message)
				msg.Message = "(removed)"
				removedCount++
				logger.Debug("从记忆中移除对应工具的结果",
					zap.String("function_name", tc.Function.Name),
					zap.Int("original_length", originalLen))
			}
		}

		// 2. reasoning_content 已在 LLMResponse.ReasoningContent 字段归一化，
		//    不会写入 ToolCall 结构；此处无需再清理。
	}

	if removedCount > 0 {
		logger.Info("智能压缩记忆完成",
			zap.Int("removed_count", removedCount),
			zap.Int("total_messages", len(m.messages)))
	}

	return nil
}

// SmartMemory 智能记忆实现（支持智能压缩）
type SmartMemory struct {
	*SimpleMemory
	strategy           CompressionStrategy // 当前压缩策略
	lastCompactTime    time.Time           // 上次压缩时间
	compressionHistory []CompressionRecord // 压缩历史记录
	lightThreshold     int                 // 轻度压缩阈值（token数）
	mediumThreshold    int                 // 中度压缩阈值（token数）
}

// CompressionRecord 压缩记录
type CompressionRecord struct {
	Time         time.Time           `json:"time"`
	Strategy     CompressionStrategy `json:"strategy"`
	BeforeSize   int                 `json:"before_size"`   // 压缩前的消息数
	AfterSize    int                 `json:"after_size"`    // 压缩后的消息数
	BeforeTokens int                 `json:"before_tokens"` // 压缩前的token数
	AfterTokens  int                 `json:"after_tokens"`  // 压缩后的token数
	RemovedCount int                 `json:"removed_count"` // 删除的消息数
}

// NewSmartMemory 创建智能记忆
func NewSmartMemory(maxSize int) *SmartMemory {
	return &SmartMemory{
		SimpleMemory:       NewSimpleMemory(maxSize),
		strategy:           StrategyLight,
		compressionHistory: make([]CompressionRecord, 0),
		lightThreshold:     maxSize * 3 / 4,  // 轻度压缩阈值：maxSize的75%
		mediumThreshold:    maxSize * 9 / 10, // 中度压缩阈值：maxSize的90%
	}
}

// GetCompressionHistory 获取压缩历史记录
func (m *SmartMemory) GetCompressionHistory() []CompressionRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]CompressionRecord, len(m.compressionHistory))
	copy(result, m.compressionHistory)
	return result
}

// GetCurrentStrategy 获取当前压缩策略
func (m *SmartMemory) GetCurrentStrategy() CompressionStrategy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.strategy
}

// SetCompressionStrategy 设置压缩策略
func (m *SmartMemory) SetCompressionStrategy(strategy CompressionStrategy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategy = strategy
	logger.Info("设置压缩策略", zap.Int("strategy", int(strategy)))
}

// AssessMessageImportance 评估消息重要性
func (m *SimpleMemory) AssessMessageImportance(msg *model.Message) float64 {
	importance := messageImportanceWeights[msg.Role]
	if importance == 0 {
		importance = 0.8 // 默认权重
	}

	// 拼装内容：tool 消息还会带上 arguments（用于重要性判定）
	content := msg.Message
	if msg.Role == "tool" && len(msg.ToolCalls) > 0 {
		for _, tc := range msg.ToolCalls {
			if args := tc.Function.Arguments; args != "" {
				content += args
			}
		}
	}

	// 如果包含重要关键词，增加权重
	for _, keyword := range preservedKeywords {
		if strings.Contains(content, keyword) {
			importance *= 1.3
			break
		}
	}

	// 如果是长消息，稍微降低权重（因为可能包含冗余信息）
	tokenCount := m.EstimateTokens(msg)
	if tokenCount > 500 {
		importance *= 0.9
	}

	return importance
}

// CalculateDynamicThreshold 计算动态压缩阈值
func (m *SmartMemory) CalculateDynamicThreshold() (threshold int, strategy CompressionStrategy) {
	currentSize := m.Size()
	totalMessages := len(m.messages)

	// 根据当前记忆大小和消息数量选择压缩策略
	if currentSize < m.lightThreshold && totalMessages < 30 {
		return m.lightThreshold, StrategyLight
	} else if currentSize < m.mediumThreshold && totalMessages < 50 {
		return m.mediumThreshold, StrategyMedium
	} else {
		return m.maxSize / 2, StrategyAggressive // 激进压缩：只保留一半
	}
}

// ShouldAutoCompact 判断是否需要自动压缩
func (m *SmartMemory) ShouldAutoCompact() (bool, CompressionStrategy) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	currentSize := m.Size()
	threshold, strategy := m.CalculateDynamicThreshold()

	// 检查是否超过阈值
	if currentSize > threshold {
		return true, strategy
	}

	// 检查是否长时间没有压缩（超过1小时）
	if time.Since(m.lastCompactTime) > time.Hour && currentSize > m.lightThreshold/2 {
		return true, StrategyLight
	}

	return false, StrategyLight
}

// SmartCompact 智能压缩（根据当前状态自动选择压缩策略）
func (m *SmartMemory) SmartCompact() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	beforeSize := len(m.messages)
	beforeTokens := m.Size()

	// 1. 先进行轻度压缩（移除长输出工具结果和思考内容）
	m.SimpleMemory.SmartCompact()

	// 2. 根据当前状态选择压缩策略
	shouldCompact, strategy := m.CalculateDynamicThreshold()
	currentTokens := m.Size()

	if currentTokens > shouldCompact || strategy == StrategyAggressive {
		switch strategy {
		case StrategyLight:
			m.compactLight()
		case StrategyMedium:
			m.compactMedium()
		case StrategyAggressive:
			m.compactAggressive()
		}
	}

	// 3. 更新压缩记录
	afterSize := len(m.messages)
	afterTokens := m.Size()

	record := CompressionRecord{
		Time:         time.Now(),
		Strategy:     strategy,
		BeforeSize:   beforeSize,
		AfterSize:    afterSize,
		BeforeTokens: beforeTokens,
		AfterTokens:  afterTokens,
		RemovedCount: beforeSize - afterSize,
	}
	m.compressionHistory = append(m.compressionHistory, record)
	m.lastCompactTime = time.Now()

	// 保留最近的10条压缩记录
	if len(m.compressionHistory) > 10 {
		m.compressionHistory = m.compressionHistory[len(m.compressionHistory)-10:]
	}

	logger.Info("智能压缩记忆完成",
		zap.Int("strategy", int(strategy)),
		zap.Int("before_size", beforeSize),
		zap.Int("after_size", afterSize),
		zap.Int("before_tokens", beforeTokens),
		zap.Int("after_tokens", afterTokens),
		zap.Int("removed_count", beforeSize-afterSize))

	return nil
}

// compactLight 轻度压缩：移除长输出工具结果
func (m *SmartMemory) compactLight() {
	// 已经由 SimpleMemory.SmartCompact() 处理
	// 这里可以添加额外的轻度压缩逻辑
}

// compactMedium 中度压缩：移除旧的工具结果，保留关键对话
func (m *SmartMemory) compactMedium() {
	// 找到最近的用户消息位置
	lastUserIdx := -1
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			lastUserIdx = i
			break
		}
	}

	// 保留从最后一条用户消息往前的所有消息
	if lastUserIdx > 0 {
		// 计算需要保留的消息数（大约保留最近20条）
		keepCount := min(20, len(m.messages))
		if lastUserIdx > len(m.messages)-keepCount {
			// 如果最后一条用户消息在需要保留的范围内，直接截断
			m.messages = m.messages[len(m.messages)-keepCount:]
		} else {
			// 保留最后一条用户消息及其之后的所有消息
			m.messages = m.messages[lastUserIdx:]
		}
	}
}

// compactAggressive 激进压缩：只保留系统消息和最近对话
func (m *SmartMemory) compactAggressive() {
	// 找到系统消息
	systemMessages := make([]*model.Message, 0)
	nonSystemMessages := make([]*model.Message, 0)

	for _, msg := range m.messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			nonSystemMessages = append(nonSystemMessages, msg)
		}
	}

	// 只保留最近10条非系统消息
	keepCount := 10
	if len(nonSystemMessages) > keepCount {
		nonSystemMessages = nonSystemMessages[len(nonSystemMessages)-keepCount:]
	}

	// 合并消息：系统消息在前，非系统消息在后
	m.messages = append(systemMessages, nonSystemMessages...)

	logger.Debug("激进压缩完成",
		zap.Int("system_messages", len(systemMessages)),
		zap.Int("non_system_messages", len(nonSystemMessages)))
}
