package attachment

import (
	"strings"
	"testing"
)

func TestChunk(t *testing.T) {
	got := Chunk("abcdefgh", 3)
	want := []string{"abc", "def", "gh"}
	if len(got) != 3 || got[0] != want[0] || got[2] != want[2] {
		t.Errorf("got=%v", got)
	}
}

func TestRanker_TopK(t *testing.T) {
	r := NewRanker()
	chunks := []string{
		"苹果是一种水果",
		"今天天气很好",
		"我喜欢吃苹果和香蕉",
		"中国首都是北京",
	}
	got := r.Rank("苹果", chunks, 2)
	if len(got) == 0 {
		t.Fatal("empty")
	}
	// 含"苹果"的应排在前面
	if !strings.Contains(got[0], "苹果") {
		t.Errorf("top1 not contain keyword: %v", got)
	}
}

func TestRanker_NoQuery(t *testing.T) {
	r := NewRanker()
	got := r.Rank("", []string{"a", "b", "c"}, 2)
	if len(got) != 2 {
		t.Errorf("len=%d", len(got))
	}
}
