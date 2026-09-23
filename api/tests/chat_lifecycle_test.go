//go:build integration

package integration

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/bootstrap"
	"github.com/Huang131/go-manus/api/internal/llm"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/mq"
	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptedLLM 按固定顺序返回响应，让 HTTP 集成测试只验证内部业务链路。
type scriptedLLM struct {
	mu        sync.Mutex
	responses []string
}

// blockingLLM 让一次模型调用停在流式阶段，供 HTTP stop 取消契约测试使用。
type blockingLLM struct {
	started    chan struct{}
	returned   chan struct{}
	startOnce  sync.Once
	returnOnce sync.Once
}

func (f *blockingLLM) Invoke(ctx context.Context, _ *llm.LLMRequest) (*llmcore.LLMResponse, error) {
	return nil, ctx.Err()
}

func (f *blockingLLM) Stream(ctx context.Context, _ *llm.LLMRequest) (<-chan llmcore.LLMDelta, error) {
	f.startOnce.Do(func() { close(f.started) })
	<-ctx.Done()
	f.returnOnce.Do(func() { close(f.returned) })
	return nil, ctx.Err()
}

func (*blockingLLM) ModelName() string    { return "blocking-fake" }
func (*blockingLLM) Temperature() float64 { return 0 }
func (*blockingLLM) MaxTokens() int       { return 1024 }

func (f *scriptedLLM) Invoke(ctx context.Context, _ *llm.LLMRequest) (*llmcore.LLMResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.responses) == 0 {
		return nil, fmt.Errorf("fake llm: no scripted response")
	}
	content := f.responses[0]
	f.responses = f.responses[1:]
	return &llmcore.LLMResponse{
		Model: "deterministic-fake",
		Message: llmcore.Message{
			Role:        model.RoleAssistant,
			ContentText: content,
		},
		FinishReason: llmcore.FinishReasonStop,
	}, nil
}

// Stream 把脚本响应适配为 RoutedLLM 使用的流式接口，确保测试走生产流式路径。
func (f *scriptedLLM) Stream(ctx context.Context, req *llm.LLMRequest) (<-chan llmcore.LLMDelta, error) {
	resp, err := f.Invoke(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan llmcore.LLMDelta, 1)
	ch <- llmcore.LLMDelta{
		ContentText:  resp.Message.ContentText,
		FinishReason: resp.FinishReason,
		Usage:        &resp.Usage,
	}
	close(ch)
	return ch, nil
}

func (*scriptedLLM) ModelName() string    { return "deterministic-fake" }
func (*scriptedLLM) Temperature() float64 { return 0 }
func (*scriptedLLM) MaxTokens() int       { return 1024 }

func newChatTestApp(t *testing.T, fake llm.LLM) *bootstrap.App {
	t.Helper()
	app, err := bootstrap.BuildWithFactories(testCfg, bootstrap.Options{
		EnablePostgres:    true,
		EnableRedis:       true,
		EnableStorage:     false,
		EnableLLM:         true,
		EnableSandbox:     false,
		EnableBrowser:     false,
		EnableSearch:      false,
		EnableAgent:       true,
		EnableRoutes:      true,
		EnableHealthCheck: true,
	}, bootstrap.Factories{NewLLM: func(*llm.LLMRuntimeConfig) llm.LLM { return fake }})
	require.NoError(t, err)
	t.Cleanup(app.Close)
	return app
}

type sseEvent struct {
	ID   string
	Type string
	Data string
}

func parseSSEEvents(t *testing.T, body string) []sseEvent {
	t.Helper()
	blocks := strings.Split(strings.TrimSpace(body), "\n\n")
	events := make([]sseEvent, 0, len(blocks))
	for _, block := range blocks {
		var event sseEvent
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "id: "):
				event.ID = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				event.Type = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				event.Data = strings.TrimPrefix(line, "data: ")
			}
		}
		if event.Type != "" {
			events = append(events, event)
		}
	}
	return events
}

func eventIndex(events []sseEvent, eventType string, start int) int {
	for i := start; i < len(events); i++ {
		if events[i].Type == eventType {
			return i
		}
	}
	return -1
}

