-- ============================================================
-- 创建 sessions 表
-- Revision: 87ed1cbb1088
-- Date: 2025-05-14
-- ============================================================

-- 创建 sessions 表
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY,
    sandbox_id VARCHAR(255),
    task_id VARCHAR(255),
    title VARCHAR(255) NOT NULL DEFAULT '',
    unread_message_count INT NOT NULL DEFAULT 0,
    latest_message TEXT NOT NULL DEFAULT '',
    latest_message_at TIMESTAMP,
    events JSONB NOT NULL DEFAULT '[]',
    files JSONB NOT NULL DEFAULT '[]',
    memories JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(255) NOT NULL DEFAULT '',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0)
);

-- 创建索引: status (状态查询)
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);

-- 创建索引: created_at (时间排序)
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at DESC);

-- 创建索引: task_id (任务查询)
CREATE INDEX IF NOT EXISTS idx_sessions_task_id ON sessions(task_id);

-- ============================================================
-- 回滚脚本
-- ============================================================
-- DROP TABLE IF EXISTS sessions;