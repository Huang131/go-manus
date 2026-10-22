package tools

import (
	"context"
	"io"
	"strings"
	"testing"
)

type attachmentReaderStub struct {
	session, id string
	calls       int
	content     string
	mime        string
}

func (s *attachmentReaderStub) DownloadAttachment(_ context.Context, session, id string) (AttachmentInfo, io.ReadCloser, error) {
	s.calls++
	s.session, s.id = session, id
	return AttachmentInfo{ID: id, Filename: "手册.md", MimeType: s.mime}, io.NopCloser(strings.NewReader(s.content)), nil
}

func TestAttachmentToolRejectsUnselectedFileBeforeStorage(t *testing.T) {
	reader := &attachmentReaderStub{content: "secret", mime: "text/plain"}
	tool := NewAttachmentTool(reader, "session-1", []string{"selected"})
	result, err := tool.Invoke(context.Background(), map[string]interface{}{"file_id": "other"})
	if err != nil || result.Success || reader.calls != 0 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, reader.calls)
	}
}

func TestAttachmentToolReadsAuthorizedFileInPages(t *testing.T) {
	reader := &attachmentReaderStub{content: "甲\n乙\n丙", mime: "text/markdown"}
	tool := NewAttachmentTool(reader, "session-1", []string{"selected"})
	result, err := tool.Invoke(context.Background(), map[string]interface{}{"file_id": "selected", "offset": float64(1), "limit": float64(1)})
	if err != nil || !result.Success {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	data := result.Data.(map[string]interface{})
	if got := data["lines"].([]string); len(got) != 1 || got[0] != "乙" {
		t.Fatalf("lines=%v", data["lines"])
	}
	if data["next_offset"] != 2 || data["eof"] != false || reader.session != "session-1" || reader.id != "selected" {
		t.Fatalf("data=%v reader=%+v", data, reader)
	}
}

func TestAttachmentToolRejectsInvalidPageAndBinary(t *testing.T) {
	reader := &attachmentReaderStub{content: "x", mime: "application/pdf"}
	tool := NewAttachmentTool(reader, "session-1", []string{"selected"})
	invalid, _ := tool.Invoke(context.Background(), map[string]interface{}{"file_id": "selected", "limit": float64(201)})
	if invalid.Success || reader.calls != 0 {
		t.Fatalf("invalid=%+v calls=%d", invalid, reader.calls)
	}
	binary, _ := tool.Invoke(context.Background(), map[string]interface{}{"file_id": "selected"})
	if binary.Success || reader.calls != 1 {
		t.Fatalf("binary=%+v calls=%d", binary, reader.calls)
	}
}
