package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/settings"
)

type applicationRunStoreStub struct {
	run           *model.Run
	createInput   CreateRunInput
	resume        *RunResume
	submittedRun  string
	history       []*model.Run
	historyTotal  int
	messages      map[string][]*model.RunMessage
	batchCalls    int
	singleCalls   int
	historyLimit  int
	historyOffset int
	getErr        error
}

func (s *applicationRunStoreStub) Create(_ context.Context, input CreateRunInput) (*model.Run, error) {
	s.createInput = input
	if s.run == nil {
		s.run = &model.Run{ID: "run-1", SessionID: input.SessionID, Status: model.RunStatusPending}
	}
	return s.run, nil
}

func (s *applicationRunStoreStub) SubmitInput(_ context.Context, runID string, _ SubmitInputRequest) (*RunResume, error) {
	s.submittedRun = runID
	return s.resume, nil
}

func (s *applicationRunStoreStub) Get(_ context.Context, _ string) (*model.Run, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.run, nil
}

func (s *applicationRunStoreStub) GetActiveBySessionID(_ context.Context, sessionID string) (*model.Run, error) {
	if s.run == nil || s.run.SessionID != sessionID || !s.run.Status.IsActive() {
		return nil, nil
	}
	return s.run, nil
}

func (s *applicationRunStoreStub) ListBySessionID(_ context.Context, _ string, limit, offset int) ([]*model.Run, int, error) {
	s.historyLimit, s.historyOffset = limit, offset
	return s.history, s.historyTotal, nil
}

func (s *applicationRunStoreStub) RequestCancel(_ context.Context, _ string) (*model.Run, error) {
	return s.run, nil
}

func (s *applicationRunStoreStub) ListMessages(_ context.Context, runID string) ([]*model.RunMessage, error) {
	s.singleCalls++
	return s.messages[runID], nil
}

func (s *applicationRunStoreStub) ListMessagesByRunIDs(_ context.Context, runIDs []string) (map[string][]*model.RunMessage, error) {
	s.batchCalls++
	result := make(map[string][]*model.RunMessage, len(runIDs))
	for _, runID := range runIDs {
		result[runID] = append([]*model.RunMessage(nil), s.messages[runID]...)
	}
	return result, nil
}

type applicationExecutorStub struct {
	startCalls      int
	startedRun      string
	startedMessages []llmcore.Message
	contexts        []attachment.FileContext
	cancelledRun    string
}

func (s *applicationExecutorStub) Start(_ context.Context, runID string, messages []llmcore.Message, contexts ...[]attachment.FileContext) (*RunExecutionHandle, error) {
	s.startCalls++
	s.startedRun = runID
	s.startedMessages = append([]llmcore.Message(nil), messages...)
	if len(contexts) > 0 {
		s.contexts = append([]attachment.FileContext(nil), contexts[0]...)
	}
	handle := &RunExecutionHandle{done: make(chan struct{})}
	close(handle.done)
	return handle, nil
}

func (s *applicationExecutorStub) Cancel(_ context.Context, runID string) (*model.Run, error) {
	s.cancelledRun = runID
	return &model.Run{ID: runID, Status: model.RunStatusCancelling}, nil
}

type sessionLookupStub struct{ session *model.Session }

func (s sessionLookupStub) GetByID(context.Context, string) (*model.Session, error) {
	return s.session, nil
}

type fileLookupStub struct{ files map[string]*model.File }

func (s fileLookupStub) GetBySessionAndID(_ context.Context, sessionID, id string) (*model.File, error) {
	file := s.files[id]
	if file == nil || file.SessionID != sessionID {
		return nil, nil
	}
	return file, nil
}

