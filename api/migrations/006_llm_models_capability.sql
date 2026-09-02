-- ============================================================
-- 006: llm_models 加能力画像 / 请求策略 / 成本策略
-- 对应 MULTI_LLM_ADAPTER_DESIGN.md 阶段 0
-- ============================================================
-- 三个 JSONB 字段：
--   capabilities   模型能力画像（tool_call/streaming/vision/...）
--   request_policy 请求侧策略（reasoning_effort 等 provider 白名单参数）
--   cost_policy    成本策略（input_price/output_price 美元 / 百万 token）
-- 全部用 JSONB 而不是一堆 bool 字段，原因是：
--   1) 不同 provider 能力集差异大，扩展不需改 schema
--   2) request_policy 是 key-value map，固定字段表达不了白名单机制
-- ============================================================

ALTER TABLE llm_models
    ADD COLUMN IF NOT EXISTS capabilities   JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS request_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS cost_policy    JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 索引：用 capabilities 中的 supports_tool_calls 过滤（未来 Orchestrator 会用）
CREATE INDEX IF NOT EXISTS idx_llm_models_capability_tool
    ON llm_models ((capabilities->>'supports_tool_calls'))
    WHERE is_enabled = TRUE;

-- 索引：用 capabilities 中的 supports_streaming 过滤
CREATE INDEX IF NOT EXISTS idx_llm_models_capability_stream
    ON llm_models ((capabilities->>'supports_streaming'))
    WHERE is_enabled = TRUE;

-- ============================================================
-- 默认能力画像：把所有现有模型的能力补全到 JSONB
-- 旧模型无 capabilities 记录，OpenAI 兼容 + Anthropic 协议默认开启 tool/stream/text
-- ============================================================
UPDATE llm_models
SET capabilities = jsonb_build_object(
    'supports_text', true,
    'supports_tool_calls', true,
    'supports_structured_output', true,
    'supports_json_mode', true,
    'supports_strict_structured_output', false,
    'supports_streaming', true,
    'supports_vision', false,
    'supports_reasoning', false,
    'max_context_tokens', 128000,
    'max_output_tokens', 8192
)
WHERE capabilities = '{}'::jsonb;

-- ============================================================
-- 回滚
-- ============================================================
-- DROP INDEX IF EXISTS idx_llm_models_capability_stream;
-- DROP INDEX IF EXISTS idx_llm_models_capability_tool;
-- ALTER TABLE llm_models DROP COLUMN IF EXISTS cost_policy;
-- ALTER TABLE llm_models DROP COLUMN IF EXISTS request_policy;
-- ALTER TABLE llm_models DROP COLUMN IF EXISTS capabilities;
