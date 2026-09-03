package external

import (
	"fmt"
	"github.com/bytedance/sonic"
	"regexp"
	"strings"

	jsonrepair "github.com/RealAlexandreAI/json-repair"
)

// RepairJSONParser 带修复功能的 JSON 解析器
// 使用专业的 json-repair 库处理 LLM 输出中常见的 JSON 格式错误
type RepairJSONParser struct{}

// NewRepairJSONParser 创建带修复功能的 JSON 解析器
func NewRepairJSONParser() *RepairJSONParser {
	return &RepairJSONParser{}
}

// Parse 解析 JSON 字符串，尝试修复常见的格式错误
func (p *RepairJSONParser) Parse(text string, v interface{}) error {
	if text == "" {
		return fmt.Errorf("json text is empty")
	}

	// 1. 预处理：移除 markdown 代码块标记
	cleaned := removeMarkdownCodeBlocks(text)

	// 2. 尝试直接解析
	if err := sonic.Unmarshal([]byte(cleaned), v); err == nil {
		return nil
	}

	// 3. 使用专业的 json-repair 库修复 JSON
	fixed, err := jsonrepair.RepairJSON(cleaned)
	if err != nil {
		// 修复失败，尝试提取 JSON 部分
		extracted := extractJSON(cleaned)
		if extracted == cleaned {
			return fmt.Errorf("failed to parse JSON after repair attempts")
		}
		fixed, _ = jsonrepair.RepairJSON(extracted)
	}

	// 4. 再次尝试解析
	if err := sonic.Unmarshal([]byte(fixed), v); err == nil {
		return nil
	}

	// 5. 尝试提取 JSON 部分并修复
	extracted := extractJSON(cleaned)
	if extracted != cleaned {
		fixedExtracted, _ := jsonrepair.RepairJSON(extracted)
		if err := sonic.Unmarshal([]byte(fixedExtracted), v); err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to parse JSON after repair attempts")
}

// markdownCodeBlockRe 匹配 ```json ... ``` 或 ``` ... ``` 代码块。
// 提升到包级 var，避免每次 Parse 重新编译。
var markdownCodeBlockRe = regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)```")

// removeMarkdownCodeBlocks 移除 markdown 代码块标记
func removeMarkdownCodeBlocks(text string) string {
	return markdownCodeBlockRe.ReplaceAllString(text, "$1")
}

// trailingCommaRe 匹配尾随逗号："," 在 }, ] 之前。
var trailingCommaRe = regexp.MustCompile(`,\s*([\]}])`)

// removeTrailingCommas 移除尾部逗号
func removeTrailingCommas(text string) string {
	return trailingCommaRe.ReplaceAllString(text, "$1")
}

// extractJSON 从文本中提取 JSON 对象/数组（处理 LLM 输出带前后说明文字的场景）
func extractJSON(text string) string {
	trimmed := strings.TrimSpace(text)

	// 查找第一个 { 或 [
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

	// 从开始位置查找匹配的结束符号（处理嵌套 + 字符串转义）
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
