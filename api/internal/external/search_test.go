package external

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

	res, err := c.Invoke(context.Background(), "hello world", &dr)
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

	res, err := c.Invoke(context.Background(), "golang", &dr)
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

	res, err := c.Invoke(context.Background(), "x", nil)
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

// 确保两家客户端都实现了 SearchEngine 接口。
var _ SearchEngine = (*TavilySearchClient)(nil)
var _ SearchEngine = (*BochaSearchClient)(nil)
