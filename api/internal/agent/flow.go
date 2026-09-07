package agent

import "github.com/Huang131/go-manus/api/internal/model"

// FlowStatus 流状态枚举
type FlowStatus string

const (
	FlowStatusIdle        FlowStatus = "idle"        // 空闲中
	FlowStatusPlanning    FlowStatus = "planning"    // 规划中
	FlowStatusExecuting   FlowStatus = "executing"   // 执行中
	FlowStatusWaiting     FlowStatus = "waiting"     // 等待用户输入
	FlowStatusUpdating    FlowStatus = "updating"    // 更新中
	FlowStatusSummarizing FlowStatus = "summarizing" // 汇总中
	FlowStatusCompleted   FlowStatus = "completed"   // 已完成
)

// BaseFlow 基础流接口
type BaseFlow interface {
	// Invoke 运行流，返回事件流
	Invoke(message *model.Message) <-chan model.BaseEvent
	// Done 返回流是否结束
	Done() bool
}
