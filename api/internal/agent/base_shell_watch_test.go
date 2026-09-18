package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"
)

type blockingWatchSandbox struct {
	external.Sandbox
	started chan struct{}
	exited  chan struct{}
	once    sync.Once
}

func (s *blockingWatchSandbox) ReadShellOutput(ctx context.Context, _ string, _ bool) (*model.ToolResult, error) {
	s.once.Do(func() { close(s.started) })
	<-ctx.Done()
	// Make a non-waiting StopShellWatch implementation fail deterministically.
	time.Sleep(25 * time.Millisecond)
	close(s.exited)
	return nil, ctx.Err()
}

func TestBaseAgentStopShellWatchWaitsForWatcherExit(t *testing.T) {
	sandbox := &blockingWatchSandbox{
		started: make(chan struct{}),
		exited:  make(chan struct{}),
	}
	agent := NewBaseAgent("test", "session-1", DefaultAgentConfig(), nil, nil)
	agent.SetEventCh(make(chan model.BaseEvent, 1))
	agent.startShellWatch(context.Background(), sandbox, "session-1")

	select {
	case <-sandbox.started:
	case <-time.After(3 * time.Second):
		t.Fatal("shell watcher did not start")
	}

	agent.StopShellWatch()
	select {
	case <-sandbox.exited:
	default:
		t.Fatal("StopShellWatch returned before watcher exited")
	}
}
