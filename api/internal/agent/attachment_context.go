package agent

import (
	"fmt"
	"strings"

	"github.com/mooc-manus/go-manus/api/internal/agent/attachment"
)

// BuildAttachmentContextSection 将已加载的附件内容拼成可注入 prompt 的字符串。
// 若附件上下文为空，返回一个空段。
func BuildAttachmentContextSection(raw interface{}) string {
	if raw == nil {
		return ""
	}
	contexts, ok := raw.([]attachment.FileContext)
	if !ok || len(contexts) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("附件内容（系统已为你内联，请优先基于以下内容回答；如下方为截断/RAG/二进制状态，可再调用 file.read 补全）：\n")
	for i, c := range contexts {
		fmt.Fprintf(&sb, "\n[%d] 文件名: %s\n", i+1, c.Filename)
		fmt.Fprintf(&sb, "    路径: %s\n", c.Filepath)
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
