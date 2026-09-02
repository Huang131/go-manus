-- ============================================================
-- 创建 sessions 表
-- Date: 2026-09-02
-- ============================================================
-- 业务说明：存储 AI Agent 的会话上下文，一个用户会话对应一条记录

CREATE TABLE IF NOT EXISTS sessions (
    id                      VARCHAR(255) PRIMARY KEY,
    sandbox_id              VARCHAR(255),
    task_id                 VARCHAR(255),
    title                   VARCHAR(255) NOT NULL DEFAULT '',
    unread_message_count    INT NOT NULL DEFAULT 0,
    latest_message          TEXT NOT NULL DEFAULT '',
    latest_message_at       TIMESTAMP,
    events                  JSONB NOT NULL DEFAULT '[]',
    files                   JSONB NOT NULL DEFAULT '[]',
    memories                JSONB NOT NULL DEFAULT '{}',
    status                  VARCHAR(255) NOT NULL DEFAULT '',
    updated_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    created_at              TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_task_id ON sessions(task_id);

-- ============================================================
-- 字段注释
-- ============================================================

COMMENT ON COLUMN sessions.id IS '会话唯一标识符（UUID），用于唯一标识一个会话';
COMMENT ON COLUMN sessions.sandbox_id IS '关联的沙箱容器 ID，标识这个会话使用的沙箱环境';
COMMENT ON COLUMN sessions.task_id IS '关联的任务 ID，支持与外部工作流系统关联';
COMMENT ON COLUMN sessions.title IS '会话标题，用户输入或 AI 自动生成';
COMMENT ON COLUMN sessions.unread_message_count IS '未读消息数量，UI 显示小红点提醒';
COMMENT ON COLUMN sessions.latest_message IS '最新一条消息的摘要，用于会话列表展示';
COMMENT ON COLUMN sessions.latest_message_at IS '最新消息的时间戳，用于会话列表按时间排序';
COMMENT ON COLUMN sessions.events IS '会话事件历史 JSON 数组，存储完整对话记录';
COMMENT ON COLUMN sessions.files IS '会话关联的文件列表 JSON 数组';
COMMENT ON COLUMN sessions.memories IS 'AI 记忆数据 JSON 对象，存储会话中学到的关键信息';
COMMENT ON COLUMN sessions.status IS '会话状态：空字符串=初始, active=活跃, completed=完成, failed=失败, cancelled=取消';
COMMENT ON COLUMN sessions.updated_at IS '最后更新时间，任何字段修改都会更新';
COMMENT ON COLUMN sessions.created_at IS '创建时间，记录会话创建时刻';

-- ============================================================
-- 回滚脚本
-- ============================================================
-- DROP INDEX IF EXISTS idx_sessions_task_id;
-- DROP INDEX IF EXISTS idx_sessions_created_at;
-- DROP INDEX IF EXISTS idx_sessions_status;
-- DROP TABLE IF EXISTS sessions;