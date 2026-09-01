package agent

import (
	"strings"
	"testing"

	"github.com/mooc-manus/go-manus/api/internal/agent/attachment"
	"github.com/mooc-manus/go-manus/api/internal/model"
)

func TestBuildAttachmentContextSection_IncludesContent(t *testing.T) {
	contexts := []attachment.FileContext{
		{
			Filename: "manual.md",
			Filepath: "/home/ubuntu/upload/manual.md",
			Mode:     attachment.ModeInline,
			Content:  "这里是正文内容",
		},
	}

	got := BuildAttachmentContextSection(contexts)
	if !strings.Contains(got, "manual.md") {
		t.Fatalf("missing filename in prompt context: %s", got)
	}
	if !strings.Contains(got, "这里是正文内容") {
		t.Fatalf("missing content in prompt context: %s", got)
	}
}

func TestCreatePlanPrompt_UsesAttachmentContext(t *testing.T) {
	msg := &model.Message{
		Message: "分析文档",
	}

	prompt := CreatePlanPrompt
	prompt = strings.Replace(prompt, "{message}", msg.Message, 1)
	prompt = strings.Replace(prompt, "{attachments}", "- /home/ubuntu/upload/manual.md\n", 1)
	prompt = strings.Replace(prompt, "{context}", BuildAttachmentContextSection([]attachment.FileContext{
		{
			Filename: "manual.md",
			Filepath: "/home/ubuntu/upload/manual.md",
			Mode:     attachment.ModeInline,
			Content:  "这里是正文内容",
		},
	}), 1)

	if !strings.Contains(prompt, "附件内容") {
		t.Fatalf("expected prompt to contain attachment context: %s", prompt)
	}
	if !strings.Contains(prompt, "这里是正文内容") {
		t.Fatalf("expected prompt to contain attachment content: %s", prompt)
	}
}
