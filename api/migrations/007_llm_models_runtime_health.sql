-- ============================================================
-- 007: llm_models 增加 runtime_health 运行时健康快照
-- 对应 MULTI_LLM_ADAPTER_DESIGN.md 第 1 项和第 2 项
-- ============================================================
-- 仅保存轻量快照，不承载完整监控历史：
--   - status
--   - recent_failures
--   - average_latency_ms
-- ============================================================

ALTER TABLE llm_models
    ADD COLUMN IF NOT EXISTS runtime_health JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE llm_models
SET runtime_health = jsonb_build_object(
    'status', 'healthy',
    'recent_failures', 0,
    'average_latency_ms', 0
)
WHERE runtime_health = '{}'::jsonb;

-- ============================================================
-- 回滚
-- ============================================================
-- ALTER TABLE llm_models DROP COLUMN IF EXISTS runtime_health;
