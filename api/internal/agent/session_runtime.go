package agent

import (
	"bytes"
	"context"
	"io"
	"mime"
	"path"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/Huang131/go-manus/api/internal/sandbox"
	"github.com/Huang131/go-manus/api/internal/service"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

// SessionRuntime 封装 Agent 运行期的会话持久化与文件协作逻辑。
//
// 它把 sessionID、会话/文件仓储、沙箱、对象存储与附件加载收敛为单一协作者，
// 避免 AgentTaskRunner 直接持有并散落操作这些依赖。所有会话状态回写都经过这里。
type SessionRuntime struct {
	sessionID   string
	sessionRep  repository.SessionRepository
	fileRep     repository.FileRepository
	sandbox     sandbox.Sandbox
	fileStorage service.FileStorage
	attLoader   *attachment.Loader
}

// NewSessionRuntime 构造运行期协作者。
func NewSessionRuntime(
	sessionID string,
	sessionRep repository.SessionRepository,
	fileRep repository.FileRepository,
	sandbox sandbox.Sandbox,
	fileStorage service.FileStorage,
) *SessionRuntime {
	r := &SessionRuntime{
		sessionID:   sessionID,
		sessionRep:  sessionRep,
		fileRep:     fileRep,
		sandbox:     sandbox,
		fileStorage: fileStorage,
	}
	if fileStorage != nil {
		r.attLoader = attachment.NewLoader(fileStorage)
	}
	return r
}

// AppendEvent 记录一次会话事件。
func (r *SessionRuntime) AppendEvent(ctx context.Context, event *model.Event) error {
	return r.sessionRep.AppendEvent(ctx, r.sessionID, event)
}

// UpdateStatus 更新会话状态。
func (r *SessionRuntime) UpdateStatus(ctx context.Context, status model.SessionStatus) error {
	return r.sessionRep.UpdateStatus(ctx, r.sessionID, status)
}

// LoadAttachments 将用户附件加载为 LLM 可见的上下文。
// 未配置存储或没有附件时返回 nil。
func (r *SessionRuntime) LoadAttachments(ctx context.Context, files []model.File, userMessage string) []attachment.FileContext {
	if r.attLoader == nil || len(files) == 0 {
		return nil
	}
	return r.attLoader.Load(ctx, files, userMessage)
}

// SyncFileToStorage 将沙箱中的文件同步到对象存储。
// 同步是尽力而为的旁路逻辑，失败只记录告警、不中断事件循环。
func (r *SessionRuntime) SyncFileToStorage(ctx context.Context, filePath string) error {
	if r.fileStorage == nil || r.sandbox == nil {
		return nil
	}

	// 以二进制通道下载：产物可能是图片/压缩包，走文本读取会因截断和编码问题损坏内容，
	// 且二进制字节数无法从文本长度还原。
	data, err := r.sandbox.DownloadFile(ctx, filePath)
	if err != nil {
		logger.WarnContext(ctx, "从沙箱下载文件失败", logger.String("filepath", filePath), logger.Err(err))
		return nil
	}

	// 对象 key 使用文件名，避免把沙箱绝对路径泄露或重复拼入对象存储路径。
	filename := path.Base(filePath)
	key := "agent/" + r.sessionID + "/" + filename

	extension := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(extension)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	err = r.fileStorage.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), mimeType)
	if err != nil {
		logger.WarnContext(ctx, "同步文件到存储失败", logger.String("filepath", filePath), logger.Err(err))
		return nil
	}

	// 创建文件记录
	file := &model.File{
		ID:        uuid.New().String(),
		SessionID: r.sessionID,
		Filename:  filename,
		Filepath:  filePath,
		Key:       key,
		Extension: extension,
		MimeType:  mimeType,
		Size:      int64(len(data)),
		CreatedAt: time.Now(),
	}
	if err := r.fileRep.Create(ctx, file); err != nil {
		logger.WarnContext(ctx, "创建文件记录失败", logger.String("filepath", filePath), logger.Err(err))
	}

	return nil
}

// StoreToolArtifacts 把工具产出里的二进制展示数据写入对象存储，
// 并在 Display 中用轻量文件引用替换掉原始字节。
//
// 工具结果会随 tool_called 事件进入 SSE 与事件库，截图之类的 base64 一旦直传
// 就会让事件体积和 token 双双膨胀。落到对象存储后，事件里只剩文件 ID，
// UI 走文件下载接口取内容，既有预览又能保持事件可重放。
//
// 预览是尽力而为的旁路能力：未配置存储或落盘失败时静默丢弃产物，
// 不影响工具结果本身。
func (r *SessionRuntime) StoreToolArtifacts(ctx context.Context, result *model.ToolResult) {
	if result == nil || len(result.Artifacts) == 0 {
		return
	}
	artifacts := result.Artifacts
	result.Artifacts = nil

	if r == nil || r.fileStorage == nil || r.fileRep == nil {
		logger.WarnContext(ctx, "未配置文件存储，丢弃工具展示产物",
			logger.Int("count", len(artifacts)))
		return
	}

	for key, artifact := range artifacts {
		ref := r.storeArtifact(ctx, key, artifact)
		if ref == nil {
			continue
		}
		result.WithDisplay(key, ref)
	}
}

