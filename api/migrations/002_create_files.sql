-- ============================================================
-- 创建 files 表
-- Date: 2026-09-02
-- ============================================================
-- 业务说明：存储用户上传或 AI 生成的文件元数据，实际文件存储在 MinIO（对象存储）

CREATE TABLE IF NOT EXISTS files (
    id          VARCHAR(255) PRIMARY KEY,
    filename    VARCHAR(255) NOT NULL DEFAULT '',
    filepath    VARCHAR(255) NOT NULL DEFAULT '',
    key         VARCHAR(255) NOT NULL DEFAULT '',
    extension   VARCHAR(255) NOT NULL DEFAULT '',
    mime_type   VARCHAR(255) NOT NULL DEFAULT '',
    size        BIGINT NOT NULL DEFAULT 0,
    session_id  VARCHAR(255) NOT NULL,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP(0)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_files_session_id ON files(session_id);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_extension ON files(extension);

-- ============================================================
-- 字段注释
-- ============================================================

COMMENT ON COLUMN files.id IS '文件唯一标识符（UUID），用于全局唯一标识';
COMMENT ON COLUMN files.filename IS '原始文件名，用户上传时的文件名';
COMMENT ON COLUMN files.filepath IS '服务器上的相对路径（本地开发环境使用）';
COMMENT ON COLUMN files.key IS 'MinIO 对象存储的 Key，用于上传/下载文件';
COMMENT ON COLUMN files.extension IS '文件扩展名（小写），用于文件类型筛选';
COMMENT ON COLUMN files.mime_type IS '文件的 MIME 类型，如 text/plain、image/png';
COMMENT ON COLUMN files.size IS '文件大小（字节）';
COMMENT ON COLUMN files.session_id IS '关联的会话 ID，一个文件只能属于一个会话';
COMMENT ON COLUMN files.updated_at IS '最后更新时间';
COMMENT ON COLUMN files.created_at IS '创建时间（上传时间）';

-- ============================================================
-- 回滚脚本
-- ============================================================
-- DROP INDEX IF EXISTS idx_files_extension;
-- DROP INDEX IF EXISTS idx_files_created_at;
-- DROP INDEX IF EXISTS idx_files_session_id;
-- DROP TABLE IF EXISTS files;