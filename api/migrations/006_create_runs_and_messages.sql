-- Run 是一次用户执行的持久化事实；messages 保存该 Run 的长期用户/助手消息。
-- 本 migration 可重复执行，便于独立测试环境按文件顺序重新应用。

CREATE TABLE IF NOT EXISTS runs (
    id                  VARCHAR(255) PRIMARY KEY,
    session_id          VARCHAR(255) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    status              VARCHAR(32) NOT NULL,
    idempotency_key     VARCHAR(255) NOT NULL,
    settings_snapshot   JSONB NOT NULL,
    prompt_hash         VARCHAR(255) NOT NULL,
    execution_snapshot  JSONB NOT NULL DEFAULT '{}'::jsonb,
    snapshot_revision   INTEGER NOT NULL DEFAULT 0 CHECK (snapshot_revision >= 0),
    waiting_message_id  VARCHAR(255),
    error_code          VARCHAR(128),
    error_message       TEXT,
    started_at          TIMESTAMP,
    finished_at         TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    updated_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    CONSTRAINT chk_runs_status CHECK (status IN (
        'pending', 'running', 'waiting_input', 'cancelling',
        'succeeded', 'failed', 'cancelled', 'interrupted'
    )),
    CONSTRAINT chk_runs_waiting_message CHECK (
        (status = 'waiting_input' AND waiting_message_id IS NOT NULL)
        OR (status <> 'waiting_input' AND waiting_message_id IS NULL)
    ),
    CONSTRAINT uq_runs_session_idempotency UNIQUE (session_id, idempotency_key)
);

CREATE TABLE IF NOT EXISTS messages (
    id                  VARCHAR(255) PRIMARY KEY,
    session_id          VARCHAR(255) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    run_id              VARCHAR(255) NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    idempotency_key     VARCHAR(255),
    reply_to_message_id VARCHAR(255) REFERENCES messages(id),
    role                VARCHAR(32) NOT NULL CHECK (role IN ('user', 'assistant')),
    content             TEXT NOT NULL,
    attachments         JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    CONSTRAINT uq_messages_run_idempotency UNIQUE (run_id, idempotency_key)
);

-- 等待问题和 Run 互相引用；使用 deferred 外键，使“状态、快照、问题消息”可在一个事务中原子落库。
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_runs_waiting_message'
    ) THEN
        ALTER TABLE runs
            ADD CONSTRAINT fk_runs_waiting_message
            FOREIGN KEY (waiting_message_id) REFERENCES messages(id)
            DEFERRABLE INITIALLY DEFERRED;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_runs_one_active_per_session
    ON runs (session_id)
    WHERE status IN ('pending', 'running', 'waiting_input', 'cancelling');

CREATE INDEX IF NOT EXISTS idx_runs_session_created_at
    ON runs (session_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_messages_run_created_at
    ON messages (run_id, created_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_messages_session_created_at
    ON messages (session_id, created_at ASC, id ASC);

COMMENT ON TABLE runs IS '一次用户执行的持久化状态、配置快照和恢复快照';
COMMENT ON TABLE messages IS 'Run 关联的长期用户与助手消息，不保存逐 token 增量';
