-- ============================================================
-- 005: 从 app_configs 迁移旧 LLM 配置到 llm_models
-- 幂等：使用 ON CONFLICT DO NOTHING
-- ============================================================

DO $$
DECLARE
    old_id       VARCHAR(255);
    old_value    JSONB;
    old_base_url TEXT;
    old_api_key  TEXT;
    old_model    TEXT;
    old_temp     NUMERIC(4,2);
    old_max_tok  INTEGER;
    existing_cnt INTEGER;
BEGIN
    SELECT id, config_value
      INTO old_id, old_value
      FROM app_configs
     WHERE config_type = 'llm' AND config_key = 'default'
     LIMIT 1;

    IF old_value IS NULL THEN
        RETURN; -- 无旧数据
    END IF;

    -- 仅在 llm_models 还没有任何记录时才迁移，避免重复
    SELECT COUNT(*) INTO existing_cnt FROM llm_models;
    IF existing_cnt > 0 THEN
        RETURN;
    END IF;

    old_base_url := old_value->>'base_url';
    old_api_key  := COALESCE(old_value->>'api_key', '');
    old_model    := old_value->>'model_name';
    old_temp     := COALESCE((old_value->>'temperature')::NUMERIC, 0.7);
    old_max_tok  := COALESCE((old_value->>'max_tokens')::INTEGER, 8192);

    IF old_base_url IS NULL OR old_model IS NULL OR old_base_url = '' OR old_model = '' THEN
        RETURN; -- 旧数据字段不全，跳过
    END IF;

    INSERT INTO llm_models (
        id, name, provider, base_url, api_key, model_name,
        temperature, max_tokens, tags, is_default, is_enabled, sort_order
    ) VALUES (
        COALESCE(old_id, gen_random_uuid()::TEXT),
        '默认模型（已迁移）',
        'custom',
        old_base_url,
        old_api_key,
        old_model,
        old_temp,
        old_max_tok,
        '[]'::jsonb,
        TRUE,
        TRUE,
        0
    )
    ON CONFLICT (provider, base_url, model_name) DO NOTHING;
END $$;

-- ============================================================
-- 回滚：仅删除迁移数据，保留 app_configs 旧行
-- ============================================================
-- DELETE FROM llm_models WHERE name = '默认模型（已迁移）';
