package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

type stopSessionRepository struct {
	repository.SessionRepository
	statusErr error
}

type successfulStopSessionRepository struct {
	repository.SessionRepository
}

func (successfulStopSessionRepository) UpdateStatus(context.Context, string, model.SessionStatus) error {
	return nil
}

func (r *stopSessionRepository) UpdateStatus(context.Context, string, model.SessionStatus) error {
	return r.statusErr
}

func TestAgentService_StopSessionReturnsStatusUpdateError(t *testing.T) {
	wantErr := errors.New("status update failed")
	svc := &AgentService{
		sessionRep:    &stopSessionRepository{statusErr: wantErr},
		taskBySession: make(map[string]*RedisStreamTask),
	}

	err := svc.StopSession(context.Background(), "session-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("StopSession() error = %v, want %v", err, wantErr)
	}
}

func TestAgentService_StopSessionKeepsTaskMappingUntilRunnerExits(t *testing.T) {
	runner := &blockingTaskRunner{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	task := NewRedisStreamTask(&mockMQWrapper{}, runner)
	svc := &AgentService{
		sessionRep:    successfulStopSessionRepository{},
		taskBySession: map[string]*RedisStreamTask{"session-1": task},
	}
	task.SetOnFinished(func() {
		svc.mu.Lock()
		delete(svc.taskBySession, "session-1")
		svc.mu.Unlock()
	})

	if err := task.Invoke(context.Background()); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	<-runner.started

	if err := svc.StopSession(context.Background(), "session-1"); err != nil {
		t.Fatalf("StopSession() error = %v", err)
	}
	svc.mu.RLock()
	_, mapped := svc.taskBySession["session-1"]
	svc.mu.RUnlock()
	if !mapped {
		t.Fatal("StopSession removed task mapping before runner exited")
	}

	close(runner.release)
	select {
	case <-task.DoneChan():
	case <-time.After(time.Second):
		t.Fatal("task did not finish after runner release")
	}
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		svc.mu.RLock()
		_, mapped = svc.taskBySession["session-1"]
		svc.mu.RUnlock()
		if !mapped {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("task mapping remained after runner exit")
}
