// Package attachment 提供附件内容加载与摘要能力，
// 使 LLM 在回答用户问题前能"看到"上传文件的真实内容，
// 而不仅是文件路径。
package attachment

// 尺寸阈值（字节）
const (
	// InlineMaxBytes 小于等于该阈值的文件会被完整内联到 prompt
	InlineMaxBytes = 8 * 1024

	// TruncateMaxBytes 小于等于该阈值的文件保留头尾片段
	TruncateMaxBytes = 64 * 1024

	// HeadTailBytes 截断模式下文件头与尾各保留的字节数
	HeadTailBytes = 2 * 1024

	// RAGChunkSize RAG 段落切分粒度
	RAGChunkSize = 1024

	// RAGTopK RAG 检索返回的段落数
	RAGTopK = 5

	// InlineContextBudget 单个会话所有附件内联进 prompt 的总预算（字节）
	// 超出预算时优先保持已加载的，剩余跳过
	InlineContextBudget = 32 * 1024
)

// LoadMode 加载模式
type LoadMode string

const (
	ModeInline    LoadMode = "inline"    // 小文件：完整内联
	ModeTruncated LoadMode = "truncated" // 中文件：头尾保留
	ModeRAG       LoadMode = "rag"       // 大文件：RAG 检索
	ModeSkipped   LoadMode = "skipped"   // 跳过（不可读）
	ModeBinary    LoadMode = "binary"    // 二进制文件：仅保留元信息
)

// FileContext 单个附件加载后的内容片段，供 LLM 消费
type FileContext struct {
	Filename string   `json:"filename"`
	Filepath string   `json:"filepath"`
	MimeType string   `json:"mime_type"`
	Mode     LoadMode `json:"mode"`
	Content  string   `json:"content"`
	Notice   string   `json:"notice,omitempty"` // 提示给 LLM 的说明
}

// ShouldInline 判断文件大小是否属于"小文件"档
func ShouldInline(size int64) bool { return size > 0 && size <= InlineMaxBytes }

// ShouldTruncate 判断文件大小是否属于"中文件"档
func ShouldTruncate(size int64) bool {
	return size > InlineMaxBytes && size <= TruncateMaxBytes
}

// ShouldRAG 判断文件大小是否属于"大文件"档
func ShouldRAG(size int64) bool {
	return size > TruncateMaxBytes
}

// IsLikelyBinary 简单判断 MIME 是否为二进制
func IsLikelyBinary(mimeType string) bool {
	if mimeType == "" {
		return false
	}
	switch {
	case len(mimeType) >= 5 && mimeType[:5] == "text/":
		return false
	case mimeType == "application/json",
		mimeType == "application/xml",
		mimeType == "application/javascript",
		mimeType == "application/x-yaml",
		mimeType == "application/yaml":
		return false
	default:
		return true
	}
}