// storeArtifact 落盘单份产物并登记文件记录，返回 UI 所需的文件引用。
// 任一步骤失败都只告警并返回 nil，由调用方跳过该项预览。
func (r *SessionRuntime) storeArtifact(ctx context.Context, key string, artifact model.ToolArtifact) map[string]interface{} {
	// 用随机文件名避免同会话多次截图互相覆盖：每次调用都应留下独立的一帧。
	filename := uuid.New().String() + filepath.Ext(artifact.Filename)
	objectKey := "agent/" + r.sessionID + "/artifacts/" + filename

	if err := r.fileStorage.Upload(
		ctx, objectKey, bytes.NewReader(artifact.Data), int64(len(artifact.Data)), artifact.MimeType,
	); err != nil {
		logger.WarnContext(ctx, "上传工具展示产物失败",
			logger.String("artifact", key), logger.Err(err))
		return nil
	}

	file := &model.File{
		ID:        uuid.New().String(),
		SessionID: r.sessionID,
		Filename:  filename,
		Filepath:  objectKey,
		Key:       objectKey,
		Extension: filepath.Ext(filename),
		MimeType:  artifact.MimeType,
		Size:      int64(len(artifact.Data)),
		CreatedAt: time.Now(),
	}
	if err := r.fileRep.Create(ctx, file); err != nil {
		logger.WarnContext(ctx, "登记工具展示产物失败",
			logger.String("artifact", key), logger.Err(err))
		return nil
	}

	return map[string]interface{}{
		"file_id":   file.ID,
		"filename":  file.Filename,
		"mime_type": file.MimeType,
		"size":      file.Size,
	}
}

// SyncUserAttachmentsToSandbox 将用户上传文件同步到沙箱，并返回可供 LLM 使用的文件路径。
// 这里不直接把 file_id 透传给模型，因为模型侧只能消费沙箱内可读路径。
func (r *SessionRuntime) SyncUserAttachmentsToSandbox(ctx context.Context, attachments []model.File) ([]string, error) {
	if len(attachments) == 0 {
		return nil, nil
	}
	if r.fileStorage == nil || r.sandbox == nil {
		result := make([]string, 0, len(attachments))
		for _, attachment := range attachments {
			if attachment.Filepath != "" {
				result = append(result, attachment.Filepath)
			}
		}
		return result, nil
	}

	result := make([]string, 0, len(attachments))
	for _, file := range attachments {
		if file.ID == "" || file.Key == "" {
			continue
		}

		reader, err := r.fileStorage.Download(ctx, file.Key)
		if err != nil {
			logger.WarnContext(ctx, "下载用户附件失败",
				logger.String("session_id", r.sessionID),
				logger.String("file_id", file.ID),
				logger.Err(err))
			continue
		}

		data, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			logger.WarnContext(ctx, "读取用户附件失败",
				logger.String("session_id", r.sessionID),
				logger.String("file_id", file.ID),
				logger.Err(err))
			continue
		}
		if closeErr != nil {
			logger.WarnContext(ctx, "关闭用户附件失败",
				logger.String("session_id", r.sessionID),
				logger.String("file_id", file.ID),
				logger.Err(closeErr))
		}

		filename := filepath.Base(file.Filename)
		if filename == "." || filename == string(filepath.Separator) || filename == "" {
			filename = file.ID
		}
		sandboxPath := filepath.Join("/home/ubuntu/upload", r.sessionID, filename)
		if _, err := r.sandbox.UploadFile(ctx, data, sandboxPath, filename); err != nil {
			logger.WarnContext(ctx, "上传用户附件到沙箱失败",
				logger.String("session_id", r.sessionID),
				logger.String("file_id", file.ID),
				logger.String("sandbox_path", sandboxPath),
				logger.Err(err))
			continue
		}

		sandboxFile := file
		sandboxFile.Filepath = sandboxPath
		result = append(result, sandboxFile.Filepath)

		// 写入 files 表（替代旧 sessions.files JSONB），按 filepath 去重
		if r.fileRep != nil {
			existing, findErr := r.fileRep.GetBySessionAndFilepath(ctx, r.sessionID, sandboxFile.Filepath)
			if findErr != nil || existing == nil {
				_ = r.fileRep.Create(ctx, &sandboxFile)
			}
		}
	}

	return result, nil
}