func readSSEEvent(t *testing.T, reader *bufio.Reader) sseEvent {
	t.Helper()
	var event sseEvent
	for {
		line, err := reader.ReadString('\n')
		require.NoError(t, err)
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			return event
		}
		switch {
		case strings.HasPrefix(line, "id: "):
			event.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			event.Type = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			event.Data = strings.TrimPrefix(line, "data: ")
		}
	}
}

func appendTaskEvent(t *testing.T, queue *mq.RedisStreamMessageQueue, taskID string, eventType model.EventType, payload map[string]any) string {
	t.Helper()
	data, err := sonic.Marshal(payload)
	require.NoError(t, err)
	wrapper, err := sonic.Marshal(&model.Event{
		Type:      eventType,
		CreatedAt: time.Now(),
		Data:      data,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := queue.Put(ctx, "task:output:"+taskID, string(wrapper))
	require.NoError(t, err)
	return id
}

func TestChatEndpoint_SuccessStreamsOrderedEvents(t *testing.T) {
	fake := &scriptedLLM{responses: []string{
		`{"message":"已制定计划","goal":"回答测试请求","title":"测试计划","language":"zh","steps":[{"id":"s1","description":"生成答案"}]}`,
		`{"success":true,"result":"步骤完成","attachments":[]}`,
		`{"steps":[]}`,
		`{"message":"最终答案","attachments":[]}`,
	}}
	app := newChatTestApp(t, fake)

	sessionID := createSessionWithServer(t, app.Engine)
	defer CleanupSessionWithDB(t, app.Postgres, sessionID)

	w := postJSONWithServer(t, app.Engine, "/api/sessions/"+sessionID+"/chat", map[string]any{"message": "请回答测试请求"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	events := parseSSEEvents(t, w.Body.String())
	require.GreaterOrEqual(t, len(events), 5)
	assert.Equal(t, "message", events[0].Type)
	assert.Equal(t, "task_id", events[1].Type)
	assert.Equal(t, "done", events[len(events)-1].Type)

	titleIndex := eventIndex(events, string(model.EventTypeTitle), 2)
	planIndex := eventIndex(events, string(model.EventTypePlan), 2)
	stepIndex := eventIndex(events, string(model.EventTypeStep), 2)
	require.GreaterOrEqual(t, titleIndex, 0)
	require.GreaterOrEqual(t, planIndex, 0)
	require.GreaterOrEqual(t, stepIndex, 0)
	assert.Less(t, titleIndex, planIndex)
	assert.Less(t, planIndex, stepIndex)

	var previousStreamID string
	for _, event := range events[2:] {
		require.Regexp(t, `^[0-9]+-[0-9]+$`, event.ID, "task event %q should carry Redis cursor", event.Type)
		if previousStreamID != "" {
			previousMillis, previousSequence := parseStreamID(t, previousStreamID)
			currentMillis, currentSequence := parseStreamID(t, event.ID)
			require.True(t, currentMillis > previousMillis || (currentMillis == previousMillis && currentSequence > previousSequence),
				"SSE task cursors must increase: %s then %s", previousStreamID, event.ID)
		}
		previousStreamID = event.ID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := app.SessionService.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionStatusCompleted, session.Status)
}

// TestChatEndpoint_LastEventIDResumesWithoutDuplicatesOrGaps 守护 HTTP SSE 断线续读契约：
// Last-Event-ID 是 exclusive Redis cursor，重连后只返回严格位于该游标之后的事件。
func TestChatEndpoint_LastEventIDResumesWithoutDuplicatesOrGaps(t *testing.T) {
	fake := &blockingLLM{started: make(chan struct{}), returned: make(chan struct{})}
	app := newChatTestApp(t, fake)

	sessionID := createSessionWithServer(t, app.Engine)
	defer CleanupSessionWithDB(t, app.Postgres, sessionID)

	taskID, err := app.AgentService.Chat(context.Background(), sessionID, &llmcore.Message{
		Role:        model.RoleUser,
		ContentText: "验证 SSE 断线续读",
	})
	require.NoError(t, err)
	select {
	case <-fake.started:
	case <-time.After(5 * time.Second):
		t.Fatal("fake LLM did not start")
	}

	queue := mq.NewRedisStreamMessageQueue(app.Redis.Client)
	beforeDisconnectID := appendTaskEvent(t, queue, taskID, model.EventTypeTitle, map[string]any{"title": "before disconnect"})

	server := httptest.NewServer(app.Engine)
	defer server.Close()
	chatURL := server.URL + "/api/sessions/" + sessionID + "/chat"

	firstReq, err := http.NewRequest(http.MethodPost, chatURL, strings.NewReader(`{}`))
	require.NoError(t, err)
	firstReq.Header.Set("Content-Type", "application/json")
	firstResp, err := server.Client().Do(firstReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, firstResp.StatusCode)
	firstEvent := readSSEEvent(t, bufio.NewReader(firstResp.Body))
	require.Equal(t, beforeDisconnectID, firstEvent.ID)
	require.Equal(t, string(model.EventTypeTitle), firstEvent.Type)
	require.NoError(t, firstResp.Body.Close())

	afterDisconnectID := appendTaskEvent(t, queue, taskID, model.EventTypePlan, map[string]any{"plan": "after disconnect"})
	doneID := appendTaskEvent(t, queue, taskID, model.EventTypeDone, map[string]any{"success": true})

	resumeReq, err := http.NewRequest(http.MethodPost, chatURL, strings.NewReader(`{}`))
	require.NoError(t, err)
	resumeReq.Header.Set("Content-Type", "application/json")
	resumeReq.Header.Set("Last-Event-ID", beforeDisconnectID)
	resumeResp, err := server.Client().Do(resumeReq)
	require.NoError(t, err)
	defer resumeResp.Body.Close()
	require.Equal(t, http.StatusOK, resumeResp.StatusCode)
	resumedBody, err := io.ReadAll(resumeResp.Body)
	require.NoError(t, err)

	resumed := parseSSEEvents(t, string(resumedBody))
	require.Len(t, resumed, 2, "resume must return every event after the cursor exactly once")
	assert.Equal(t, []string{afterDisconnectID, doneID}, []string{resumed[0].ID, resumed[1].ID})
	assert.Equal(t, []string{string(model.EventTypePlan), string(model.EventTypeDone)}, []string{resumed[0].Type, resumed[1].Type})
	for _, event := range resumed {
		assert.NotEqual(t, beforeDisconnectID, event.ID, "Last-Event-ID must be exclusive")
	}

	// 任务仍由 blocking fake 持有，显式停止并等待 runner 退出，避免 t.Cleanup 关闭 Redis 后
	// 后台收尾再尝试设置 stream retention，产生与测试契约无关的资源关闭告警。
	require.NoError(t, app.AgentService.StopSession(context.Background(), sessionID))
	select {
	case <-fake.returned:
	case <-time.After(5 * time.Second):
		t.Fatal("fake LLM did not stop after SSE resume test")
	}
}

func parseStreamID(t *testing.T, id string) (int64, int64) {
	t.Helper()
	parts := strings.SplitN(id, "-", 2)
	require.Len(t, parts, 2)
	millis, err := strconv.ParseInt(parts[0], 10, 64)
	require.NoError(t, err)
	sequence, err := strconv.ParseInt(parts[1], 10, 64)
	require.NoError(t, err)
	return millis, sequence
}

func TestStopEndpoint_CancelsActiveAgentAndPreservesCancelledStatus(t *testing.T) {
	fake := &blockingLLM{started: make(chan struct{}), returned: make(chan struct{})}
	app := newChatTestApp(t, fake)

	sessionID := createSessionWithServer(t, app.Engine)
	defer CleanupSessionWithDB(t, app.Postgres, sessionID)

	_, err := app.AgentService.Chat(context.Background(), sessionID, &llmcore.Message{
		Role:        model.RoleUser,
		ContentText: "请停止这个请求",
	})
	require.NoError(t, err)

	select {
	case <-fake.started:
	case <-time.After(5 * time.Second):
		t.Fatal("fake LLM did not start")
	}

	stopResponse := postJSONWithServer(t, app.Engine, "/api/sessions/"+sessionID+"/stop", nil)
	assertOK(t, stopResponse)

	select {
	case <-fake.returned:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not cancel the active LLM call")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := app.SessionService.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionStatusCancelled, session.Status)

	// 等待后台收尾后再次读取，验证 runner 的终态投影不会覆盖 cancelled。
	time.Sleep(100 * time.Millisecond)
	session, err = app.SessionService.GetSession(ctx, sessionID)
	require.NoError(t, err)
	assert.Equal(t, model.SessionStatusCancelled, session.Status)
}
