-- ============================================================
-- 创建 llm_models 表
-- Date: 2026-09-02
-- ============================================================
-- 业务说明：多模型配置管理，支持同时配置多个 LLM 模型，agent 启动时读取默认模型

CREATE TABLE IF NOT EXISTS llm_models (
    id              VARCHAR(255) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    provider        VARCHAR(64) NOT NULL,
    base_url        VARCHAR(512) NOT NULL,
    api_key         VARCHAR(1024) NOT NULL,
    model_name      VARCHAR(255) NOT NULL,
    temperature     NUMERIC(4,2) NOT NULL DEFAULT 0.7,
    max_tokens      INTEGER NOT NULL DEFAULT 8192,
    tags            JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    is_enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    capabilities    JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_policy  JSONB NOT NULL DEFAULT '{}'::jsonb,
    cost_policy     JSONB NOT NULL DEFAULT '{}'::jsonb,
    runtime_health  JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    CONSTRAINT uq_llm_models_provider_url_model UNIQUE (provider, base_url, model_name)
);

-- 创建索引
CREATE UNIQUE INDEX IF NOT EXISTS uq_llm_models_default ON llm_models (is_default) WHERE is_default = TRUE;
CREATE INDEX IF NOT EXISTS idx_llm_models_capability_tool ON llm_models ((capabilities->>'supports_tool_calls')) WHERE is_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_llm_models_capability_stream ON llm_models ((capabilities->>'supports_streaming')) WHERE is_enabled = TRUE;
CREATE INDEX IF NOT EXISTS idx_llm_models_enabled ON llm_models (is_enabled, sort_order, created_at);

-- ============================================================
-- 字段注释
-- ============================================================

COMMENT ON COLUMN llm_models.id IS '模型配置唯一标识符（UUID）';
COMMENT ON COLUMN llm_models.name IS '模型显示名称，UI 展示给用户看的名称';
COMMENT ON COLUMN llm_models.provider IS '模型提供商标识：openai=OpenAI, anthropic=Anthropic, google=Gemini, azure=Azure, ollama=本地Ollama, custom=自定义';
COMMENT ON COLUMN llm_models.base_url IS 'API 端点基础 URL，如 https://api.openai.com/v1';
COMMENT ON COLUMN llm_models.api_key IS 'API 密钥，用于 API 认证';
COMMENT ON COLUMN llm_models.model_name IS '模型标识符（API 调用时使用），如 gpt-4、claude-3-sonnet';
COMMENT ON COLUMN llm_models.temperature IS '采样温度（0.0-2.0），控制输出随机性，0.0最确定，2.0最随机';
COMMENT ON COLUMN llm_models.max_tokens IS '最大生成 token 数，限制单次响应的最大长度';
COMMENT ON COLUMN llm_models.tags IS '模型标签 JSON 数组，用于分组和筛选';
COMMENT ON COLUMN llm_models.is_default IS '是否为默认模型，同一时刻只能有一个默认模型';
COMMENT ON COLUMN llm_models.is_enabled IS '是否启用，禁用后不出现在选择列表中';
COMMENT ON COLUMN llm_models.sort_order IS '排序权重，数值越小排越前面';
COMMENT ON COLUMN llm_models.capabilities IS '模型能力画像 JSON：supports_tool_calls=支持工具调用, supports_vision=支持图片, supports_streaming=支持流式响应';
COMMENT ON COLUMN llm_models.request_policy IS '请求侧策略 JSON，存储不同 Provider 特有的请求参数';
COMMENT ON COLUMN llm_models.cost_policy IS '成本策略 JSON：input_price=输入价格$/百万token, output_price=输出价格$/百万token';
COMMENT ON COLUMN llm_models.runtime_health IS '运行时健康快照 JSON：status=健康状态, recent_failures=最近失败次数, average_latency_ms=平均延迟';
COMMENT ON COLUMN llm_models.created_at IS '创建时间';
COMMENT ON COLUMN llm_models.updated_at IS '最后更新时间';

-- ============================================================
-- 初始数据（幂等：仅当表为空时插入）
-- ============================================================

DO $$
DECLARE
    existing_cnt INTEGER;
BEGIN
    SELECT COUNT(*) INTO existing_cnt FROM llm_models;
    IF existing_cnt > 0 THEN
        RETURN;
    END IF;

    INSERT INTO llm_models (
        id, name, provider, base_url, api_key, model_name,
        temperature, max_tokens, tags, is_default, is_enabled, sort_order,
        capabilities, request_policy, cost_policy, runtime_health
    ) VALUES (
        gen_random_uuid()::TEXT,
        'GPT-4',
        'openai',
        'https://api.openai.com/v1',
        '',
        'gpt-4',
        0.7,
        8192,
        '["gpt-4", "openai", "默认"]'::jsonb,
        TRUE,
        TRUE,
        0,
        '{"supports_text": true, "supports_tool_calls": true, "supports_structured_output": true,
          "supports_json_mode": true, "supports_strict_structured_output": false,
          "supports_streaming": true, "supports_vision": false, "supports_reasoning": false,
          "max_context_tokens": 128000, "max_output_tokens": 8192}'::jsonb,
        '{}'::jsonb,
        '{"input_price": 30, "output_price": 60}'::jsonb,
        '{"status": "healthy", "recent_failures": 0, "average_latency_ms": 0}'::jsonb
    )
    ON CONFLICT (provider, base_url, model_name) DO NOTHING;
END $$;

-- ============================================================
-- 回滚脚本
-- ============================================================
-- DROP INDEX IF EXISTS idx_llm_models_enabled;
-- DROP INDEX IF EXISTS idx_llm_models_capability_stream;
-- DROP INDEX IF EXISTS idx_llm_models_capability_tool;
-- DROP INDEX IF EXISTS uq_llm_models_default;
-- DROP TABLE IF EXISTS llm_models;