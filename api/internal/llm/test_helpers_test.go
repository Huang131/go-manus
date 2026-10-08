package llm

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func responseJSON(status int, payload interface{}) (*http.Response, error) {
	body, err := sonic.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}, nil
}

// contextBlockingBody 先返回固定前缀，随后阻塞到请求取消，并暴露 Close 完成信号。
type contextBlockingBody struct {
	ctx       context.Context
	prefix    *strings.Reader
	closed    chan struct{}
	closeOnce sync.Once
}

func newContextBlockingBody(ctx context.Context, prefix string) *contextBlockingBody {
	return &contextBlockingBody{
		ctx:    ctx,
		prefix: strings.NewReader(prefix),
		closed: make(chan struct{}),
	}
}

func (b *contextBlockingBody) Read(p []byte) (int, error) {
	if b.prefix.Len() > 0 {
		return b.prefix.Read(p)
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *contextBlockingBody) Close() error {
	b.closeOnce.Do(func() { close(b.closed) })
	return nil
}
