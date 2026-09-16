package agent

import (
	"context"
	"io"
	"mime"
	"path"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"

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
	sandbox     external.Sandbox
	fileStorage COSFileStorage
	attLoader   *attachment.Loader
}

// NewSessionRuntime 构造运行期协作者。
func NewSessionRuntime(
	sessionID string,
	sessionRep repository.SessionRepository,
	fileRep repository.FileRepository,
	sandbox external.Sandbox,
	fileStorage COSFileStorage,
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

	// 从沙箱读取文件
	result, err := r.sandbox.ReadFile(ctx, filePath, nil, nil, false, 0)
	if err != nil {
		logger.WarnContext(ctx, "从沙箱读取文件失败", logger.String("filepath", filePath), logger.Err(err))
		return nil
	}
	if !result.Success {
		logger.WarnContext(ctx, "从沙箱读取文件失败", logger.String("filepath", filePath), logger.String("message", result.Message))
		return nil
	}

	// 提取文件内容
	var content string
	if dataMap, ok := result.Data.(map[string]interface{}); ok {
		if c, ok := dataMap["content"].(string); ok {
			content = c
		}
	}

	// 对象 key 使用文件名，避免把沙箱绝对路径泄露或重复拼入对象存储路径。
	filename := path.Base(filePath)
	key := "agent/" + r.sessionID + "/" + filename
	err = r.fileStorage.Upload(ctx, key, &readerWrapper{data: []byte(content)}, int64(len(content)), "text/plain")
	if err != nil {
		logger.WarnContext(ctx, "同步文件到存储失败", logger.String("filepath", filePath), logger.Err(err))
		return nil
	}

	// 创建文件记录
	extension := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(extension)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	file := &model.File{
		ID:        uuid.New().String(),
		SessionID: r.sessionID,
		Filename:  filename,
		Filepath:  filePath,
		Key:       key,
		Extension: extension,
		MimeType:  mimeType,
		Size:      int64(len(content)),
		CreatedAt: time.Now(),
	}
	if err := r.fileRep.Create(ctx, file); err != nil {
		logger.WarnContext(ctx, "创建文件记录失败", logger.String("filepath", filePath), logger.Err(err))
	}

	return nil
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

// readerWrapper 将 []byte 适配为 io.Reader。
type readerWrapper struct {
	data []byte
	pos  int
}

func (r *readerWrapper) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
