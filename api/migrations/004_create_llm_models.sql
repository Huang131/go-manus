-- ============================================================
-- 004: 多模型配置
-- 支持多个 LLM 模型并存，agent 启动读取 is_default=true 的那一个
-- ============================================================

CREATE TABLE IF NOT EXISTS llm_models (
    id          VARCHAR(255) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    provider    VARCHAR(64)  NOT NULL,
    base_url    VARCHAR(512) NOT NULL,
    api_key     VARCHAR(1024) NOT NULL,
    model_name  VARCHAR(255) NOT NULL,
    temperature NUMERIC(4,2)  NOT NULL DEFAULT 0.7,
    max_tokens  INTEGER       NOT NULL DEFAULT 8192,
    tags        JSONB         NOT NULL DEFAULT '[]'::jsonb,
    is_default  BOOLEAN       NOT NULL DEFAULT FALSE,
    is_enabled  BOOLEAN       NOT NULL DEFAULT TRUE,
    sort_order  INTEGER       NOT NULL DEFAULT 0,
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    updated_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    CONSTRAINT uq_llm_models_provider_url_model UNIQUE (provider, base_url, model_name)
);

-- 同一时刻只能有 1 个 is_default=true（partial unique index）
CREATE UNIQUE INDEX IF NOT EXISTS uq_llm_models_default
    ON llm_models (is_default) WHERE is_default = TRUE;

CREATE INDEX IF NOT EXISTS idx_llm_models_enabled
    ON llm_models (is_enabled, sort_order, created_at);

-- ============================================================
-- 回滚
-- ============================================================
-- DROP INDEX IF EXISTS idx_llm_models_enabled;
-- DROP INDEX IF EXISTS uq_llm_models_default;
-- DROP TABLE IF EXISTS llm_models;
