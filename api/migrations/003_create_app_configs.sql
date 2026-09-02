-- ============================================================
-- 创建 app_configs 表
-- Date: 2026-09-02
-- ============================================================
-- 业务说明：通用配置存储表，用于存储 LLM/Agent/MCP/A2A 等各类配置的 JSON 数据
-- 注意：LLM 模型配置已迁移到 llm_models 表，此表保留用于其他配置类型

CREATE TABLE IF NOT EXISTS app_configs (
    id              VARCHAR(255) PRIMARY KEY,
    config_type     VARCHAR(255) NOT NULL,
    config_key      VARCHAR(255) NOT NULL,
    config_value    JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    CONSTRAINT uq_app_configs_type_key UNIQUE (config_type, config_key)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_app_configs_config_type ON app_configs(config_type);
CREATE INDEX IF NOT EXISTS idx_app_configs_config_key ON app_configs(config_key);

-- ============================================================
-- 字段注释
-- ============================================================

COMMENT ON COLUMN app_configs.id IS '配置项唯一标识符（UUID）';
COMMENT ON COLUMN app_configs.config_type IS '配置类型：llm=大模型, agent=智能体, mcp=MCP配置, a2a=A2A配置, system=系统配置';
COMMENT ON COLUMN app_configs.config_key IS '配置键，在同一类型下区分不同的配置项';
COMMENT ON COLUMN app_configs.config_value IS '配置值（JSON 格式），存储任意结构的配置数据';
COMMENT ON COLUMN app_configs.created_at IS '创建时间';
COMMENT ON COLUMN app_configs.updated_at IS '最后更新时间';

-- ============================================================
-- 回滚脚本
-- ============================================================
-- DROP INDEX IF EXISTS idx_app_configs_config_key;
-- DROP INDEX IF EXISTS idx_app_configs_config_type;
-- DROP TABLE IF EXISTS app_configs;