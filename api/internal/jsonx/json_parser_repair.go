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
	Stage string // 发生阶段：preprocess/repair
	Type  string // 修复类型：complete_brackets/json_repair
}

// ParseRepairs 包含所有修复步骤的详情
type ParseRepairs struct {
	Repairs      []Repair // 按执行顺序排列的修复步骤
	WasRepaired  bool     // 是否进行了任何修复
	StrategyUsed string   // 实际使用的修复策略：complete_brackets/json_repair
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

	// 规范化：移除 markdown 代码块
	cleaned := removeMarkdownCodeBlocks(text)
	cleaned = strings.TrimSpace(cleaned)

	// 尝试直接解析（最快路径，干净 JSON 直接返回）
	if err := sonic.UnmarshalString(cleaned, v); err == nil {
		return result, nil // 无需修复，WasRepaired=false
	}

	// 尝试修复，按"轻量优先"顺序执行
	// 原则：每次修复成功后立即记录 Strategy，避免依赖最终状态推断

	// 策略 1：补全截断 JSON 的括号
	if completed := completeTruncatedJSON(cleaned); completed != cleaned {
		if err := sonic.UnmarshalString(completed, v); err == nil {
			result.Repairs = append(result.Repairs,
				Repair{Stage: "preprocess", Type: "complete_brackets"})
			result.StrategyUsed = "complete_brackets"
			result.WasRepaired = true
			return result, nil
		}
		// bracket 补全后仍需其他策略继续修复
		cleaned = completed
	}

	// 策略 2：json-repair 修复
	if fixed, err := jsonrepair.RepairJSON(cleaned); err == nil && fixed != cleaned {
		if err := sonic.UnmarshalString(fixed, v); err == nil {
			result.Repairs = append(result.Repairs,
				Repair{Stage: "repair", Type: "json_repair"})
			result.StrategyUsed = "json_repair"
			result.WasRepaired = true
			return result, nil
		}
		cleaned = fixed // json-repair 改变了文本，继续用修复结果
	}

	// 策略 3：最终兜底——对 cleaned 再 json-repair 一次（edge case）
	// 进入这里说明 cleaned 已经经过 bracket 补全和/或 json-repair，但仍无法解析
	if fixed, err := jsonrepair.RepairJSON(cleaned); err == nil && fixed != cleaned {
		if err := sonic.UnmarshalString(fixed, v); err == nil {
			result.Repairs = append(result.Repairs,
				Repair{Stage: "repair", Type: "json_repair"})
			result.StrategyUsed = "json_repair"
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
