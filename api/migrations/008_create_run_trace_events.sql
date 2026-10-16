-- run_trace_events 仅保存 UI 可回放的关键执行过程；逐 token 文本仍由 Redis Stream 短期承载。
-- 可重复执行，供 Docker 首次初始化与独立测试环境按文件顺序建表。

CREATE TABLE IF NOT EXISTS run_trace_events (
    run_id       VARCHAR(255) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    sequence     BIGINT NOT NULL CHECK (sequence > 0),
    event_type   VARCHAR(32) NOT NULL CHECK (event_type IN (
        'plan', 'step', 'tool_calling', 'tool_called', 'wait', 'error'
    )),
    payload      JSONB NOT NULL,
    created_at   TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (run_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_run_trace_events_run_sequence
    ON run_trace_events (run_id, sequence);

-- 同一秒内的消息、计划与工具事件也必须保留稳定顺序；新部署直接创建微秒列，
-- 已存在的开发库在执行此初始化脚本时原地提升精度，无需额外迁移服务。
ALTER TABLE messages
    ALTER COLUMN created_at TYPE TIMESTAMP(6),
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP(6);

ALTER TABLE run_trace_events
    ALTER COLUMN created_at TYPE TIMESTAMP(6),
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP(6);

COMMENT ON TABLE run_trace_events IS 'Run 的有限 UI 回放追踪记录，不保存 token 增量或二进制产物';
