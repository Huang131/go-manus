package agent

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/sandbox"
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
	mq := newInMemoryMessageQueue()
	svc := &AgentService{
		repos:         Repositories{Session: repo},
		caps:          Capabilities{LLM: &mockLLM{}, MessageQueue: mq},
		agentConfig:   DefaultAgentConfig(),
		toolsProvider: &ToolProvider{},
		taskBySession: make(map[string]*RedisStreamTask),
	}

	requestCtx := context.WithValue(context.Background(), "request-id", "req-1")
	requestCtx, cancel := context.WithCancel(requestCtx)
	cancel()

	_, err := svc.Chat(requestCtx, "session-1", &llmcore.Message{Role: model.RoleUser, ContentText: "hello"})
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

func (r *attachmentFileRepository) GetBySessionAndID(ctx context.Context, sessionID, id string) (*model.File, error) {
	file := r.files[id]
	if file == nil || file.SessionID != sessionID {
		return nil, nil
	}
	return file, nil
}

func TestAgentService_ResolveMessageAttachments(t *testing.T) {
	fileRepo := &attachmentFileRepository{
		files: map[string]*model.File{
			"file-1": {
				ID:        "file-1",
				SessionID: "session-1",
				Filename:  "input.txt",
				Key:       "files/session/file-1.txt",
			},
		},
	}
	agentService := &AgentService{repos: Repositories{File: fileRepo}}

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

func TestAgentService_ResolveMessageAttachmentsRejectsOtherSession(t *testing.T) {
	fileRepo := &attachmentFileRepository{files: map[string]*model.File{
		"file-a": {ID: "file-a", SessionID: "session-a", Key: "files/session-a/file-a"},
	}}
	service := &AgentService{repos: Repositories{File: fileRepo}}

	got := service.resolveMessageAttachments(context.Background(), "session-b", []string{"file-a"})
	if len(got) != 0 {
		t.Fatalf("attachments = %+v, want no cross-session attachment", got)
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

func (s *attachmentStorage) Delete(ctx context.Context, key string) error {
	return nil
}

type attachmentSandbox struct {
	sandbox.Sandbox
	filepath string
	filename string
	data     []byte
}

// DownloadFile 走二进制通道：产物同步必须字节原样落存储，不能被文本读取截断。
func (s *attachmentSandbox) DownloadFile(ctx context.Context, filepath string) ([]byte, error) {
	return append([]byte(nil), s.data...), nil
}

type generatedFileRepository struct {
	repository.FileRepository
	created *model.File
}

func (r *generatedFileRepository) Create(ctx context.Context, file *model.File) error {
	r.created = file
	return nil
}

func TestSessionRuntime_SyncFileToStorageRegistersMetadata(t *testing.T) {
	storage := &attachmentStorage{data: []byte("generated")}
	fileRepo := &generatedFileRepository{}
	runtime := NewSessionRuntime("session-1", nil, fileRepo, &attachmentSandbox{data: []byte("generated")}, storage)

	if err := runtime.SyncFileToStorage(context.Background(), "/tmp/report.txt"); err != nil {
		t.Fatalf("SyncFileToStorage() error = %v", err)
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
	if storage.uploadedKey != "agent/session-1/report.txt" || storage.uploadedSize != int64(len("generated")) {
		t.Fatalf("uploaded object = key:%q size:%d", storage.uploadedKey, storage.uploadedSize)
	}
	if storage.uploadedType != file.MimeType {
		t.Fatalf("uploaded content type = %q, want %q", storage.uploadedType, file.MimeType)
	}
}

func (s *attachmentSandbox) UploadFile(ctx context.Context, fileData []byte, filepath, filename string) (*model.ToolResult, error) {
	s.filepath = filepath
	s.filename = filename
	s.data = append([]byte(nil), fileData...)
	return model.NewToolResult(nil), nil
}

// TestSessionRuntime_StoreToolArtifactsReplacesBytesWithFileRef 保证 tool_called 事件里
// 只会出现文件引用：原始截图字节落存储后从结果中清除，事件不再被 base64 撑大。
func TestSessionRuntime_StoreToolArtifactsReplacesBytesWithFileRef(t *testing.T) {
	const marker = "PNG-BINARY-MARKER"
	storage := &attachmentStorage{}
	fileRepo := &generatedFileRepository{}
	runtime := NewSessionRuntime("session-1", nil, fileRepo, nil, storage)

	result := model.NewToolResult(map[string]interface{}{"bytes": len(marker)}).
		WithArtifact(browserScreenshotArtifact, model.ToolArtifact{
			Filename: browserScreenshotFilename,
			MimeType: browserScreenshotMimeType,
			Data:     []byte(marker),
		})

	runtime.StoreToolArtifacts(context.Background(), result)

	if len(result.Artifacts) != 0 {
		t.Fatalf("artifacts = %#v, want cleared after persistence", result.Artifacts)
	}
	ref, ok := result.Display[browserScreenshotArtifact].(map[string]interface{})
	if !ok {
		t.Fatalf("display = %#v, want file reference", result.Display)
	}
	fileID, _ := ref["file_id"].(string)
	if fileID == "" || ref["mime_type"] != browserScreenshotMimeType || ref["size"] != int64(len(marker)) {
		t.Fatalf("file reference = %#v", ref)
	}
	if fileRepo.created == nil {
		t.Fatal("expected file record for artifact")
	}
	if fileRepo.created.SessionID != "session-1" || fileRepo.created.MimeType != browserScreenshotMimeType {
		t.Fatalf("file record = %+v", fileRepo.created)
	}
	if fileRepo.created.Extension != ".png" {
		t.Fatalf("file extension = %q, want .png", fileRepo.created.Extension)
	}
	if storage.uploadedType != browserScreenshotMimeType || storage.uploadedSize != int64(len(marker)) {
		t.Fatalf("upload = type:%q size:%d", storage.uploadedType, storage.uploadedSize)
	}
	if !strings.HasPrefix(storage.uploadedKey, "agent/session-1/artifacts/") || !strings.HasSuffix(storage.uploadedKey, ".png") {
		t.Fatalf("uploaded key = %q", storage.uploadedKey)
	}

	// 事件序列化走 JSON()：落盘后必须只带文件引用，不带原始字节。
	if strings.Contains(result.JSON(), marker) {
		t.Fatalf("JSON() leaked artifact bytes: %s", result.JSON())
	}
	if !strings.Contains(result.JSON(), fileID) {
		t.Fatalf("JSON() should carry file reference: %s", result.JSON())
	}
}

// TestSessionRuntime_StoreToolArtifactsDropsWithoutStorage 存储缺失时预览是尽力而为的旁路能力：
// 产物直接丢弃，不影响工具结果本身。
func TestSessionRuntime_StoreToolArtifactsDropsWithoutStorage(t *testing.T) {
	runtime := NewSessionRuntime("session-1", nil, nil, nil, nil)
	result := model.NewToolResult(nil).WithArtifact(browserScreenshotArtifact, model.ToolArtifact{
		Filename: browserScreenshotFilename,
		MimeType: browserScreenshotMimeType,
		Data:     []byte("png"),
	})

	runtime.StoreToolArtifacts(context.Background(), result)

	if len(result.Artifacts) != 0 || len(result.Display) != 0 {
		t.Fatalf("result = %+v, want artifact dropped without storage", result)
	}
}

func TestSessionRuntime_SyncUserAttachmentsToSandbox(t *testing.T) {
	sandbox := &attachmentSandbox{}
	runtime := NewSessionRuntime("session-1", nil, nil, sandbox, &attachmentStorage{data: []byte("file content")})

	got, err := runtime.SyncUserAttachmentsToSandbox(context.Background(), []model.File{
		{
			ID:       "file-1",
			Filename: "../../input.txt",
			Key:      "files/session-1/file-1.txt",
		},
	})
	if err != nil {
		t.Fatalf("SyncUserAttachmentsToSandbox() error = %v", err)
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
