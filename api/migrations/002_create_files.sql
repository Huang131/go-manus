-- ============================================================
-- 创建 files 表
-- Revision: 0e0d242438bc
-- Date: 2025-05-21
-- ============================================================

-- 创建 files 表
CREATE TABLE IF NOT EXISTS files (
    id VARCHAR(255) PRIMARY KEY,
    filename VARCHAR(255) NOT NULL DEFAULT '',
    filepath VARCHAR(255) NOT NULL DEFAULT '',
    key VARCHAR(255) NOT NULL DEFAULT '',
    extension VARCHAR(255) NOT NULL DEFAULT '',
    mime_type VARCHAR(255) NOT NULL DEFAULT '',
    size BIGINT NOT NULL DEFAULT 0,
    session_id VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0)
);

-- 创建索引: session_id (关联查询)
CREATE INDEX IF NOT EXISTS idx_files_session_id ON files(session_id);

-- 创建索引: created_at (时间排序)
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);

-- 创建索引: extension (文件类型查询)
CREATE INDEX IF NOT EXISTS idx_files_extension ON files(extension);

-- ============================================================
-- 回滚脚本
-- ============================================================
-- DROP TABLE IF EXISTS files;