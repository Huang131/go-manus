package external

import "encoding/json"

// JSONParser JSON 解析器接口
// 用于解析 LLM 返回的 JSON 字符串，并修复可能存在的格式错误
type JSONParser interface {
	// Parse 解析 JSON 字符串
	// text: JSON 字符串
	// v: 目标对象指针
	// 返回解析后的对象和错误
	Parse(text string, v interface{}) error
}

// DefaultJSONParser 默认 JSON 解析器（标准库）
type DefaultJSONParser struct{}

// NewDefaultJSONParser 创建默认 JSON 解析器
func NewDefaultJSONParser() *DefaultJSONParser {
	return &DefaultJSONParser{}
}

// Parse 解析 JSON 字符串
func (p *DefaultJSONParser) Parse(text string, v interface{}) error {
	return json.Unmarshal([]byte(text), v)
}
