-- 将存量 Agent 配置收敛为唯一、完整且可校验的 AgentSettings 结构。
-- 项目尚未上线，因此非法值直接替换为当前默认值，不保留静默 clamp 兼容语义。
-- 本 migration 幂等；合法字段保留，缺失、非整数或越界字段恢复默认值。
WITH normalized AS (
    SELECT
        id,
        jsonb_build_object(
            'max_iterations',
            CASE
                WHEN jsonb_typeof(config_value -> 'max_iterations') = 'number' THEN
                    CASE
                        WHEN (config_value ->> 'max_iterations')::numeric BETWEEN 1 AND 100
                             AND trunc((config_value ->> 'max_iterations')::numeric) =
                                 (config_value ->> 'max_iterations')::numeric
                        THEN ((config_value ->> 'max_iterations')::numeric)::integer
                        ELSE 10
                    END
                ELSE 10
            END,
            'max_retries',
            CASE
                WHEN jsonb_typeof(config_value -> 'max_retries') = 'number' THEN
                    CASE
                        WHEN (config_value ->> 'max_retries')::numeric BETWEEN 1 AND 5
                             AND trunc((config_value ->> 'max_retries')::numeric) =
                                 (config_value ->> 'max_retries')::numeric
                        THEN ((config_value ->> 'max_retries')::numeric)::integer
                        ELSE 3
                    END
                ELSE 3
            END,
            'max_search_results',
            CASE
                WHEN jsonb_typeof(config_value -> 'max_search_results') = 'number' THEN
                    CASE
                        WHEN (config_value ->> 'max_search_results')::numeric BETWEEN 1 AND 10
                             AND trunc((config_value ->> 'max_search_results')::numeric) =
                                 (config_value ->> 'max_search_results')::numeric
                        THEN ((config_value ->> 'max_search_results')::numeric)::integer
                        ELSE 10
                    END
                ELSE 10
            END
        ) AS config_value
    FROM app_configs
    WHERE config_type = 'agent' AND config_key = 'default'
)
UPDATE app_configs AS config
SET config_value = normalized.config_value,
    updated_at = CURRENT_TIMESTAMP(0)
FROM normalized
WHERE config.id = normalized.id
  AND config.config_value IS DISTINCT FROM normalized.config_value;
