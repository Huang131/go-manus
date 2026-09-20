package service

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
	"github.com/google/uuid"
)

// SessionService 会话服务接口
type SessionService interface {
	CreateSession(ctx context.Context) (*model.Session, error)
	GetSession(ctx context.Context, id string) (*model.Session, error)
	GetAllSessions(ctx context.Context) ([]*model.Session, error)
	ListSessions(ctx context.Context, limit, offset int) ([]*model.Session, int, error)
	DeleteSession(ctx context.Context, id string) error
	// RenameSession 重命名会话标题，标题去空格后不得为空，且最长 100 个字符
	RenameSession(ctx context.Context, id string, title string) error
	ClearUnreadCount(ctx context.Context, id string) error
	// GetSessionFiles 返回会话的文件列表，数据源统一为 files 表，单一数据源。
	// fileRepo 未注入时返回 FailedPrecondition。
	GetSessionFiles(ctx context.Context, id string) ([]*model.File, error)
	// AppendEvent 追加事件。仓储层会一并回填事件 ID/时间戳、更新 latest_message，
	// 并对 assistant 的完整回复递增 unread_message_count。
	AppendEvent(ctx context.Context, sessionID string, event *model.Event) error

	// GetVNCURL 返回会话对应的 VNC WebSocket 地址。
	// go-manus 当前使用单一共享 sandbox 服务（区别于 mooc-manus 的 per-session Docker），
	// 因此 VNC URL 与 session 无关；保留 sessionID 入参是为了对齐 mooc-manus 接口契约
	// （get_vnc_url(session_id)），同时校验会话存在性。
	// sandboxAddress 未配置时返回 FailedPrecondition。
	GetVNCURL(ctx context.Context, sessionID string) (string, error)
}

// DefaultSessionListLimit 是会话列表未指定 limit 时的默认条数。
// HTTP 层与服务层共用同一常量，避免两处各写一份魔数而改漏。
const DefaultSessionListLimit = 20

// defaultVNCPort 是 sandbox 的 VNC WebSocket 端口。
// go-manus 使用单一共享 sandbox，各会话共用同一 VNC 端口。
const defaultVNCPort = 5901

// DefaultSessionService 会话服务默认实现
type DefaultSessionService struct {
	repo           repository.SessionRepository
	fileRepo       repository.FileRepository
	sandboxAddress string
	vncPort        int
}

// NewSessionService 创建会话服务。
// fileRepo / sandboxAddress 允许为空，表示对应能力未启用
func NewSessionService(repo repository.SessionRepository, fileRepo repository.FileRepository, sandboxAddress string) SessionService {
	return &DefaultSessionService{
		repo:           repo,
		fileRepo:       fileRepo,
		sandboxAddress: sandboxAddress,
		vncPort:        defaultVNCPort,
	}
}

// CreateSession 创建会话 (固定标题为"新对话"，与原项目一致)
func (s *DefaultSessionService) CreateSession(ctx context.Context) (*model.Session, error) {
	now := time.Now()
	session := &model.Session{
		ID:                 uuid.New().String(),
		Title:              "新对话",
		UnreadMessageCount: 0,
		LatestMessage:      "",
		LatestMessageAt:    nil,
		Events:             []model.Event{},
		Status:             model.SessionStatusPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

// GetSession 获取会话
func (s *DefaultSessionService) GetSession(ctx context.Context, id string) (*model.Session, error) {
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, apperr.NotFound("会话不存在")
	}
	return session, nil
}

// GetAllSessions 获取所有会话
func (s *DefaultSessionService) GetAllSessions(ctx context.Context) ([]*model.Session, error) {
	return s.repo.GetAll(ctx)
}

// ListSessions 获取会话列表
func (s *DefaultSessionService) ListSessions(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	if limit <= 0 {
		limit = DefaultSessionListLimit
	}
	return s.repo.List(ctx, limit, offset)
}

// DeleteSession 删除会话（软删除）。
// 仓储层以 RowsAffected==0 判定会话不存在并返回 ErrSessionNotFound，
// 因此这里不前置查询，避免多一次数据库往返。
func (s *DefaultSessionService) DeleteSession(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ClearUnreadCount 清空未读数
func (s *DefaultSessionService) ClearUnreadCount(ctx context.Context, id string) error {
	return s.repo.SetUnreadCount(ctx, id, 0)
}

// GetSessionFiles 获取会话的文件列表，先校验会话存在再读取 files 表
func (s *DefaultSessionService) GetSessionFiles(ctx context.Context, id string) ([]*model.File, error) {
	// 先确认 session 存在
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, apperr.NotFound("会话不存在")
	}
	if s.fileRepo == nil {
		return nil, apperr.FailedPrecondition("file repository 未注入，GetSessionFiles 不可用")
	}
	files, err := s.fileRepo.ListBySessionID(ctx, id)
	if err != nil {
		return nil, err
	}
	return files, nil
}

// AppendEvent 追加事件；ID/时间戳回填、latest_message 投影与未读数累计由仓储层完成
func (s *DefaultSessionService) AppendEvent(ctx context.Context, sessionID string, event *model.Event) error {
	return s.repo.AppendEvent(ctx, sessionID, event)
}

// RenameSession 重命名会话标题
func (s *DefaultSessionService) RenameSession(ctx context.Context, id string, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return apperr.BadRequest("标题不能为空")
	}
	if len([]rune(title)) > 100 {
		return apperr.BadRequest("标题最长 100 个字符")
	}
	return s.repo.UpdateTitle(ctx, id, title)
}

// GetVNCURL 从 sandboxAddress (http(s)://host[:port]) 派生 ws://host:5901 或 wss://host:5901。
func (s *DefaultSessionService) GetVNCURL(ctx context.Context, sessionID string) (string, error) {
	session, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	// repo 对不存在的会话返回 (nil, nil)：不校验会让任意 sessionID 建立 VNC 代理
	if session == nil {
		return "", apperr.NotFound("会话不存在: " + sessionID)
	}
	if s.sandboxAddress == "" {
		return "", apperr.FailedPrecondition("sandbox 地址未配置")
	}

	u, err := url.Parse(s.sandboxAddress)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	if host == "" {
		return "", apperr.BadRequest("sandbox 地址缺少 host")
	}

	scheme := "ws"
	if strings.EqualFold(u.Scheme, "https") {
		scheme = "wss"
	}
	return scheme + "://" + host + ":" + strconv.Itoa(s.vncPort), nil
}
