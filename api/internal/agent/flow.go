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
	FlowStatusFailed      FlowStatus = "failed"      // 已失败
)

// ToSessionStatus 把流状态投影成会话状态，作为三套状态（Flow/Session/Execution）之间唯一的投影入口。
// 终态 completed 需结合计划是否含失败步骤：有失败步骤则判 failed，否则 completed，
// 避免业务失败在会话层被误报为完成。
func (s FlowStatus) ToSessionStatus(plan *model.Plan) model.SessionStatus {
	switch s {
	case FlowStatusWaiting:
		return model.SessionStatusWaiting
	case FlowStatusFailed:
		return model.SessionStatusFailed
	case FlowStatusCompleted:
		if planHasFailedStep(plan) {
			return model.SessionStatusFailed
		}
		return model.SessionStatusCompleted
	default:
		return model.SessionStatusRunning
	}
}
