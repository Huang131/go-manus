package agent

import (
	"strings"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/agent/attachment"
)

func TestBuildAttachmentContextSection_Empty(t *testing.T) {
	if got := BuildAttachmentContextSection(nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestBuildAttachmentContextSection_Typed(t *testing.T) {
	ctx := []attachment.FileContext{
		{Filename: "a.md", Filepath: "/a.md", Mode: attachment.ModeInline, Content: "正文"},
	}
	got := BuildAttachmentContextSection(ctx)
	if !strings.Contains(got, "a.md") {
		t.Errorf("missing filename")
	}
	if !strings.Contains(got, "正文") {
		t.Errorf("missing content")
	}
	if !strings.Contains(got, "inline") {
		t.Errorf("missing mode")
	}
}

func TestBuildAttachmentContextSection_FromMap(t *testing.T) {
	// 模拟从 JSON 反序列化后传入
	m := []map[string]interface{}{
		{
			"filename": "b.md",
			"filepath": "/b.md",
			"mode":     "truncated",
			"content":  "头...尾",
			"notice":   "中文件",
		},
	}
	got := BuildAttachmentContextSection(m)
	if !strings.Contains(got, "头...尾") {
		t.Errorf("missing content: %s", got)
	}
	if !strings.Contains(got, "truncated") {
		t.Errorf("missing mode")
	}
}
