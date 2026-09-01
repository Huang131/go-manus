package attachment

import (
	"strings"
	"unicode"
)

// Chunk 按固定字节粒度切分文本
func Chunk(text string, size int) []string {
	if size <= 0 || text == "" {
		return nil
	}
	runes := []rune(text)
	var chunks []string
	for i := 0; i < len(runes); i += size {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

// Ranker 基于词频的简易段落排序器
type Ranker struct {
	stopwords map[string]struct{}
}

// NewRanker 创建排序器（带一组常见中文/英文停用词）
func NewRanker() *Ranker {
	stop := map[string]struct{}{
		"的": {}, "了": {}, "是": {}, "在": {}, "和": {}, "与": {}, "或": {},
		"a": {}, "the": {}, "of": {}, "to": {}, "and": {}, "or": {},
	}
	return &Ranker{stopwords: stop}
}

// Rank 根据 query 从 chunks 中选出 top-k 最相关的段落
func (r *Ranker) Rank(query string, chunks []string, topK int) []string {
	if topK <= 0 || len(chunks) == 0 {
		return nil
	}
	qTokens := tokenize(query, r.stopwords)
	if len(qTokens) == 0 {
		// 无有效 query 词时取前 k 个
		if len(chunks) > topK {
			return chunks[:topK]
		}
		return chunks
	}
	type scored struct {
		idx   int
		score int
	}
	results := make([]scored, 0, len(chunks))
	for i, c := range chunks {
		results = append(results, scored{idx: i, score: score(c, qTokens, r.stopwords)})
	}
	// 简单选择排序（chunks 数量通常不大）
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
	if topK > len(results) {
		topK = len(results)
	}
	out := make([]string, 0, topK)
	for i := 0; i < topK; i++ {
		if results[i].score == 0 {
			break
		}
		out = append(out, chunks[results[i].idx])
	}
	return out
}

func score(text string, qTokens map[string]int, stop map[string]struct{}) int {
	tokens := tokenize(text, stop)
	s := 0
	for tok, cnt := range tokens {
		if q, ok := qTokens[tok]; ok {
			s += q * cnt
		}
	}
	return s
}

// tokenize 拆词：连续中文字符按 2 字符切，其他按非字母数字切
func tokenize(text string, stop map[string]struct{}) map[string]int {
	out := map[string]int{}
	runes := []rune(strings.ToLower(text))
	buf := make([]rune, 0, 8)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		w := string(buf)
		if _, ok := stop[w]; !ok && len(w) > 0 {
			out[w]++
		}
		buf = buf[:0]
	}
	for _, r := range runes {
		switch {
		case unicode.Is(unicode.Han, r):
			// 中文：按 2 字符切（防止过细爆词表）
			buf = append(buf, r)
			if len(buf) >= 2 {
				flush()
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			buf = append(buf, r)
		default:
			flush()
		}
	}
	flush()
	return out
}
