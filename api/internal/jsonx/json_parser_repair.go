package jsonx

import (
	"fmt"
	"regexp"
	"strings"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
	"github.com/bytedance/sonic"
)

// Repair 描述一次修复操作
type Repair struct {
	Stage string // 发生阶段：preprocess/repair/extract
	Type  string // 修复类型：remove_markdown/complete_brackets/json_repair/extract_json
}

// ParseRepairs 包含所有修复步骤的详情
type ParseRepairs struct {
	Repairs     []Repair // 按执行顺序排列的修复步骤
	WasRepaired bool     // 是否进行了任何修复
}

// RepairJSONParser 带修复功能的 JSON 解析器
// 使用专业的 json-repair 库处理 LLM 输出中常见的 JSON 格式错误
type RepairJSONParser struct{}

// NewRepairJSONParser 创建带修复功能的 JSON 解析器
func NewRepairJSONParser() *RepairJSONParser {
	return &RepairJSONParser{}
}

// Parse 解析 JSON 字符串，尝试修复常见的格式错误
func (p *RepairJSONParser) Parse(text string, v interface{}) error {
	_, err := p.ParseWithRepairs(text, v)
	return err
}

// ParseWithRepairs 解析 JSON 字符串并返回修复步骤详情
// 用于埋点和调试，观测 LLM 输出的 JSON 修复情况
func (p *RepairJSONParser) ParseWithRepairs(text string, v interface{}) (*ParseRepairs, error) {
	result := &ParseRepairs{}

	if text == "" {
		return result, fmt.Errorf("json text is empty")
	}

	// 预处理：移除 markdown 代码块标记
	cleaned := removeMarkdownCodeBlocks(text)
	if cleaned != text {
		if err := sonic.UnmarshalString(cleaned, v); err == nil {
			result.Repairs = append(result.Repairs, Repair{Stage: "preprocess", Type: "remove_markdown"})
			result.WasRepaired = true
			return result, nil
		}
		// markdown 移除后解析失败，基于 cleaned 继续后续修复
		text = cleaned
	}

	// 尝试直接解析
	if err := sonic.UnmarshalString(text, v); err == nil {
		return result, nil
	}

	// 兜底：补全截断 JSON 的结尾括号（在 json-repair 前优先尝试）
	// 仅在 brackets 真正补全且解析成功时才记录 repair
	if completed := completeTruncatedJSON(text); completed != text {
		if err := sonic.UnmarshalString(completed, v); err == nil {
			result.Repairs = append(result.Repairs, Repair{Stage: "preprocess", Type: "complete_brackets"})
			result.WasRepaired = true
			return result, nil
		}
		// bracket 补全未成功，用 completed 文本继续（可能需要 json-repair 进一步修复）
		text = completed
	}

	// 使用 json-repair 修复（基于当前 text，可能经过了 bracket 补全）
	fixed, err := jsonrepair.RepairJSON(text)
	if err == nil && fixed != text {
		if err := sonic.UnmarshalString(fixed, v); err == nil {
			result.Repairs = append(result.Repairs, Repair{Stage: "repair", Type: "json_repair"})
			result.WasRepaired = true
			return result, nil
		}
	}

	// 尝试提取 JSON 部分（回到 cleaned 避免 bracket 补全的干扰）
	extracted := extractJSON(cleaned)
	if extracted != cleaned {
		if err := sonic.UnmarshalString(extracted, v); err == nil {
			result.Repairs = append(result.Repairs, Repair{Stage: "extract", Type: "extract_json"})
			result.WasRepaired = true
			return result, nil
		}
		// 提取后仍需 json-repair 修复
		if fixedExtracted, err := jsonrepair.RepairJSON(extracted); err == nil && fixedExtracted != extracted {
			if err := sonic.UnmarshalString(fixedExtracted, v); err == nil {
				result.Repairs = append(result.Repairs, Repair{Stage: "extract", Type: "extract_json"})
				result.WasRepaired = true
				return result, nil
			}
		}
	}

	// 尝试对 cleaned 直接 json-repair（edge case：既不是截断也不是嵌入前后文）
	if fixedCleaned, err := jsonrepair.RepairJSON(cleaned); err == nil && fixedCleaned != cleaned {
		if err := sonic.UnmarshalString(fixedCleaned, v); err == nil {
			result.Repairs = append(result.Repairs, Repair{Stage: "repair", Type: "json_repair"})
			result.WasRepaired = true
			return result, nil
		}
	}

	return result, fmt.Errorf("failed to parse JSON after repair attempts")
}

// isTruncatedJSON 检测 JSON 是否被截断（括号不平衡）
// 只有在括号明显不平衡时才返回 true，避免误判正常 JSON
func isTruncatedJSON(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) == 0 {
		return false
	}

	// 必须以 { 或 [ 开始
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return false
	}

	depth := 0
	inString := false
	escape := false

	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' && inString {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		}
	}

	return depth > 0
}

// completeTruncatedJSON 补全截断 JSON 的结尾括号
func completeTruncatedJSON(text string) string {
	if !isTruncatedJSON(text) {
		return text
	}

	trimmed := strings.TrimSpace(text)

	openCurly, closeCurly, openBracket, closeBracket := 0, 0, 0, 0
	inString := false
	escape := false

	for i := 0; i < len(trimmed); i++ {
		c := trimmed[i]

		if escape {
			escape = false
			continue
		}

		if c == '\\' && inString {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch c {
		case '{':
			openCurly++
		case '}':
			closeCurly++
		case '[':
			openBracket++
		case ']':
			closeBracket++
		}
	}

	var suffix strings.Builder
	missingCurly := openCurly - closeCurly
	missingBracket := openBracket - closeBracket

	for i := 0; i < missingBracket; i++ {
		suffix.WriteString("]")
	}
	for i := 0; i < missingCurly; i++ {
		suffix.WriteString("}")
	}

	if suffix.Len() == 0 {
		return text
	}

	return trimmed + suffix.String()
}

// markdownCodeBlockRe 匹配 ```json ... ``` 或 ``` ... ``` 代码块。
// 提升到包级 var，避免每次 Parse 重新编译。
var markdownCodeBlockRe = regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")

// removeMarkdownCodeBlocks 移除 markdown 代码块标记
func removeMarkdownCodeBlocks(text string) string {
	return markdownCodeBlockRe.ReplaceAllString(text, "$1")
}

// extractJSON 从文本中提取 JSON 对象/数组（处理 LLM 输出带前后说明文字的场景）
func extractJSON(text string) string {
	trimmed := strings.TrimSpace(text)

	startIdx := -1
	for i, r := range trimmed {
		if r == '{' || r == '[' {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return text
	}

	var endIdx int
	depth := 0
	inString := false
	escape := false

	for i := startIdx; i < len(trimmed); i++ {
		c := rune(trimmed[i])

		if escape {
			escape = false
			continue
		}

		if c == '\\' {
			escape = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '{' || c == '[' {
			depth++
		} else if c == '}' || c == ']' {
			depth--
			if depth == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx > startIdx {
		return trimmed[startIdx:endIdx]
	}

	return text
}