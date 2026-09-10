package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

type stopSessionRepository struct {
	repository.SessionRepository
	statusErr error
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
