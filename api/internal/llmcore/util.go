package llmcore

import "encoding/json"

// jsonMarshal 包内统一入口，方便单测替换 / 引入第三方 JSON 库
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
