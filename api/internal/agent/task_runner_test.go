package agent

import (
	"bytes"
	"context"
	"github.com/bytedance/sonic"
	"io"
	"testing"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

func TestSafeMarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "正常序列化字符串",
			input:   map[string]interface{}{"key": "value"},
			wantErr: false,
		},
		{
			name:    "序列化嵌套结构",
			input:   map[string]interface{}{"key": map[string]interface{}{"nested": "value"}},
			wantErr: false,
		},
		{
			name:    "序列化 nil 值",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "序列化数组",
			input:   []int{1, 2, 3},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeMarshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("safeMarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// 验证结果可以反序列化
				var decoded interface{}
				if err := sonic.Unmarshal(result, &decoded); err != nil {
					t.Errorf("safeMarshal() result = %s, is not valid JSON", string(result))
				}
			}
		})
	}
}

func TestPopRetryConfig(t *testing.T) {
	// 测试重试配置常量的合理性
	if popRetryBaseDelay <= 0 {
		t.Errorf("popRetryBaseDelay should be positive, got %v", popRetryBaseDelay)
	}

	if popRetryMaxDelay <= 0 {
		t.Errorf("popRetryMaxDelay should be positive, got %v", popRetryMaxDelay)
	}

	if popRetryMaxDelay < popRetryBaseDelay {
		t.Errorf("popRetryMaxDelay (%v) should be >= popRetryBaseDelay (%v)", popRetryMaxDelay, popRetryBaseDelay)
	}

	if popRetryMaxCount <= 0 {
		t.Errorf("popRetryMaxCount should be positive, got %d", popRetryMaxCount)
	}
}

func TestMustMarshal_BackwardCompatibility(t *testing.T) {
	// 测试 mustMarshal 向后兼容性
	result := mustMarshal(map[string]interface{}{"key": "value"})
	if len(result) == 0 {
		t.Error("mustMarshal() should return non-empty result for valid input")
	}

	// mustMarshal 不应该 panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("mustMarshal() panicked: %v", r)
		}
	}()

	// 测试无效输入（channel 无法序列化）
	chanInput := make(chan int)
	_ = mustMarshal(chanInput) // 应该返回 "null"
}

type attachmentFileRepository struct {
	repository.FileRepository
	files map[string]*model.File
}

func (r *attachmentFileRepository) GetByID(ctx context.Context, id string) (*model.File, error) {
	return r.files[id], nil
}

func TestAgentService_ResolveMessageAttachments(t *testing.T) {
	fileRepo := &attachmentFileRepository{
		files: map[string]*model.File{
			"file-1": {
				ID:       "file-1",
				Filename: "input.txt",
				Key:      "files/session/file-1.txt",
			},
		},
	}
	agentService := &AgentService{fileRep: fileRepo}

	got := agentService.resolveMessageAttachments(
		context.Background(),
		"session-1",
		[]string{" file-1 ", "", "missing"},
	)

	if len(got) != 1 {
		t.Fatalf("attachments = %d, want 1", len(got))
	}
	if got[0].ID != "file-1" || got[0].Key != "files/session/file-1.txt" {
		t.Fatalf("attachment = %+v, want file-1 metadata", got[0])
	}
}

func TestMessageTool_NotifyUserSchema(t *testing.T) {
	tool := NewMessageTool()
	functions := tool.GetTools()
	if len(functions) == 0 {
		t.Fatal("message tool schema is empty")
	}
	if functions[0]["name"] != "message_notify_user" {
		t.Fatalf("function name = %v, want message_notify_user", functions[0]["name"])
	}

	result, err := tool.InvokeWithName(
		"message_notify_user",
		context.Background(),
		map[string]interface{}{"text": "已完成"},
	)
	if err != nil {
		t.Fatalf("InvokeWithName() error = %v", err)
	}
	if !result.Success {
		t.Fatalf("InvokeWithName() result = %+v, want success", result)
	}
}

type attachmentStorage struct {
	data []byte
}

func (s *attachmentStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	return nil
}

func (s *attachmentStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.data)), nil
}

func (s *attachmentStorage) GetURL(ctx context.Context, key string) (string, error) {
	return "", nil
}

type attachmentSandbox struct {
	external.Sandbox
	filepath string
	filename string
	data     []byte
}

func (s *attachmentSandbox) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	s.filepath = filepath
	s.filename = filename
	s.data = append([]byte(nil), fileData...)
	return model.NewToolResult(nil), nil
}

func TestAgentTaskRunner_SyncUserAttachmentsToSandbox(t *testing.T) {
	sandbox := &attachmentSandbox{}
	runner := &AgentTaskRunner{
		sessionID:   "session-1",
		fileStorage: &attachmentStorage{data: []byte("file content")},
		attLoader:   attachment.NewLoader(&attachmentStorage{data: []byte("file content")}),
		sandbox:     sandbox,
	}

	got, err := runner.syncUserAttachmentsToSandbox(context.Background(), []model.File{
		{
			ID:       "file-1",
			Filename: "../../input.txt",
			Key:      "files/session-1/file-1.txt",
		},
	})
	if err != nil {
		t.Fatalf("syncUserAttachmentsToSandbox() error = %v", err)
	}

	wantPath := "/home/ubuntu/upload/session-1/input.txt"
	if len(got) != 1 || got[0] != wantPath {
		t.Fatalf("attachments = %v, want [%s]", got, wantPath)
	}
	if sandbox.filepath != wantPath {
		t.Fatalf("uploaded filepath = %s, want %s", sandbox.filepath, wantPath)
	}
	if string(sandbox.data) != "file content" {
		t.Fatalf("uploaded data = %q, want %q", string(sandbox.data), "file content")
	}
}
