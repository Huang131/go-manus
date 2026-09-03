package llmcore

// PlanSchema Planner 输出的结构化 Schema
// 对应 MULTI_LLM_ADAPTER_DESIGN.md 阶段 1a 提前定义
//
// 设计原则：
//   - 所有字符串字段都有 minLength/maxLength 约束，限定 LLM 自由发挥
//   - steps.id 用 pattern 约束，强制 "step_N" 格式
//   - depends_on 用 enum-like 数组，便于后续 Planner 引擎做拓扑排序
//   - language enum 限定，业务方可按用户语言渲染
//
// 注意：此 Schema 不会直接作为 response_format 发给上游
// 而是配合 Agent 层：声明 strict schema → Adapter 优先用 response_format；不支持的 Adapter 降级为 tool call
// （V1 阶段 4 才接，本文件先冻结 schema，避免阶段 1 → 阶段 4 返工）
var PlanSchema = map[string]interface{}{
	"type": "object",
	"required": []string{
		"message", "goal", "title", "language", "steps",
	},
	"properties": map[string]interface{}{
		"message": map[string]interface{}{
			"type":        "string",
			"minLength":   1,
			"maxLength":   500,
			"description": "用户可见的简短答复，1-3 句话",
		},
		"goal": map[string]interface{}{
			"type":        "string",
			"minLength":   1,
			"maxLength":   200,
			"description": "用户目标的一句话概括",
		},
		"title": map[string]interface{}{
			"type":        "string",
			"minLength":   1,
			"maxLength":   100,
			"description": "计划标题，给用户看的",
		},
		"language": map[string]interface{}{
			"type":        "string",
			"enum":        []string{"zh-CN", "en-US", "ja-JP"},
			"description": "输出语言",
		},
		"steps": map[string]interface{}{
			"type":        "array",
			"minItems":    1,
			"maxItems":    20,
			"description": "执行步骤列表",
			"items": map[string]interface{}{
				"type":     "object",
				"required": []string{"id", "description"},
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"pattern":     "^step_[0-9]{1,3}$",
						"description": "步骤唯一 id，形如 step_1, step_2",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"minLength":   5,
						"maxLength":   200,
						"description": "步骤描述，给用户看的",
					},
					"depends_on": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string", "pattern": "^step_[0-9]{1,3}$"},
						"maxItems":    20,
						"description": "依赖的其他 step id 列表",
					},
				},
			},
		},
	},
}

// PlanSchemaJSON 把 PlanSchema 转成 []byte 方便发往 OpenAI response_format
// 内部用，调用方无需关心
func PlanSchemaJSON() []byte {
	b, _ := jsonMarshal(PlanSchema)
	return b
}

// PlanSchemaName 工具名（用作 response_format.json_schema.name）
// OpenAI 要求这个名字有规律
const PlanSchemaName = "agent_plan"

// PlanSchemaDescription 工具描述
const PlanSchemaDescription = "Structured plan for an agent task"
