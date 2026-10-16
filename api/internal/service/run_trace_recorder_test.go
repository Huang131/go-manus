package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/model"
)

type runTraceStoreStub struct {
	events []*model.RunTraceEvent
	err    error
}

func (s *runTraceStoreStub) Append(_ context.Context, _ string, event *model.RunTraceEvent) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, event)
	return nil
}

type runTraceEventPublisherStub struct {
	published int
	err       error
}

func (p *runTraceEventPublisherStub) Publish(_ context.Context, _ string, _ model.BaseEvent) error {
	p.published++
	return p.err
}

func TestRunTraceEventPublisherPersistsSanitizedPayloadBeforeRedis(t *testing.T) {
	store := &runTraceStoreStub{}
	redis := &runTraceEventPublisherStub{}
	publisher := NewRunTraceEventPublisher(store, redis)
	event := &model.ToolCalledEvent{
		Name:         "tool_called",
		ToolCallID:   "call-1",
		FunctionName: "request",
		Arguments: map[string]interface{}{
			"password": "argument-password",
			"nested":   map[string]interface{}{"TOKEN": "argument-token"},
		},
		Result: &model.ToolResult{Data: map[string]interface{}{
			"authorization": "Bearer result-token",
			"items":         []interface{}{map[string]interface{}{"secret": "result-secret"}},
		}},
	}

	if err := publisher.Publish(context.Background(), "run-1", event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("persisted events = %d, want 1", len(store.events))
	}
	if redis.published != 1 {
		t.Fatalf("redis publishes = %d, want 1", redis.published)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(store.events[0].Data, &payload); err != nil {
		t.Fatalf("persisted payload is not JSON: %v", err)
	}
	arguments := payload["arguments"].(map[string]interface{})
	if arguments["password"] != "[REDACTED]" {
		t.Fatalf("password = %#v, want redacted", arguments["password"])
	}
	if arguments["nested"].(map[string]interface{})["TOKEN"] != "[REDACTED]" {
		t.Fatalf("nested token = %#v, want redacted", arguments["nested"])
	}
	result := payload["result"].(map[string]interface{})
	data := result["data"].(map[string]interface{})
	if data["authorization"] != "[REDACTED]" {
		t.Fatalf("result authorization = %#v, want redacted", data["authorization"])
	}
	items := data["items"].([]interface{})
	if items[0].(map[string]interface{})["secret"] != "[REDACTED]" {
		t.Fatalf("nested result secret = %#v, want redacted", items[0])
	}
	if event.Arguments["password"] != "argument-password" {
		t.Fatal("Publish() mutated the live event")
	}
}

func TestRunTraceEventPublisherReturnsPersistenceFailureWithoutRedisPublish(t *testing.T) {
	want := errors.New("database unavailable")
	store := &runTraceStoreStub{err: want}
	redis := &runTraceEventPublisherStub{}
	publisher := NewRunTraceEventPublisher(store, redis)

	err := publisher.Publish(context.Background(), "run-1", model.NewWaitEvent())
	if !errors.Is(err, want) {
		t.Fatalf("Publish() error = %v, want %v", err, want)
	}
	if !errors.Is(err, ErrRunTracePersistence) {
		t.Fatalf("Publish() error = %v, want ErrRunTracePersistence", err)
	}
	if redis.published != 0 {
		t.Fatalf("redis publishes = %d, want 0 after persistence failure", redis.published)
	}
}

func TestRunTraceEventPublisherIgnoresRedisFailureAfterPersisting(t *testing.T) {
	store := &runTraceStoreStub{}
	redis := &runTraceEventPublisherStub{err: errors.New("redis unavailable")}
	publisher := NewRunTraceEventPublisher(store, redis)

	if err := publisher.Publish(context.Background(), "run-1", model.NewWaitEvent()); err != nil {
		t.Fatalf("Publish() error = %v, want nil when redis fails", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("persisted events = %d, want 1", len(store.events))
	}
}

func TestRunTraceEventPublisherOnlyPersistsWhitelistedEvents(t *testing.T) {
	store := &runTraceStoreStub{}
	redis := &runTraceEventPublisherStub{}
	publisher := NewRunTraceEventPublisher(store, redis)

	for _, event := range []model.BaseEvent{
		&model.MessageDeltaEvent{MessageID: "message-1", Delta: "token"},
		&model.MessageDoneEvent{MessageID: "message-1", Content: "done"},
		&model.ShellOutputEvent{SessionID: "session-1"},
	} {
		if err := publisher.Publish(context.Background(), "run-1", event); err != nil {
			t.Fatalf("Publish(%s) error = %v", event.GetType(), err)
		}
	}
	if len(store.events) != 0 {
		t.Fatalf("persisted events = %d, want 0", len(store.events))
	}
	if redis.published != 3 {
		t.Fatalf("redis publishes = %d, want 3", redis.published)
	}
}

func TestRunTraceEventPublisherTruncatesOversizedPayloadToSafeJSON(t *testing.T) {
	store := &runTraceStoreStub{}
	publisher := NewRunTraceEventPublisher(store, nil)
	secret := strings.Repeat("very-secret-data", 8*1024)
	event := &model.ToolCallingEvent{Arguments: map[string]interface{}{"payload": secret}}

	if err := publisher.Publish(context.Background(), "run-1", event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("persisted events = %d, want 1", len(store.events))
	}
	if len(store.events[0].Data) > 64*1024 {
		t.Fatalf("payload size = %d, want <= 65536", len(store.events[0].Data))
	}
	if !json.Valid(store.events[0].Data) {
		t.Fatalf("oversized payload is not valid JSON: %q", store.events[0].Data)
	}
	if strings.Contains(string(store.events[0].Data), secret) {
		t.Fatal("oversized payload retained the original secret")
	}
}
