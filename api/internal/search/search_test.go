package search

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
)

func TestTavilySearchClient_Invoke(t *testing.T) {
	var gotAuth, gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[{"title":"t1","url":"https://a","content":"c1","score":0.9},{"title":"t2","url":"https://b","content":"c2","score":0.8}]}`))
	}))
	defer srv.Close()

	dr := "w"
	c := NewTavilySearchClientWithTimeout("key-foo", time.Second)
	c.baseURL = srv.URL

	res, err := c.Invoke(context.Background(), "hello world", &dr, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatalf("unexpected failure: %s", res.Message)
	}
	if gotAuth != "Bearer key-foo" {
		t.Errorf("Authorization = %q, want Bearer key-foo", gotAuth)
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type = %q", gotCT)
	}
	if !strings.Contains(gotBody, `"query":"hello world"`) {
		t.Errorf("body missing query: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"time_range":"week"`) {
		t.Errorf("body missing time_range mapping: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"max_results":7`) {
		t.Errorf("body missing max_results: %s", gotBody)
	}

	data, ok := res.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Data type = %T, want map", res.Data)
	}
	results, ok := data["results"].(*model.SearchResults)
	if !ok {
		t.Fatalf("results type = %T, want *model.SearchResults", data["results"])
	}
	if results.TotalResults != 2 || len(results.Results) != 2 {
		t.Fatalf("got %d results (total=%d), want 2", len(results.Results), results.TotalResults)
	}
	if results.Results[0].Title != "t1" || results.Results[0].URL != "https://a" || results.Results[0].Snippet != "c1" {
		t.Errorf("result[0] = %+v", results.Results[0])
	}
}

func TestBochaSearchClient_Invoke(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"webPages":{"value":[{"name":"b1","url":"https://x","snippet":"s1"}]}}}`))
	}))
	defer srv.Close()

	dr := "m"
	c := NewBochaSearchClientWithTimeout("key-bar", time.Second)
	c.baseURL = srv.URL

	res, err := c.Invoke(context.Background(), "golang", &dr, 6)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatalf("unexpected failure: %s", res.Message)
	}
	if gotAuth != "Bearer key-bar" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"freshness":"oneMonth"`) {
		t.Errorf("body missing freshness mapping: %s", gotBody)
	}
	if !strings.Contains(gotBody, `"count":6`) {
		t.Errorf("body missing count: %s", gotBody)
	}

	data := res.Data.(map[string]interface{})
	results := data["results"].(*model.SearchResults)
	if len(results.Results) != 1 {
		t.Fatalf("got %d results, want 1", len(results.Results))
	}
	if results.Results[0].Title != "b1" || results.Results[0].URL != "https://x" || results.Results[0].Snippet != "s1" {
		t.Errorf("result[0] = %+v", results.Results[0])
	}
}

func TestBochaSearchClient_Invoke_NonZeroCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":10001,"message":"no permission"}`))
	}))
	defer srv.Close()

	c := NewBochaSearchClientWithTimeout("key", time.Second)
	c.baseURL = srv.URL

	res, err := c.Invoke(context.Background(), "x", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if res.Success {
		t.Fatal("expected failure for non-zero bocha code")
	}
	if !strings.Contains(res.Message, "code=10001") {
		t.Errorf("Message = %q, want code=10001", res.Message)
	}
}

func TestSearchDateRangeMapping(t *testing.T) {
	str := func(s string) *string { return &s }

	cases := []struct {
		in     *string
		tavily string
		bocha  string
	}{
		{nil, "", ""},
		{str("d"), "day", "oneDay"},
		{str("w"), "week", "oneWeek"},
		{str("m"), "month", "oneMonth"},
		{str("y"), "year", "oneYear"},
		{str("unknown"), "", ""},
	}
	for _, c := range cases {
		if got := tavilyTimeRange(c.in); got != c.tavily {
			t.Errorf("tavilyTimeRange(%v) = %q, want %q", c.in, got, c.tavily)
		}
		if got := bochaFreshness(c.in); got != c.bocha {
			t.Errorf("bochaFreshness(%v) = %q, want %q", c.in, got, c.bocha)
		}
	}
}

// 确保各客户端都实现了 SearchEngine 接口。
var _ SearchEngine = (*TavilySearchClient)(nil)
var _ SearchEngine = (*BochaSearchClient)(nil)
var _ SearchEngine = (*GoogleSearchClient)(nil)

// TestGoogleSearchClient_Invoke_SendsSearchEngineID 验证 cx（Search Engine ID）被写入请求。
// 复现并锁定历史 bug：SearchEngineID 配置后曾在工厂/客户端两层被丢弃，cx 恒为空，
// Google Custom Search 会以 400 拒绝。
func TestGoogleSearchClient_Invoke_SendsSearchEngineID(t *testing.T) {
	var gotCX, gotKey, gotQ, gotNum string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotCX = q.Get("cx")
		gotKey = q.Get("key")
		gotQ = q.Get("q")
		gotNum = q.Get("num")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"queries":{"request":[]},"searchInformation":{"totalResults":"0"},"items":[]}`))
	}))
	defer srv.Close()

	c := NewGoogleSearchClientWithTimeout("key-foo", "my-engine-id", time.Second)
	c.baseURL = srv.URL

	res, err := c.Invoke(context.Background(), "golang", nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success {
		t.Fatalf("unexpected failure: %s", res.Message)
	}
	if gotCX != "my-engine-id" {
		t.Errorf("cx = %q, want my-engine-id（SearchEngineID 未被写入请求）", gotCX)
	}
	if gotKey != "key-foo" {
		t.Errorf("key = %q, want key-foo", gotKey)
	}
	if gotQ != "golang" {
		t.Errorf("q = %q, want golang", gotQ)
	}
	if gotNum != "7" {
		t.Errorf("num = %q, want 7", gotNum)
	}
}

func TestGoogleSearchClient_Invoke_CapsLimitAtTen(t *testing.T) {
	var gotNum string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotNum = r.URL.Query().Get("num")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"queries":{"request":[]},"searchInformation":{"totalResults":"0"},"items":[]}`))
	}))
	defer srv.Close()

	c := NewGoogleSearchClientWithTimeout("key", "engine", time.Second)
	c.baseURL = srv.URL
	if _, err := c.Invoke(context.Background(), "golang", nil, 50); err != nil {
		t.Fatal(err)
	}
	if gotNum != "10" {
		t.Fatalf("num = %q, want 10", gotNum)
	}
}

// TestGoogleSearchClient_Invoke_MissingSearchEngineID 验证 cx 缺失时在本地直接失败，
// 而不是发出一个注定被 Google 拒绝的请求。
func TestGoogleSearchClient_Invoke_MissingSearchEngineID(t *testing.T) {
	c := NewGoogleSearchClientWithTimeout("key-foo", "", time.Second)
	res, err := c.Invoke(context.Background(), "golang", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if res.Success {
		t.Fatal("expected failure when Search Engine ID is missing")
	}
	if !strings.Contains(res.Message, "Search Engine ID") {
		t.Errorf("Message = %q, want mention Search Engine ID", res.Message)
	}
}
