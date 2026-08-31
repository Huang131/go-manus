-- ============================================================
-- 创建 app_configs 表
-- 存储 LLM/Agent/MCP/A2A 等配置
-- ============================================================

-- 创建 app_configs 表
CREATE TABLE IF NOT EXISTS app_configs (
    id VARCHAR(255) PRIMARY KEY,
    config_type VARCHAR(255) NOT NULL,
    config_key VARCHAR(255) NOT NULL,
    config_value JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    -- 唯一约束: config_type + config_key
    CONSTRAINT uq_app_configs_type_key UNIQUE (config_type, config_key)
);

-- 创建索引: config_type (类型查询)
CREATE INDEX IF NOT EXISTS idx_app_configs_config_type ON app_configs(config_type);

-- 创建索引: config_key (键查询)
CREATE INDEX IF NOT EXISTS idx_app_configs_config_key ON app_configs(config_key);

-- ============================================================
-- 回滚脚本
-- ============================================================
-- DROP TABLE IF EXISTS app_configs;