package attachment

import (
	"context"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/Huang131/go-manus/api/internal/model"
)

// mockStorage 用于测试的内存 FileStorage
type mockStorage struct {
	data map[string][]byte
	err  error
}

func (m *mockStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	d, ok := m.data[key]
	if !ok {
		return nil, io.EOF
	}
	return io.NopCloser(strings.NewReader(string(d))), nil
}

func TestShouldInlineAndTruncate(t *testing.T) {
	cases := []struct {
		size     int64
		inline   bool
		truncate bool
		rag      bool
	}{
		{100, true, false, false},
		{InlineMaxBytes, true, false, false},
		{InlineMaxBytes + 1, false, true, false},
		{TruncateMaxBytes, false, true, false},
		{TruncateMaxBytes + 1, false, false, true},
	}
	for _, c := range cases {
		gotInline := ShouldInline(c.size)
		gotTrunc := ShouldTruncate(c.size)
		gotRAG := ShouldRAG(c.size)
		if gotInline != c.inline || gotTrunc != c.truncate || gotRAG != c.rag {
			t.Errorf("size=%d: inline=%v want %v; trunc=%v want %v; rag=%v want %v",
				c.size, gotInline, c.inline, gotTrunc, c.truncate, gotRAG, c.rag)
		}
	}
}

func TestLoader_InlineSmall(t *testing.T) {
	content := "Hello, world! 你好世界。"
	store := &mockStorage{data: map[string][]byte{"f1": []byte(content)}}
	l := NewLoader(store)
	got := l.Load(context.Background(), []model.File{
		{ID: "f1", Filename: "a.txt", Filepath: "/tmp/a.txt", Key: "f1", Size: int64(len(content))},
	}, "你好")
	if len(got) != 1 {
		t.Fatalf("len=%d", len(got))
	}
	fc := got[0]
	if fc.Mode != ModeInline {
		t.Errorf("mode=%s want inline", fc.Mode)
	}
	if fc.Content != content {
		t.Errorf("content=%q want %q", fc.Content, content)
	}
}

func TestLoader_TruncateMiddle(t *testing.T) {
	body := strings.Repeat("中", 20*1024) // 20K
	store := &mockStorage{data: map[string][]byte{"f1": []byte(body)}}
	l := NewLoader(store)
	got := l.Load(context.Background(), []model.File{
		{ID: "f1", Filename: "big.md", Filepath: "/tmp/big.md", Key: "f1", Size: int64(len(body))},
	}, "中")
	if len(got) != 1 || got[0].Mode != ModeTruncated {
		t.Fatalf("mode=%v", got[0].Mode)
	}
	if !strings.Contains(got[0].Content, "[... 内容已截断 ...]") {
		t.Errorf("missing truncate marker")
	}
}

func TestLoader_RAGLarge(t *testing.T) {
	// 构造 200K 字节，混入关键词 "重要关键词"
	padding := strings.Repeat("这是一些普通的填充内容用于稀释检索结果。\n", 1500)
	// 在 100K 处插入关键段
	insertAt := len(padding) / 2
	head := padding[:insertAt]
	tail := padding[insertAt:]
	key := "\n\n【关键段】这里包含了重要关键词与结论。\n\n"
	body := head + key + tail
	// 确保 > 64K
	if len(body) < TruncateMaxBytes+1 {
		extra := strings.Repeat("x", TruncateMaxBytes+1-len(body))
		body = body + extra
	}
	store := &mockStorage{data: map[string][]byte{"f1": []byte(body)}}
	l := NewLoader(store)
	got := l.Load(context.Background(), []model.File{
		{ID: "f1", Filename: "huge.md", Filepath: "/tmp/huge.md", Key: "f1", Size: int64(len(body))},
	}, "重要关键词")
	if got[0].Mode != ModeRAG {
		t.Fatalf("mode=%v size=%d", got[0].Mode, len(body))
	}
	if !strings.Contains(got[0].Content, "重要关键词") {
		t.Errorf("RAG should contain keyword")
	}
}

func TestLoader_BinarySkipped(t *testing.T) {
	store := &mockStorage{data: map[string][]byte{"f1": {}}}
	l := NewLoader(store)
	got := l.Load(context.Background(), []model.File{
		{ID: "f1", Filename: "a.png", Filepath: "/a.png", Key: "f1", MimeType: "image/png", Size: 100},
	}, "")
	if got[0].Mode != ModeBinary {
		t.Errorf("mode=%v want binary", got[0].Mode)
	}
}

func TestLoader_DownloadError(t *testing.T) {
	store := &mockStorage{err: io.ErrUnexpectedEOF}
	l := NewLoader(store)
	got := l.Load(context.Background(), []model.File{
		{ID: "f1", Filename: "a.txt", Filepath: "/a.txt", Key: "f1", Size: 10},
	}, "")
	if got[0].Mode != ModeSkipped {
		t.Errorf("mode=%v", got[0].Mode)
	}
	if !strings.Contains(got[0].Notice, "下载失败") {
		t.Errorf("notice=%q", got[0].Notice)
	}
}

func TestLoader_InlineBudget(t *testing.T) {
	// 五个 7K 文件（各 ≤8K inline 阈值）；
	// 累计 5*7=35K > 32K budget，最后一个应被预算拒绝
	store := &mockStorage{data: map[string][]byte{
		"f1": []byte(strings.Repeat("a", 7000)),
		"f2": []byte(strings.Repeat("b", 7000)),
		"f3": []byte(strings.Repeat("c", 7000)),
		"f4": []byte(strings.Repeat("d", 7000)),
		"f5": []byte(strings.Repeat("e", 7000)),
	}}
	l := NewLoader(store)
	got := l.Load(context.Background(), []model.File{
		{ID: "f1", Filename: "a", Filepath: "/a", Key: "f1", Size: 7000},
		{ID: "f2", Filename: "b", Filepath: "/b", Key: "f2", Size: 7000},
		{ID: "f3", Filename: "c", Filepath: "/c", Key: "f3", Size: 7000},
		{ID: "f4", Filename: "d", Filepath: "/d", Key: "f4", Size: 7000},
		{ID: "f5", Filename: "e", Filepath: "/e", Key: "f5", Size: 7000},
	}, "")
	// 前 4 个 28K 还能进，第 5 个 7K 时 budget=4K<7K → skip
	if got[3].Mode != ModeInline {
		t.Errorf("f4 mode=%v", got[3].Mode)
	}
	if got[4].Mode != ModeSkipped || !strings.Contains(got[4].Notice, "预算") {
		t.Errorf("f5 mode=%v notice=%q", got[4].Mode, got[4].Notice)
	}
}

func TestNormalizeText(t *testing.T) {
	if !utf8.ValidString(normalizeText([]byte("中文 ok"))) {
		t.Errorf("utf8 invalid")
	}
	// 模拟 latin1 字节 0xE9（é）→ 转为 rune
	if got := normalizeText([]byte{0xE9}); got != "é" {
		t.Errorf("latin1=%q", got)
	}
}
