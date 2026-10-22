package agent

import (
	"fmt"
	"strings"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
)

// BuildAttachmentContextSection 将已加载的附件内容拼成可注入 prompt 的字符串。
// 若附件上下文为空，返回一个空段。
func BuildAttachmentContextSection(contexts []attachment.FileContext) string {
	if len(contexts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("附件内容（系统已为你内联，请优先基于以下内容回答；如标记为截断/RAG，可使用 attachment 工具按 file_id 分页读取本轮授权附件）：\n")
	for i, c := range contexts {
		fmt.Fprintf(&sb, "\n[%d] 文件名: %s\n", i+1, c.Filename)
		if c.FileID != "" {
			fmt.Fprintf(&sb, "    文件 ID: %s\n", c.FileID)
		}
		fmt.Fprintf(&sb, "    加载模式: %s\n", c.Mode)
		if c.Notice != "" {
			fmt.Fprintf(&sb, "    提示: %s\n", c.Notice)
		}
		if c.Content != "" {
			fmt.Fprintf(&sb, "    内容:\n%s\n", c.Content)
		}
	}
	return sb.String()
}
