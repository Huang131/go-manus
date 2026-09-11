package agent

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

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

type attachmentFileRepository struct {
	repository.FileRepository
	files map[string]*model.File
}

type chatContextSessionRepository struct {
	repository.SessionRepository
	mu        sync.Mutex
	appendCtx context.Context
}

func (r *chatContextSessionRepository) GetByID(context.Context, string) (*model.Session, error) {
	return &model.Session{ID: "session-1"}, nil
}

func (r *chatContextSessionRepository) UpdateLatestMessage(context.Context, string, string) error {
	return nil
}

func (r *chatContextSessionRepository) AppendEvent(ctx context.Context, _ string, _ *model.Event) error {
	r.mu.Lock()
	r.appendCtx = ctx
	r.mu.Unlock()
	return nil
}

func (r *chatContextSessionRepository) getAppendContext() context.Context {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.appendCtx
}

func TestAgentServiceChatUsesDetachedContextForMessagePersistence(t *testing.T) {
	repo := &chatContextSessionRepository{}
	svc := &AgentService{
		sessionRep:    repo,
		llm:           &mockLLM{},
		agentConfig:   DefaultAgentConfig(),
		mq:            newInMemoryMessageQueue(),
		taskBySession: make(map[string]*RedisStreamTask),
	}

	requestCtx := context.WithValue(context.Background(), "request-id", "req-1")
	requestCtx, cancel := context.WithCancel(requestCtx)
	cancel()

	_, err := svc.Chat(requestCtx, "session-1", &llmcore.Message{Role: llmcore.RoleUser, ContentText: "hello"})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	appendCtx := repo.getAppendContext()
	if appendCtx == nil {
		t.Fatal("AppendEvent context was nil")
	}
	if appendCtx.Err() != nil {
		t.Fatalf("AppendEvent context error = %v, want nil", appendCtx.Err())
	}
	if got := appendCtx.Value("request-id"); got != "req-1" {
		t.Fatalf("AppendEvent context value = %v, want req-1", got)
	}

	svc.Shutdown()
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

func TestAgentService_GetActiveTaskIDClearsCompletedTask(t *testing.T) {
	agentService := &AgentService{
		taskBySession: make(map[string]*RedisStreamTask),
	}
	session := &model.Session{ID: "session-1"}

	task := NewRedisStreamTask(&mockMQWrapper{}, &mockTaskRunner{})
	agentService.taskBySession[session.ID] = task
	task.Cancel()

	taskID, err := agentService.GetActiveTaskID(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("GetActiveTaskID() error = %v", err)
	}
	if taskID != "" {
		t.Fatalf("GetActiveTaskID() = %q, want empty after task completion", taskID)
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
	data         []byte
	uploadedKey  string
	uploadedSize int64
	uploadedType string
}

func (s *attachmentStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	s.uploadedKey = key
	s.uploadedSize = size
	s.uploadedType = contentType
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

func (s *attachmentSandbox) ReadFile(ctx context.Context, filepath string, startLine, endLine *int, sudo bool, maxLength int) (*model.ToolResult, error) {
	return model.NewToolResult(map[string]interface{}{"content": string(s.data)}), nil
}

type generatedFileRepository struct {
	repository.FileRepository
	created *model.File
}

func (r *generatedFileRepository) Create(ctx context.Context, file *model.File) error {
	r.created = file
	return nil
}

func TestAgentTaskRunner_SyncFileToStorageRegistersMetadata(t *testing.T) {
	storage := &attachmentStorage{data: []byte("generated")}
	fileRepo := &generatedFileRepository{}
	runner := &AgentTaskRunner{
		sessionID:   "session-1",
		fileStorage: storage,
		fileRep:     fileRepo,
		sandbox:     &attachmentSandbox{data: []byte("generated")},
	}

	if err := runner.syncFileToStorage(context.Background(), "/tmp/report.txt"); err != nil {
		t.Fatalf("syncFileToStorage() error = %v", err)
	}
	if fileRepo.created == nil {
		t.Fatal("expected generated file record")
	}
	file := fileRepo.created
	if file.ID == "" || file.SessionID != "session-1" {
		t.Fatalf("file identity = %+v", file)
	}
	if file.Filename != "report.txt" || file.Extension != ".txt" {
		t.Fatalf("file name metadata = %+v", file)
	}
	if file.MimeType != "text/plain; charset=utf-8" || file.Size != int64(len("generated")) {
		t.Fatalf("file content metadata = %+v", file)
	}
	if file.CreatedAt.IsZero() {
		t.Fatal("expected created_at")
	}
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
