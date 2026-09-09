package agent

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
