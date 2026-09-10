package agent

import (
	"context"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"mime"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/Huang131/go-manus/api/internal/agent/attachment"
	"github.com/Huang131/go-manus/api/internal/external"
	"github.com/Huang131/go-manus/api/internal/llmcore"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/google/uuid"

	"github.com/Huang131/go-manus/api/pkg/logger"
)

const (
	// Pop 重试配置
	popRetryBaseDelay = 100 * time.Millisecond // 基础重试延迟
	popRetryMaxDelay  = 1 * time.Second        // 最大重试延迟
	popRetryMaxCount  = 10                     // 最大连续错误次数
)

// AgentTaskRunner 基于 Agent 智能体的任务运行器
// 对齐 Python 版本的 AgentTaskRunner
type AgentTaskRunner struct {
	mu          sync.Mutex
	sessionID   string
	config      *AgentConfig
	llm         external.LLM
	tools       []Tool
	flow        *PlannerReActFlow
	sessionRep  repository.SessionRepository
	fileRep     repository.FileRepository
	sandbox     external.Sandbox
	fileStorage COSFileStorage
	attLoader   *attachment.Loader
}

// COSFileStorage 文件存储接口（简化版）
type COSFileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	GetURL(ctx context.Context, key string) (string, error)
}

// AgentTaskRunnerConfig AgentTaskRunner 配置
type AgentTaskRunnerConfig struct {
	SessionID   string
	AgentConfig *AgentConfig
	LLM         external.LLM
	Tools       []Tool
	SessionRep  repository.SessionRepository
	FileRep     repository.FileRepository
	Sandbox     external.Sandbox
	FileStorage COSFileStorage
}

// NewAgentTaskRunner 创建任务运行器
func NewAgentTaskRunner(cfg *AgentTaskRunnerConfig) *AgentTaskRunner {
	runner := &AgentTaskRunner{
		sessionID:   cfg.SessionID,
		config:      cfg.AgentConfig,
		llm:         cfg.LLM,
		tools:       cfg.Tools,
		sessionRep:  cfg.SessionRep,
		fileRep:     cfg.FileRep,
		sandbox:     cfg.Sandbox,
		fileStorage: cfg.FileStorage,
	}
	if cfg.FileStorage != nil {
		runner.attLoader = attachment.NewLoader(cfg.FileStorage)
	}

	// 创建流程
	runner.flow = NewPlannerReActFlow(cfg.SessionID, cfg.AgentConfig, cfg.LLM, cfg.Tools)

	return runner
}

