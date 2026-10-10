-- Session 只保存会话元数据；执行状态、消息和恢复快照由 runs/messages 负责。
-- 这是未上线项目的破坏式迁移，不回填历史 sessions.events。

DROP INDEX IF EXISTS idx_sessions_task_id;
DROP INDEX IF EXISTS idx_sessions_status;

ALTER TABLE sessions
    DROP COLUMN IF EXISTS task_id,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS events;
