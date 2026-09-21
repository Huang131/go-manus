package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/sandbox"
)

type blockingWatchSandbox struct {
	sandbox.Sandbox
	started        chan struct{}
	cancelObserved chan struct{}
	release        chan struct{}
	exited         chan struct{}
	once           sync.Once
}

func (s *blockingWatchSandbox) ReadShellOutput(ctx context.Context, _ string, _ bool) (*model.ToolResult, error) {
	s.once.Do(func() { close(s.started) })
	select {
	case <-ctx.Done():
		close(s.cancelObserved)
	case <-s.release:
	}
	<-s.release
	close(s.exited)
	return nil, ctx.Err()
}

func TestBaseAgentStopShellWatchWaitsForWatcherExit(t *testing.T) {
	sandbox := &blockingWatchSandbox{
		started:        make(chan struct{}),
		cancelObserved: make(chan struct{}),
		release:        make(chan struct{}),
		exited:         make(chan struct{}),
	}
	agent := NewBaseAgent("test", "session-1", DefaultAgentConfig(), nil, nil)
	agent.SetEventCh(make(chan model.BaseEvent, 1))
	agent.startShellWatch(context.Background(), sandbox, "session-1")

	select {
	case <-sandbox.started:
	case <-time.After(3 * time.Second):
		t.Fatal("shell watcher did not start")
	}

	stopReturned := make(chan struct{})
	go func() {
		agent.StopShellWatch()
		close(stopReturned)
	}()

	select {
	case <-sandbox.cancelObserved:
	case <-time.After(time.Second):
		t.Fatal("shell watcher did not observe cancellation")
	}
	select {
	case <-stopReturned:
		t.Fatal("StopShellWatch returned before watcher exited")
	default:
	}

	close(sandbox.release)
	select {
	case <-sandbox.exited:
	case <-time.After(time.Second):
		t.Fatal("shell watcher did not exit after release")
	}
	select {
	case <-stopReturned:
	case <-time.After(time.Second):
		t.Fatal("StopShellWatch did not return after watcher exit")
	}
}
