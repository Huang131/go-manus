package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
)

type streamingAgentLLM struct {
	mockLLM
}

func (m *streamingAgentLLM) Stream(context.Context, *llm.LLMRequest) (<-chan llmcore.LLMDelta, error) {
	ch := make(chan llmcore.LLMDelta, 3)
	ch <- llmcore.LLMDelta{ContentText: "你"}
	ch <- llmcore.LLMDelta{ContentText: "好"}
	ch <- llmcore.LLMDelta{FinishReason: llmcore.FinishReasonStop}
	close(ch)
	return ch, nil
}

func TestBaseAgentInvokePublishesTextDeltas(t *testing.T) {
	llm := &streamingAgentLLM{}
	agent := NewBaseAgent("react", "session-1", DefaultAgentConfig(), llm, nil)
	events := make(chan model.BaseEvent, 4)
	agent.SetEventCh(events)

	result, err := agent.Invoke(context.Background(), "system", "query")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result.Content != "你好" {
		t.Fatalf("Invoke() content = %q, want 你好", result.Content)
	}

	close(events)
	var got []model.EventType
	for event := range events {
		got = append(got, event.GetType())
	}
	if len(got) != 3 || got[0] != model.EventTypeMessageDelta ||
		got[1] != model.EventTypeMessageDelta || got[2] != model.EventTypeMessageDone {
		t.Fatalf("stream events = %v, want two deltas followed by done", got)
	}
}

func TestReActAgentSummarizeReportsWhetherDeltasWereEmitted(t *testing.T) {
	llm := &streamingAgentLLM{}
	agent := NewReActAgent("session-1", DefaultAgentConfig(), llm, nil)
	events := make(chan model.BaseEvent, 4)
	agent.SetEventCh(events)

	summary, attachments, emitted, err := agent.Summarize(context.Background())
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if summary != "你好" {
		t.Fatalf("summary = %q, want 你好", summary)
	}
	if len(attachments) != 0 {
		t.Fatalf("attachments = %v, want none", attachments)
	}
	if !emitted {
		t.Fatal("emitted = false, want true for streaming LLM")
	}

	close(events)
	var got []model.EventType
	for event := range events {
		got = append(got, event.GetType())
	}
	if len(got) != 3 || got[0] != model.EventTypeMessageDelta || got[2] != model.EventTypeMessageDone {
		t.Fatalf("stream events = %v, want delta/delta/done", got)
	}
}

type blockingStreamingAgentLLM struct {
	streamStarted chan struct{}
}

func (m *blockingStreamingAgentLLM) Invoke(context.Context, *llm.LLMRequest) (*llmcore.LLMResponse, error) {
	return nil, context.Canceled
}

func (m *blockingStreamingAgentLLM) ModelName() string    { return "blocking" }
func (m *blockingStreamingAgentLLM) Temperature() float64 { return 0 }
func (m *blockingStreamingAgentLLM) MaxTokens() int       { return 0 }

func (m *blockingStreamingAgentLLM) Stream(context.Context, *llm.LLMRequest) (<-chan llmcore.LLMDelta, error) {
	close(m.streamStarted)
	return make(chan llmcore.LLMDelta), nil
}

func TestBaseAgentInvokeStreamingStopsWhenProviderLeavesStreamOpen(t *testing.T) {
	mock := &blockingStreamingAgentLLM{streamStarted: make(chan struct{})}
	agent := NewBaseAgent("react", "session-1", DefaultAgentConfig(), mock, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, _, err := agent.invokeLLMWithEmission(ctx, &llm.LLMRequest{}, false)
		resultCh <- err
	}()

	<-mock.streamStarted
	cancel()

	select {
	case err := <-resultCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("invokeLLMWithEmission() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("invokeLLMWithEmission() did not stop after context cancellation")
	}
}
