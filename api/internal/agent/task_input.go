package agent

import (
	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/llmcore"
)

// TaskInput 一次任务执行的用户输入（运行期载体）。
//
// Message 进入记忆与 LLM 上下文；AttachmentContexts 是 AttachmentLoader
// 预加载的附件正文，仅用于 prompt 注入，不持久化、不进入记忆。
type TaskInput struct {
	// Message 用户消息（user 角色）
	Message llmcore.Message
	// AttachmentContexts 已加载到 LLM 上下文的附件内容
	AttachmentContexts []attachment.FileContext
}
