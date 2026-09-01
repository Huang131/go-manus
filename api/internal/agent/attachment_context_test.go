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