// Invoke 实现 TaskRunner 接口
// 从 task.input_stream 获取事件，执行 Flow，结果写入 task.output_stream
// 对齐 Python: await self._task_runner.invoke(self)
func (r *AgentTaskRunner) Invoke(ctx context.Context, task *RedisStreamTask) error {
	r.mu.Lock()
	if r.flow == nil {
		r.flow = NewPlannerReActFlow(r.sessionID, r.config, r.llm, r.tools)
	}
	r.mu.Unlock()

	// 首次运行，更新会话状态为运行中
	if r.flow.GetPlan() == nil {
		if err := r.sessionRep.UpdateStatus(ctx, r.sessionID, model.SessionStatusRunning); err != nil {
			logger.WarnContext(ctx, "更新会话运行状态失败",
				logger.String("session_id", r.sessionID),
				logger.Err(err))
		}
	}

	logger.InfoContext(ctx, "AgentTaskRunner 开始执行",
		logger.String("session_id", r.sessionID),
		logger.String("task_id", task.ID()))

	// 死循环修复：添加重试延迟和最大错误次数限制
	var popRetryCount int
	var popRetryDelay = popRetryBaseDelay

	// 从 input_stream 循环获取事件并处理
	for {
		// 检查任务是否已取消或完成
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "AgentTaskRunner 上下文取消，退出执行",
				logger.String("task_id", task.ID()))
			return ctx.Err()
		case <-task.DoneChan():
			logger.InfoContext(ctx, "AgentTaskRunner 任务完成，退出执行",
				logger.String("task_id", task.ID()))
			return nil
		default:
		}

		// 阻塞获取输入消息
		_, data, err := task.InputStream().Pop(ctx)
		if err != nil {
			// 检查是否是 context 取消
			if ctx.Err() != nil {
				logger.InfoContext(ctx, "AgentTaskRunner 上下文取消，退出执行",
					logger.String("task_id", task.ID()))
				return ctx.Err()
			}
			// 检查是否是任务完成信号
			if task.Done() {
				logger.InfoContext(ctx, "AgentTaskRunner 任务完成，退出执行",
					logger.String("task_id", task.ID()))
				return nil
			}

			// 死循环修复：累计错误次数，超过阈值则退出
			popRetryCount++
			if popRetryCount >= popRetryMaxCount {
				logger.ErrorContext(ctx, "获取输入消息连续失败次数过多，退出执行",
					logger.String("task_id", task.ID()),
					logger.Int("retry_count", popRetryCount),
					logger.Err(err))
				return fmt.Errorf("pop retry exceeded max count: %d", popRetryMaxCount)
			}

			// 指数退避延迟
			logger.WarnContext(ctx, "获取输入消息失败，等待重试",
				logger.String("task_id", task.ID()),
				logger.Int("retry_count", popRetryCount),
				logger.Dur("retry_delay", popRetryDelay),
				logger.Err(err))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(popRetryDelay):
				// 指数退避，最大延迟限制
				popRetryDelay = popRetryDelay * 2
				if popRetryDelay > popRetryMaxDelay {
					popRetryDelay = popRetryMaxDelay
				}
			}
			continue
		}

		// 重置错误计数
		popRetryCount = 0
		popRetryDelay = popRetryBaseDelay

		if data == "" {
			// 无消息，继续等待
			continue
		}

		// 解析事件
		var inputEvent model.MessageEvent
		if err := sonic.UnmarshalString(data, &inputEvent); err != nil {
			logger.WarnContext(ctx, "解析输入事件失败",
				logger.String("data", data),
				logger.Err(err))
			continue
		}

		attachments, err := r.syncUserAttachmentsToSandbox(ctx, inputEvent.Attachments)
		if err != nil {
			logger.WarnContext(ctx, "同步用户附件失败",
				logger.String("session_id", r.sessionID),
				logger.Err(err))
		}

		// 加载附件内容到 LLM 上下文（解决"只列计划"问题）
		var attachmentContexts []attachment.FileContext
		if r.attLoader != nil && len(inputEvent.Attachments) > 0 {
			contexts := r.attLoader.Load(ctx, inputEvent.Attachments, inputEvent.Message)
			attachmentContexts = contexts
			if len(contexts) > 0 {
				logger.InfoContext(ctx, "已加载附件内容到 LLM 上下文",
					logger.String("session_id", r.sessionID),
					logger.Int("count", len(contexts)))
			}
		}

		// 转换为 Flow 需要的任务输入
		input := &TaskInput{
			Message: llmcore.Message{
				Role:        llmcore.MessageRole(inputEvent.Role),
				ContentText: inputEvent.Message,
				Attachments: attachments,
			},
			AttachmentContexts: attachmentContexts,
		}

		// 运行 Flow
		eventChan := r.flow.Invoke(ctx, input)

		// 处理 Flow 输出事件
		for event := range eventChan {
			// 业务事件先序列化为 payload
			eventJSON, err := sonic.Marshal(event)
			if err != nil {
				logger.ErrorContext(ctx, "序列化事件失败",
					logger.String("task_id", task.ID()),
					logger.Err(err))
				continue
			}

			// 生成事件元数据：event_id 与 created_at 在创建时确定，
			// 平铺进业务 payload（对齐 Python 版 BaseEventData 结构），
			// Redis 流与 DB 各存一份，SSE 直传无需再解析重组。
			eventID := uuid.New().String()
			eventCreatedAt := time.Now()

			// 业务事件序列化为 payload 并注入元数据
			var payload map[string]interface{}
			if err := sonic.Unmarshal(eventJSON, &payload); err == nil {
				payload["event_id"] = eventID
				payload["created_at"] = eventCreatedAt.Unix()
				eventJSON, err = sonic.Marshal(payload)
				if err != nil {
					logger.ErrorContext(ctx, "序列化事件 payload 失败",
						logger.String("task_id", task.ID()),
						logger.Err(err))
					continue
				}
			}
			// payload 不是对象时（罕见），保留原始 payload，仅元数据落在信封上

			// 同时包一层 model.Event（与 DB 一致），并写入 Redis Stream，
			// 否则 GetOutput 反序列化得到的是业务 payload，event.Type 永远是空，
			// SSE 推送时没有 event:xxx 业务类型行，前端 lastEventIdRef 也拿不到。
			baseEvent := &model.Event{
				ID:        eventID,
				Type:      event.GetType(),
				CreatedAt: eventCreatedAt,
				Data:      eventJSON,
			}
			wrappedJSON, err := sonic.Marshal(baseEvent)
			if err != nil {
				logger.ErrorContext(ctx, "序列化 model.Event 失败",
					logger.String("task_id", task.ID()),
					logger.Err(err))
				continue
			}

			outputID, err := task.OutputStream().Put(ctx, string(wrappedJSON))
			if err != nil {
				logger.WarnContext(ctx, "写入 output_stream 失败",
					logger.String("task_id", task.ID()),
					logger.Err(err))
			}

			// 同步到会话数据库
			if err := r.sessionRep.AppendEvent(ctx, r.sessionID, baseEvent); err != nil {
				logger.WarnContext(ctx, "添加事件到会话失败",
					logger.String("session_id", r.sessionID),
					logger.Err(err))
			}

			// 处理不同类型的事件
			switch e := event.(type) {
			case *model.PlanEvent:
				if e.Status == model.PlanEventStatusCompleted {
					// 计划完成，更新会话状态
					if err := r.sessionRep.UpdateStatus(ctx, r.sessionID, model.SessionStatusCompleted); err != nil {
						logger.WarnContext(ctx, "更新会话完成状态失败",
							logger.String("session_id", r.sessionID),
							logger.Err(err))
					}
				}
			case *model.ErrorEvent:
				logger.ErrorContext(ctx, "Agent 运行出错", logger.String("error", e.Message))
			case *model.StepEvent:
				if e.Status == model.StepEventStatusCompleted && e.Step.Success {
					// 步骤完成，同步附件文件
					for _, filePath := range e.Step.Attachments {
						_ = r.syncFileToStorage(ctx, filePath)
					}
				}
			}

			logger.DebugContext(ctx, "AgentTaskRunner 输出事件",
				logger.String("task_id", task.ID()),
				logger.String("event_id", outputID),
				logger.String("event_type", string(event.GetType())))
		}
	}
}