func TestRunApplicationServiceSubmitInputRetryDoesNotRevalidateDeletedAttachments(t *testing.T) {
	store, _, _ := waitingRunFixture()
	runs := NewRunService(store)
	executor := &applicationExecutorStub{}
	files := fileLookupStub{files: map[string]*model.File{
		"file-1": {ID: "file-1", SessionID: "session-1", Filename: "extra.txt"},
	}}
	service := NewRunApplicationService(
		runs, executor, sessionLookupStub{session: &model.Session{ID: "session-1"}},
		&files, settingsSourceStub{}, &attachmentLoaderStub{}, "prompt-hash",
	)
	input := SubmitInputRequest{IdempotencyKey: "reply-1", ReplyToMessageID: "question-1", Content: "补充信息", Attachments: []string{"file-1"}}
	if _, err := service.SubmitInput(context.Background(), "run-1", input); err != nil {
		t.Fatalf("first SubmitInput() error = %v", err)
	}
	files.files = nil
	result, err := service.SubmitInput(context.Background(), "run-1", input)
	if err != nil {
		t.Fatalf("idempotent SubmitInput() error = %v", err)
	}
	if result.Handle != nil || result.Run == nil || !result.Run.Status.IsActive() {
		t.Fatalf("idempotent SubmitInput() = %+v, want no second execution", result)
	}
	if executor.startCalls != 1 || executor.startedRun != "run-1" || len(store.messages) != 3 {
		t.Fatalf("idempotent retry changed execution/message count: starts=%d executor=%q messages=%d", executor.startCalls, executor.startedRun, len(store.messages))
	}

	input.Content = "不同回答"
	if _, err := service.SubmitInput(context.Background(), "run-1", input); err == nil {
		t.Fatal("SubmitInput() accepted different content with the same idempotency key")
	}
}

type settingsSourceStub struct{ value *settings.AgentSettings }

func (s settingsSourceStub) GetAgentSettings(context.Context) (*settings.AgentSettings, error) {
	return s.value, nil
}

type attachmentLoaderStub struct {
	seenFiles   []model.File
	seenMessage string
	result      []attachment.FileContext
}

func (s *attachmentLoaderStub) Load(_ context.Context, files []model.File, message string) []attachment.FileContext {
	s.seenFiles = append([]model.File(nil), files...)
	s.seenMessage = message
	return append([]attachment.FileContext(nil), s.result...)
}

