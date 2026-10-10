package service

import (
	"context"
	"testing"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/settings"
)

type applicationRunStoreStub struct {
	run          *model.Run
	createInput  CreateRunInput
	resume       *RunResume
	submittedRun string
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
	return s.run, nil
}

func (s *applicationRunStoreStub) GetActiveBySessionID(_ context.Context, sessionID string) (*model.Run, error) {
	if s.run == nil || s.run.SessionID != sessionID || !s.run.Status.IsActive() {
		return nil, nil
	}
	return s.run, nil
}

func (s *applicationRunStoreStub) RequestCancel(_ context.Context, _ string) (*model.Run, error) {
	return s.run, nil
}

type applicationExecutorStub struct {
	startedRun      string
	startedMessages []llmcore.Message
	contexts        []attachment.FileContext
	cancelledRun    string
}

func (s *applicationExecutorStub) Start(_ context.Context, runID string, messages []llmcore.Message, contexts ...[]attachment.FileContext) (*RunExecutionHandle, error) {
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
