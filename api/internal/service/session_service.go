package service

import (
	"context"
	"github.com/bytedance/sonic"
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
	IncrementUnreadCount(ctx context.Context, id string) error
	DecrementUnreadCount(ctx context.Context, id string) error
	ClearUnreadCount(ctx context.Context, id string) error
	GetSessionFiles(ctx context.Context, id string) ([]*model.File, error)
	AppendEvent(ctx context.Context, sessionID string, event *model.Event) error
	StreamSession(ctx context.Context, id string) (*model.Session, error)
	Chat(ctx context.Context, sessionID string, message string) error

	// GetVNCURL 返回会话对应的 VNC WebSocket 地址。
	// go-manus 当前使用单一共享 sandbox 服务（区别于 mooc-manus 的 per-session Docker），
	// 因此 VNC URL 与 session 无关；保留 sessionID 入参是为了对齐 mooc-manus 接口契约
	// （get_vnc_url(session_id)），同时校验会话存在性。
	GetVNCURL(ctx context.Context, sessionID string) (string, error)
}

// DefaultSessionService 会话服务默认实现
type DefaultSessionService struct {
	repo           repository.SessionRepository
	fileRepo       repository.FileRepository
	sandboxAddress string
	vncPort        int
}

// NewSessionService 创建会话服务（不含 fileRepo，GetSessionFiles 会报错）
func NewSessionService(repo repository.SessionRepository) SessionService {
	return &DefaultSessionService{
		repo:    repo,
		vncPort: 5901,
	}
}

// NewSessionServiceWithSandbox 创建带 sandbox 配置的会话服务
// 注意：fileRepo 为可选，GetSessionFiles 依赖它。如果未注入且被调用，方法内会返回错误。
func NewSessionServiceWithSandbox(repo repository.SessionRepository, fileRepo repository.FileRepository, sandboxAddress string) SessionService {
	return &DefaultSessionService{
		repo:           repo,
		fileRepo:       fileRepo,
		sandboxAddress: sandboxAddress,
		vncPort:        5901,
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
	return s.repo.GetByID(ctx, id)
}

// GetAllSessions 获取所有会话
func (s *DefaultSessionService) GetAllSessions(ctx context.Context) ([]*model.Session, error) {
	return s.repo.GetAll(ctx)
}

// ListSessions 获取会话列表
func (s *DefaultSessionService) ListSessions(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.List(ctx, limit, offset)
}

// DeleteSession 删除会话
func (s *DefaultSessionService) DeleteSession(ctx context.Context, id string) error {
	// 先检查会话是否存在
	session, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if session == nil {
		return apperr.NotFound("会话不存在")
	}
	return s.repo.Delete(ctx, id)
}

// ClearUnreadCount 清空未读数
func (s *DefaultSessionService) ClearUnreadCount(ctx context.Context, id string) error {
	return s.repo.SetUnreadCount(ctx, id, 0)
}

// IncrementUnreadCount 原子增加未读数
func (s *DefaultSessionService) IncrementUnreadCount(ctx context.Context, id string) error {
	return s.repo.IncrementUnreadCount(ctx, id)
}

// DecrementUnreadCount 原子减少未读数 (最低为0)
func (s *DefaultSessionService) DecrementUnreadCount(ctx context.Context, id string) error {
	return s.repo.DecrementUnreadCount(ctx, id)
}

// GetSessionFiles 获取会话的文件列表
// 统一走 files 表（替代旧 sessions.files JSONB），单一数据源，永不不一致
func (s *DefaultSessionService) GetSessionFiles(ctx context.Context, id string) ([]*model.File, error) {
	// 先确认 session 存在
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
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

// AppendEvent 追加事件
func (s *DefaultSessionService) AppendEvent(ctx context.Context, sessionID string, event *model.Event) error {
	return s.repo.AppendEvent(ctx, sessionID, event)
}

// StreamSession 流式获取会话
func (s *DefaultSessionService) StreamSession(ctx context.Context, id string) (*model.Session, error) {
	return s.repo.GetByID(ctx, id)
}

// Chat 发送消息 (将消息内容封装到事件数据中)
func (s *DefaultSessionService) Chat(ctx context.Context, sessionID string, message string) error {
	msgEvent := model.MessageEvent{
		Type:    model.EventTypeMessage,
		Role:    "user",
		Message: message,
	}
	data, _ := sonic.Marshal(msgEvent)

	event := &model.Event{
		ID:        uuid.New().String(),
		Type:      model.EventTypeMessage,
		CreatedAt: time.Now(),
		Data:      data,
	}
	return s.AppendEvent(ctx, sessionID, event)
}

// GetVNCURL 返回会话对应的 VNC WebSocket 地址。
// 实现：从 sandboxAddress (http(s)://host[:port]) 派生 ws://host:5901 或 wss://host:5901。
// 之所以保留 sessionID 入参，是为了对齐 mooc-manus 的 get_vnc_url(session_id) 接口契约
// （即使 go-manus 当前使用单一共享 sandbox，仍然校验会话存在）。
func (s *DefaultSessionService) GetVNCURL(ctx context.Context, sessionID string) (string, error) {
	if _, err := s.repo.GetByID(ctx, sessionID); err != nil {
		return "", err
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
