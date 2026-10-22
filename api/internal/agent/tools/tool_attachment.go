package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/Huang131/go-manus/api/internal/model"
)

const ToolNameAttachment = "attachment"

// AttachmentReader 是对象存储附件读取的最小边界。实现必须在服务端校验会话归属。
type AttachmentReader interface {
	DownloadAttachment(context.Context, string, string) (AttachmentInfo, io.ReadCloser, error)
}

type AttachmentInfo struct{ ID, Filename, MimeType string }

// AttachmentTool 仅允许本次 Run 的附件白名单，按行分页读取，不接收存储路径。
type AttachmentTool struct {
	reader    AttachmentReader
	sessionID string
	allowed   map[string]struct{}
}

func NewAttachmentTool(reader AttachmentReader, sessionID string, fileIDs []string) *AttachmentTool {
	allowed := make(map[string]struct{}, len(fileIDs))
	for _, id := range fileIDs {
		if id != "" {
			allowed[id] = struct{}{}
		}
	}
	return &AttachmentTool{reader: reader, sessionID: sessionID, allowed: allowed}
}
func (t *AttachmentTool) Name() string { return ToolNameAttachment }
func (t *AttachmentTool) Description() string {
	return "按 file_id 分页读取本轮用户明确授权的会话文本附件。"
}
func (t *AttachmentTool) ReadOnly() bool { return true }
func (t *AttachmentTool) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{
		"file_id": map[string]interface{}{"type": "string", "description": "附件 ID"},
		"offset":  map[string]interface{}{"type": "integer", "minimum": 0, "description": "起始行号，从 0 开始"},
		"limit":   map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 200, "description": "最多返回行数，默认 50"},
	}, "required": []string{"file_id"}}
}
func (t *AttachmentTool) Invoke(ctx context.Context, params map[string]interface{}) (*model.ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return model.NewToolError("附件读取已取消"), nil
	}
	id, ok := params["file_id"].(string)
	if !ok || strings.TrimSpace(id) == "" {
		return model.NewToolError("file_id 必须是非空字符串"), nil
	}
	if _, ok := t.allowed[id]; !ok {
		return model.NewToolError("该附件未在本轮消息中授权"), nil
	}
	offset, err := attachmentInt(params, "offset", 0, 0, 1<<31-1)
	if err != nil {
		return model.NewToolError(err.Error()), nil
	}
	limit, err := attachmentInt(params, "limit", 50, 1, 200)
	if err != nil {
		return model.NewToolError(err.Error()), nil
	}
	info, reader, err := t.reader.DownloadAttachment(ctx, t.sessionID, id)
	if err != nil {
		return model.NewToolError("读取附件失败: " + err.Error()), nil
	}
	defer reader.Close()
	if info.MimeType != "" && !strings.HasPrefix(info.MimeType, "text/") && info.MimeType != "application/json" && info.MimeType != "application/xml" && info.MimeType != "application/x-yaml" && info.MimeType != "application/yaml" {
		return model.NewToolError("二进制附件不支持文本读取"), nil
	}
	scanner := bufio.NewScanner(reader)
	const maxPageBytes = 96 * 1024
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	lines := make([]string, 0, limit+1)
	pageBytes := 0
	more := false
	for index := 0; scanner.Scan(); index++ {
		if err := ctx.Err(); err != nil {
			return model.NewToolError("附件读取已取消"), nil
		}
		if index >= offset {
			line := scanner.Text()
			if len(lines) >= limit || pageBytes+len(line) > maxPageBytes {
				more = true
				break
			}
			lines = append(lines, line)
			pageBytes += len(line) + 1
		}
	}
	if err := scanner.Err(); err != nil {
		return model.NewToolError("读取附件内容失败: " + err.Error()), nil
	}
	eof := !more
	return model.NewToolResult(map[string]interface{}{"file_id": id, "filename": info.Filename, "offset": offset, "lines": lines, "next_offset": offset + len(lines), "eof": eof}), nil
}
func attachmentInt(params map[string]interface{}, name string, fallback, min, max int) (int, error) {
	v, exists := params[name]
	if !exists {
		return fallback, nil
	}
	n, ok := v.(float64)
	if !ok || math.IsNaN(n) || math.IsInf(n, 0) || n < float64(min) || n > float64(max) || math.Trunc(n) != n {
		return 0, fmt.Errorf("%s 必须是 %d 到 %d 的整数", name, min, max)
	}
	return int(n), nil
}