func TestRunApplicationServiceCreateFreezesSettingsLoadsOwnedAttachmentsAndStarts(t *testing.T) {
	store := &applicationRunStoreStub{}
	executor := &applicationExecutorStub{}
	loader := &attachmentLoaderStub{result: []attachment.FileContext{{Filename: "report.txt", Content: "结论"}}}
	cfg := settings.AgentSettings{MaxIterations: 7, MaxRetries: 2, MaxSearchResults: 5}
	service := NewRunApplicationService(
		store, executor, sessionLookupStub{session: &model.Session{ID: "session-1"}},
		fileLookupStub{files: map[string]*model.File{"file-1": {ID: "file-1", SessionID: "session-1", Filename: "report.txt"}}},
		settingsSourceStub{value: &cfg}, loader, "prompt-hash",
	)

	result, err := service.Create(context.Background(), CreateApplicationRunInput{
		SessionID: "session-1", IdempotencyKey: "request-1", Content: "分析报告", AttachmentIDs: []string{"file-1"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Run.ID != "run-1" || result.Handle == nil {
		t.Fatalf("Create() = %+v, want started run", result)
	}
	if store.createInput.SettingsSnapshot != cfg || store.createInput.PromptHash != "prompt-hash" {
		t.Fatalf("frozen input = %+v, want settings and prompt snapshot", store.createInput)
	}
	if executor.startedRun != "run-1" || len(executor.contexts) != 1 || executor.contexts[0].Content != "结论" {
		t.Fatalf("executor input = run=%q contexts=%+v, want loaded attachment", executor.startedRun, executor.contexts)
	}
	if len(loader.seenFiles) != 1 || loader.seenFiles[0].ID != "file-1" || loader.seenMessage != "分析报告" {
		t.Fatalf("loader input = files=%+v message=%q", loader.seenFiles, loader.seenMessage)
	}
}

func TestRunApplicationServiceCreateReturnsStartedRunWithoutRefresh(t *testing.T) {
	store := &applicationRunStoreStub{getErr: errors.New("refresh created run failed")}
	executor := &applicationExecutorStub{}
	service := NewRunApplicationService(
		store, executor, sessionLookupStub{session: &model.Session{ID: "session-1"}},
		nil, settingsSourceStub{}, nil, "prompt-hash",
	)

	result, err := service.Create(context.Background(), CreateApplicationRunInput{
		SessionID: "session-1", IdempotencyKey: "request-1", Content: "分析报告",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result == nil || result.Handle == nil || result.Run == nil || result.Run.Status != model.RunStatusRunning {
		t.Fatalf("Create() = %#v, want a started run", result)
	}
	if executor.startCalls != 1 {
		t.Fatalf("executor start calls = %d, want 1", executor.startCalls)
	}
}

func TestRunApplicationServiceCreateRejectsAttachmentOutsideSession(t *testing.T) {
	service := NewRunApplicationService(
		&applicationRunStoreStub{}, &applicationExecutorStub{}, sessionLookupStub{session: &model.Session{ID: "session-1"}},
		fileLookupStub{files: map[string]*model.File{"file-2": {ID: "file-2", SessionID: "other-session"}}},
		settingsSourceStub{}, nil, "prompt-hash",
	)

	_, err := service.Create(context.Background(), CreateApplicationRunInput{
		SessionID: "session-1", IdempotencyKey: "request-1", Content: "分析", AttachmentIDs: []string{"file-2"},
	})
	appErr, ok := err.(*apperr.Error)
	if !ok || appErr.Kind != apperr.KindNotFound {
		t.Fatalf("Create() error = %T %v, want attachment not found", err, err)
	}
}

func TestRunApplicationServiceSubmitInputRestartsWithOwnedAttachmentContext(t *testing.T) {
	store := &applicationRunStoreStub{
		run: &model.Run{ID: "run-1", SessionID: "session-1", Status: model.RunStatusWaitingInput},
		resume: &RunResume{Run: &model.Run{ID: "run-1", SessionID: "session-1", Status: model.RunStatusRunning}, StartExecution: true,
			Messages: []llmcore.Message{{Role: model.RoleUser, ContentText: "补充信息"}}},
	}
	executor := &applicationExecutorStub{}
	loader := &attachmentLoaderStub{result: []attachment.FileContext{{Filename: "extra.txt", Content: "补充正文"}}}
	service := NewRunApplicationService(
		store, executor, sessionLookupStub{},
		fileLookupStub{files: map[string]*model.File{"file-1": {ID: "file-1", SessionID: "session-1", Filename: "extra.txt"}}},
		settingsSourceStub{}, loader, "prompt-hash",
	)

	result, err := service.SubmitInput(context.Background(), "run-1", SubmitInputRequest{
		IdempotencyKey: "reply-1", ReplyToMessageID: "question-1", Content: "补充信息", Attachments: []string{"file-1"},
	})
	if err != nil {
		t.Fatalf("SubmitInput() error = %v", err)
	}
	if result.Handle == nil || executor.startedRun != "run-1" || len(executor.contexts) != 1 || executor.contexts[0].Content != "补充正文" {
		t.Fatalf("SubmitInput() did not restart with attachment context: %+v", executor)
	}
}

func TestRunApplicationServiceListHistoryBatchesMessages(t *testing.T) {
	first := &model.Run{ID: "run-1", SessionID: "session-1"}
	second := &model.Run{ID: "run-2", SessionID: "session-1"}
	store := &applicationRunStoreStub{
		history:      []*model.Run{first, nil, second},
		historyTotal: 2,
		messages: map[string][]*model.RunMessage{
			"run-1": {{RunID: "run-1", Content: "first"}},
			"run-2": {{RunID: "run-2", Content: "second"}},
		},
	}
	svc := NewRunApplicationService(store, &applicationExecutorStub{}, nil, nil, nil, nil, "prompt-hash")

	page, err := svc.ListBySessionID(context.Background(), "session-1", 20, 5)
	if err != nil {
		t.Fatalf("ListBySessionID() error = %v", err)
	}
	if store.batchCalls != 1 || store.singleCalls != 0 {
		t.Fatalf("message reads = batch:%d single:%d, want batch:1 single:0", store.batchCalls, store.singleCalls)
	}
	if len(page.Items) != 2 || page.Items[0].Messages[0].Content != "first" || page.Items[1].Messages[0].Content != "second" {
		t.Fatalf("history page = %+v, want messages grouped by run", page)
	}
	if page.Total != 2 || page.Limit != 20 || page.Offset != 5 {
		t.Fatalf("pagination = %+v, want total=2 limit=20 offset=5", page)
	}
}
