package sse

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
)

// errorReader 在读取指定字节后返回错误，用于测试 scanner 错误处理
type errorReader struct {
	data   []byte
	pos    int64
	errAt  int // 在第 errAt 个字节后返回错误
	errMsg string
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.pos >= int64(len(r.data)) {
		return 0, io.EOF
	}
	if r.pos >= int64(r.errAt) {
		return 0, errors.New(r.errMsg)
	}
	remaining := int64(len(r.data)) - r.pos
	toCopy := int64(len(p))
	if toCopy > remaining {
		toCopy = remaining
	}
	copy(p, r.data[r.pos:r.pos+toCopy])
	r.pos += toCopy
	return int(toCopy), nil
}

func TestParseWithContext_RaceDetector(t *testing.T) {
	// 使用 race detector 验证并发安全性
	sseData := `event: content_block_delta
data: {"type":"text_delta","text":"hello"}

event: message_delta
data: {"type":"delta","text":"world"}
`

	for i := 0; i < 100; i++ {
		wg := sync.WaitGroup{}
		for j := 0; j < 10; j++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				r := bytes.NewReader([]byte(sseData))
				var frames int64
				_ = ParseWithContext(context.Background(), r, DefaultConfig, func(ctx context.Context, frame Frame) bool {
					atomic.AddInt64(&frames, 1)
					return true
				})
				if frames != 2 {
					t.Errorf("expected 2 frames, got %d", frames)
				}
			}(j)
		}
		wg.Wait()
	}
}

func TestParseWithContext_ConcurrentDifferentData(t *testing.T) {
	// 验证不同数据源并发解析不会互相干扰
	inputs := []string{
		`data: {"msg":"a"}
`,
		`data: {"msg":"b"}
`,
		`data: {"msg":"c"}
`,
		`event: custom
data: {"msg":"d"}
`,
		`data: {"part1":"e"}
data: {"part2":"f"}
`,
	}

	var wg sync.WaitGroup
	for i, input := range inputs {
		wg.Add(1)
		go func(idx int, data string) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				r := bytes.NewReader([]byte(data))
				var frames int64
				_ = ParseWithContext(context.Background(), r, DefaultConfig, func(ctx context.Context, frame Frame) bool {
					atomic.AddInt64(&frames, 1)
					return true
				})
				if frames != 1 {
					t.Errorf("goroutine %d iteration %d: expected 1 frame, got %d", idx, j, frames)
				}
			}
		}(i, input)
	}
	wg.Wait()
}

func TestParse_BasicFunctionality(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMsgs []string
	}{
		{
			name:     "single frame",
			input:    "data: hello\n",
			wantMsgs: []string{"hello"},
		},
		{
			name:     "multiple frames with separator",
			input:    "data: one\n\ndata: two\n",
			wantMsgs: []string{"one", "two"},
		},
		{
			name:     "event type",
			input:    "event: custom\nevent: type\ndata: body\n",
			wantMsgs: []string{"body"},
		},
		{
			name:     "multiline data",
			input:    "data: line1\ndata: line2\n",
			wantMsgs: []string{"line1\nline2"},
		},
		{
			name:     "empty data skipped",
			input:    "\n\n",
			wantMsgs: nil,
		},
		{
			name:     "CRLF handling",
			input:    "data: test\r\n\r\ndata: test2\r\n",
			wantMsgs: []string{"test", "test2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			var msgs []string
			_ = Parse(r, DefaultConfig, func(frame Frame) bool {
				msgs = append(msgs, string(frame.Data))
				return true
			})
			if len(msgs) != len(tt.wantMsgs) {
				t.Errorf("got %d frames, want %d", len(msgs), len(tt.wantMsgs))
				return
			}
			for i, want := range tt.wantMsgs {
				if msgs[i] != want {
					t.Errorf("frame[%d] = %q, want %q", i, msgs[i], want)
				}
			}
		})
	}
}