// Destroy 实现 TaskRunner 接口
// 销毁运行器，释放资源
func (r *AgentTaskRunner) Destroy() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.flow != nil {
		// 清理 flow 资源
		r.flow = nil
	}

	logger.Info("AgentTaskRunner 已销毁",
		logger.String("session_id", r.sessionID))

	return nil
}

// OnDone 实现 TaskRunner 接口
// 任务完成时的回调
func (r *AgentTaskRunner) OnDone(task *RedisStreamTask) {
	logger.Info("AgentTaskRunner 任务完成回调",
		logger.String("task_id", task.ID()),
		logger.String("session_id", r.sessionID))

	// 可选：更新会话状态为完成
	// _ = r.sessionRep.UpdateStatus(context.Background(), r.sessionID, model.SessionStatusCompleted)
}

// Done 返回任务是否完成
func (r *AgentTaskRunner) Done() bool {
	if r.flow == nil {
		return true
	}
	return r.flow.Done()
}

// GetStatus 返回当前状态
func (r *AgentTaskRunner) GetStatus() FlowStatus {
	if r.flow == nil {
		return FlowStatusIdle
	}
	return r.flow.GetStatus()
}

// GetPlan 返回当前计划
func (r *AgentTaskRunner) GetPlan() *model.Plan {
	if r.flow == nil {
		return nil
	}
	return r.flow.GetPlan()
}

// syncFileToStorage 将沙箱中的文件同步到存储。
// 同步是尽力而为的旁路逻辑，失败只记录告警、不中断事件循环。
func (r *AgentTaskRunner) syncFileToStorage(ctx context.Context, filePath string) error {
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

// syncUserAttachmentsToSandbox 将用户上传文件同步到沙箱，并返回可供 LLM 使用的文件路径。
// 这里不直接把 file_id 透传给模型，因为模型侧只能消费沙箱内可读路径。
func (r *AgentTaskRunner) syncUserAttachmentsToSandbox(ctx context.Context, attachments []model.File) ([]string, error) {
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

// readerWrapper io.Reader 实现
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
