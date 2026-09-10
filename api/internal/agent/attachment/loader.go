package attachment

import (
	"context"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/Huang131/go-manus/api/pkg/logger"

	"github.com/Huang131/go-manus/api/internal/model"
)

// FileStorage 文件存储最小接口（由 agent.COSFileStorage 实现）
type FileStorage interface {
	Download(ctx context.Context, key string) (io.ReadCloser, error)
}

// Loader 将用户附件转为 LLM 可见的 FileContext
type Loader struct {
	storage FileStorage
	ranker  *Ranker
}

// NewLoader 构造 Loader
func NewLoader(storage FileStorage) *Loader {
	return &Loader{
		storage: storage,
		ranker:  NewRanker(),
	}
}

// Load 加载所有附件并返回 FileContext 列表。
// 任何单个附件失败都不会中断整体加载。
func (l *Loader) Load(ctx context.Context, files []model.File, userMessage string) []FileContext {
	if len(files) == 0 {
		return nil
	}
	out := make([]FileContext, 0, len(files))
	budget := InlineContextBudget
	for _, f := range files {
		fc := l.loadOne(ctx, f, userMessage, &budget)
		out = append(out, fc)
	}
	return out
}

func (l *Loader) loadOne(ctx context.Context, f model.File, userMessage string, budget *int) FileContext {
	logger.InfoContext(ctx, "加载附件",
		logger.String("file_id", f.ID),
		logger.String("filename", f.Filename),
		logger.Int64("size", f.Size),
		logger.String("mime", f.MimeType))

	if IsLikelyBinary(f.MimeType) {
		return FileContext{
			Filename: f.Filename,
			Filepath: f.Filepath,
			MimeType: f.MimeType,
			Mode:     ModeBinary,
			Notice:   "二进制文件，已跳过内联；如需分析请告知",
		}
	}

	if f.ID == "" || f.Key == "" {
		return FileContext{
			Filename: f.Filename,
			Filepath: f.Filepath,
			MimeType: f.MimeType,
			Mode:     ModeSkipped,
			Notice:   "缺少文件 key，无法读取",
		}
	}

	reader, err := l.storage.Download(ctx, f.Key)
	if err != nil {
		logger.WarnContext(ctx, "下载附件失败",
			logger.String("file_id", f.ID),
			logger.Err(err))
		return FileContext{
			Filename: f.Filename,
			Filepath: f.Filepath,
			MimeType: f.MimeType,
			Mode:     ModeSkipped,
			Notice:   "下载失败：" + err.Error(),
		}
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		logger.WarnContext(ctx, "读取附件内容失败",
			logger.String("file_id", f.ID),
			logger.Err(err))
		return FileContext{
			Filename: f.Filename,
			Filepath: f.Filepath,
			MimeType: f.MimeType,
			Mode:     ModeSkipped,
			Notice:   "读取失败：" + err.Error(),
		}
	}

	content := normalizeText(data)
	size := int64(len(content))

	switch {
	case ShouldInline(size):
		if *budget-len(content) < 0 {
			return FileContext{
				Filename: f.Filename,
				Filepath: f.Filepath,
				MimeType: f.MimeType,
				Mode:     ModeSkipped,
				Notice:   "已超出本轮内联预算，跳过",
			}
		}
		*budget -= len(content)
		return FileContext{
			Filename: f.Filename,
			Filepath: f.Filepath,
			MimeType: f.MimeType,
			Mode:     ModeInline,
			Content:  content,
		}
	case ShouldTruncate(size):
		head := headBytes(content, HeadTailBytes)
		tail := tailBytes(content, HeadTailBytes)
		return FileContext{
			Filename: f.Filename,
			Filepath: f.Filepath,
			MimeType: f.MimeType,
			Mode:     ModeTruncated,
			Content:  head + "\n\n[... 内容已截断 ...]\n\n" + tail,
			Notice:   "中文件，仅保留头尾",
		}
	default:
		chunks := Chunk(content, RAGChunkSize)
		hit := l.ranker.Rank(userMessage, chunks, RAGTopK)
		if len(hit) == 0 {
			// 兜底取首段
			if len(chunks) > 0 {
				hit = chunks[:min(1, len(chunks))]
			}
		}
		body := strings.Join(hit, "\n\n---\n\n")
		return FileContext{
			Filename: f.Filename,
			Filepath: f.Filepath,
			MimeType: f.MimeType,
			Mode:     ModeRAG,
			Content:  body,
			Notice:   "大文件，已按相关性检索 top 段落",
		}
	}
}

// normalizeText 处理编码：UTF-8 完整则用原值，否则退回 latin1
func normalizeText(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	// latin1 → utf-8 等价：每个字节直接当 rune
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

func headBytes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func tailBytes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