func TestParse_EventTypePreserved(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantEvents []string
		wantDatas  []string
	}{
		{
			name:       "event type preserved",
			input:      "event: custom\nevent: type\ndata: body\n",
			wantEvents: []string{"type"}, // SSE 规范：event: 只是声明下一个 data 的类型，不会单独发送 frame
			wantDatas:  []string{"body"},
		},
		{
			name:       "event followed by data",
			input:      "event: content_block_delta\ndata: {\"type\":\"text\"}\n",
			wantEvents: []string{"content_block_delta"},
			wantDatas:  []string{`{"type":"text"}`},
		},
		{
			name:       "empty event line",
			input:      "event: \ndata: hello\n",
			wantEvents: []string{""},
			wantDatas:  []string{"hello"},
		},
		{
			name:       "event with whitespace",
			input:      "event:   custom  \ndata: hello\n",
			wantEvents: []string{"custom"}, // TrimSpace 处理
			wantDatas:  []string{"hello"},
		},
		{
			name:       "no event type",
			input:      "data: hello\n",
			wantEvents: []string{""},
			wantDatas:  []string{"hello"},
		},
		{
			name:       "multiple events",
			input:      "event: first\ndata: a\n\nevent: second\ndata: b\n",
			wantEvents: []string{"first", "second"},
			wantDatas:  []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			var events []string
			var datas []string
			_ = Parse(r, DefaultConfig, func(frame Frame) bool {
				events = append(events, frame.Event)
				datas = append(datas, string(frame.Data))
				return true
			})
			if len(events) != len(tt.wantEvents) {
				t.Errorf("got %d frames, want %d", len(events), len(tt.wantEvents))
				return
			}
			for i, want := range tt.wantEvents {
				if events[i] != want {
					t.Errorf("frame[%d].Event = %q, want %q", i, events[i], want)
				}
			}
			for i, want := range tt.wantDatas {
				if datas[i] != want {
					t.Errorf("frame[%d].Data = %q, want %q", i, datas[i], want)
				}
			}
		})
	}
}

func TestParse_CommentLines(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantDatas []string
	}{
		{
			name:      "comment before data",
			input:     ": this is comment\ndata: hello\n",
			wantDatas: []string{"hello"},
		},
		{
			name:      "comment between frames",
			input:     "data: a\n\n: comment\n\ndata: b\n",
			wantDatas: []string{"a", "b"},
		},
		{
			name:      "multiple comments",
			input:     ": comment1\n: comment2\ndata: hello\n",
			wantDatas: []string{"hello"},
		},
		{
			name:      "comment only",
			input:     ": only comment\n",
			wantDatas: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			var datas []string
			_ = Parse(r, DefaultConfig, func(frame Frame) bool {
				datas = append(datas, string(frame.Data))
				return true
			})
			if len(datas) != len(tt.wantDatas) {
				t.Errorf("got %d frames, want %d", len(datas), len(tt.wantDatas))
				return
			}
			for i, want := range tt.wantDatas {
				if datas[i] != want {
					t.Errorf("frame[%d] = %q, want %q", i, datas[i], want)
				}
			}
		})
	}
}

