//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestRunTraceRepositoryAppendAndListByRunIDs 守护历史回放契约：
// 同一 Run 的可追溯事件按写入顺序持久化，并能被一次批量查询完整取回。
func TestRunTraceRepositoryAppendAndListByRunIDs(t *testing.T) {
	ctx := context.Background()
	runs := testRunRepository(t)
	traces := repository.NewRunTraceRepository(testDB)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "trace-"+uuid.NewString())
	_, inserted, err := runs.CreateWithInitialMessage(ctx, run, initialRunMessage(run))
	require.NoError(t, err)
	require.True(t, inserted)

	require.NoError(t, traces.Append(ctx, run.ID, &model.RunTraceEvent{
		Type: model.EventTypeToolCalling,
		Data: json.RawMessage(`{"tool_call_id":"call-1","function_name":"shell","arguments":{"command":"pwd"}}`),
	}))
	require.NoError(t, traces.Append(ctx, run.ID, &model.RunTraceEvent{
		Type: model.EventTypeToolCalled,
		Data: json.RawMessage(`{"tool_call_id":"call-1","function_name":"shell","result":{"success":true}}`),
	}))

	eventsByRunID, err := traces.ListByRunIDs(ctx, []string{"", run.ID, run.ID})
	require.NoError(t, err)
	require.Len(t, eventsByRunID, 1)
	require.Len(t, eventsByRunID[run.ID], 2)
	require.EqualValues(t, 1, eventsByRunID[run.ID][0].Sequence)
	require.Equal(t, model.EventTypeToolCalling, eventsByRunID[run.ID][0].Type)
	require.EqualValues(t, 2, eventsByRunID[run.ID][1].Sequence)
	require.Equal(t, model.EventTypeToolCalled, eventsByRunID[run.ID][1].Type)
}

// TestRunTraceRepositoryAppendAssignsDistinctSequencesConcurrently 守护并发执行时的回放顺序：
// 多个事件同时落同一 Run 也必须得到连续且不重复的序号。
func TestRunTraceRepositoryAppendAssignsDistinctSequencesConcurrently(t *testing.T) {
	ctx := context.Background()
	runs := testRunRepository(t)
	traces := repository.NewRunTraceRepository(testDB)
	sessionID := createSessionForTest(t)
	t.Cleanup(func() { CleanupSession(t, sessionID) })

	run := newRunForTest(sessionID, "trace-concurrent-"+uuid.NewString())
	_, inserted, err := runs.CreateWithInitialMessage(ctx, run, initialRunMessage(run))
	require.NoError(t, err)
	require.True(t, inserted)

	const eventCount = 8
	start := make(chan struct{})
	errs := make(chan error, eventCount)
	var workers sync.WaitGroup
	for index := 0; index < eventCount; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			<-start
			errs <- traces.Append(ctx, run.ID, &model.RunTraceEvent{
				Type: model.EventTypeStep,
				Data: json.RawMessage(`{"step":{"id":"step-` + string(rune('a'+index)) + `"}}`),
			})
		}(index)
	}
	close(start)
	workers.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	eventsByRunID, err := traces.ListByRunIDs(ctx, []string{run.ID})
	require.NoError(t, err)
	require.Len(t, eventsByRunID[run.ID], eventCount)
	for index, event := range eventsByRunID[run.ID] {
		require.EqualValues(t, index+1, event.Sequence)
	}
}