func TestParseWithContext_ScannerError(t *testing.T) {
	// 测试 scanner 错误处理
	// 注意：bufio.Scanner 内部有 buffer，错误可能延迟或不传播
	// 这里是验证 reader 出错时 parser 的行为
	tests := []struct {
		name    string
		data    string
		errAt   int
		wantErr bool
	}{
		{
			name:    "normal data no error",
			data:    "data: hello\n",
			errAt:   100, // 足够大，不会触发
			wantErr: false,
		},
		{
			name:    "early read error",
			data:    "data: hello",
			errAt:   0, // 在第一个 Read 就触发错误
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &errorReader{
				data:   []byte(tt.data),
				errAt:  tt.errAt,
				errMsg: "simulated read error",
			}
			var frames int
			err := ParseWithContext(context.Background(), reader, DefaultConfig, func(ctx context.Context, frame Frame) bool {
				frames++
				return true
			})
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseWithContext_ContextCancellation(t *testing.T) {
	// 验证 context 取消能正确中断解析
	// 生成一个足够长的 SSE 流，每行都很小但总长度大
	var longInput bytes.Buffer
	for i := 0; i < 1000; i++ {
		longInput.WriteString("data: line")
		longInput.WriteString("\n\n")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	var frames int64
	err := ParseWithContext(ctx, &longInput, DefaultConfig, func(ctx context.Context, frame Frame) bool {
		atomic.AddInt64(&frames, 1)
		return true
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// 取消后可能处理了几个 frame，但不应该处理所有 1000 个
	if frames >= 1000 {
		t.Errorf("context cancel did not stop parsing, processed %d frames", frames)
	}
}

func TestParseWithContext_HandlerReturnsFalse(t *testing.T) {
	// 验证 handler 返回 false 时停止解析
	input := "data: one\n\ndata: two\n\ndata: three\n"
	r := bytes.NewReader([]byte(input))

	var frames int64
	err := ParseWithContext(context.Background(), r, DefaultConfig, func(ctx context.Context, frame Frame) bool {
		atomic.AddInt64(&frames, 1)
		return atomic.LoadInt64(&frames) < 2 // 只处理前 2 个 frame
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if frames != 2 {
		t.Errorf("expected 2 frames before stop, got %d", frames)
	}
}

func TestParse_RetryAndIdFields(t *testing.T) {
	// 验证 retry: 和 id: 字段被忽略（这是设计决策）
	// 这些字段不应导致错误，只是被忽略
	tests := []struct {
		name      string
		input     string
		wantDatas []string
	}{
		{
			name:      "retry field ignored",
			input:     "retry: 3000\ndata: hello\n",
			wantDatas: []string{"hello"},
		},
		{
			name:      "id field ignored",
			input:     "id: event-123\ndata: hello\n",
			wantDatas: []string{"hello"},
		},
		{
			name:      "retry and id ignored",
			input:     "retry: 3000\nid: event-123\ndata: hello\n",
			wantDatas: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			var datas []string
			_ = Parse(r, DefaultConfig, func(frame Frame) bool {
				datas = append(datas, string(frame.Data))
				return true
			})
			if len(datas) != len(tt.wantDatas) {
				t.Errorf("got %d frames, want %d", len(datas), len(tt.wantDatas))
				return
			}
			for i, want := range tt.wantDatas {
				if datas[i] != want {
					t.Errorf("frame[%d] = %q, want %q", i, datas[i], want)
				}
			}
		})
	}
}

func TestParse_DataFieldWithEmptyContent(t *testing.T) {
	// 测试 data: 后面为空内容的情况
	// 注意：TrimSpace 后为空的 data 行不会发送 frame
	tests := []struct {
		name      string
		input     string
		wantDatas []string
	}{
		{
			name:      "empty data field",
			input:     "data:\n",
			wantDatas: nil, // TrimSpace 后为空，不会发送 frame
		},
		{
			name:      "empty data with separator",
			input:     "data:\n\ndata: hello\n",
			wantDatas: []string{"hello"}, // 第一个空 data 不发送，第二个发送
		},
		{
			name:      "whitespace only",
			input:     "data:   \n",
			wantDatas: nil, // TrimSpace 后为空，不会发送 frame
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			var datas []string
			_ = Parse(r, DefaultConfig, func(frame Frame) bool {
				datas = append(datas, string(frame.Data))
				return true
			})
			if len(datas) != len(tt.wantDatas) {
				t.Errorf("got %d frames, want %d", len(datas), len(tt.wantDatas))
				return
			}
			for i, want := range tt.wantDatas {
				if datas[i] != want {
					t.Errorf("frame[%d] = %q, want %q", i, datas[i], want)
				}
			}
		})
	}
}
