package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// MockSessionRepository 用于测试的 Repository Mock。
//
// 契约：写操作命中不存在的会话必须返回 apperr.ErrSessionNotFound，与
// PostgresSessionRepository 的 RowsAffected==0 行为一致。否则 service 的
// 错误传播路径在单测中不可达，mock 的"宽容"会掩盖真实回归。
type MockSessionRepository struct {
	sessions   map[string]*model.Session
	events     map[string][]model.Event
	createErr  error
	getErr     error
	deleteErr  error
	listLimit  int
	listOffset int
}

func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{
		sessions: make(map[string]*model.Session),
		events:   make(map[string][]model.Event),
	}
}

// mustGet 按真实仓储契约取会话：不存在时返回 ErrSessionNotFound。
func (m *MockSessionRepository) mustGet(id string) (*model.Session, error) {
	session, ok := m.sessions[id]
	if !ok {
		return nil, apperr.ErrSessionNotFound
	}
	return session, nil
}

func (m *MockSessionRepository) Create(_ context.Context, session *model.Session) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.sessions[session.ID] = session
	m.events[session.ID] = []model.Event{}
	return nil
}

func (m *MockSessionRepository) GetByID(_ context.Context, id string) (*model.Session, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	session, ok := m.sessions[id]
	if !ok {
		// 与真实仓储契约一致：not found 返回 (nil, nil)，由 service 层判断 nil 转 NotFound
		return nil, nil
	}
	return session, nil
}

func (m *MockSessionRepository) GetAll(_ context.Context) ([]*model.Session, error) {
	result := make([]*model.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *MockSessionRepository) List(_ context.Context, limit, offset int) ([]*model.Session, int, error) {
	// 只记录 service 传入的分页参数，不做切片分页：分页是真实 SQL LIMIT/OFFSET
	// 的职责（集成测试覆盖），mock 复刻分页逻辑会与真实实现漂移。
	m.listLimit = limit
	m.listOffset = offset

	result := make([]*model.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, len(result), nil
}

// Update 当前 service 层不调用，仅为满足接口实现。
func (m *MockSessionRepository) Update(_ context.Context, session *model.Session) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *MockSessionRepository) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, err := m.mustGet(id); err != nil {
		return err
	}
	delete(m.sessions, id)
	delete(m.events, id)
	return nil
}

func (m *MockSessionRepository) AppendEvent(_ context.Context, id string, event *model.Event) error {
	session, err := m.mustGet(id)
	if err != nil {
		return err
	}
	session.Events = append(session.Events, *event)
	m.events[id] = session.Events
	return nil
}

func (m *MockSessionRepository) UpdateTitle(_ context.Context, id string, title string) error {
	session, err := m.mustGet(id)
	if err != nil {
		return err
	}
	session.Title = title
	return nil
}

func (m *MockSessionRepository) UpdateLatestMessage(_ context.Context, id string, message string) error {
	session, err := m.mustGet(id)
	if err != nil {
		return err
	}
	session.LatestMessage = message
	return nil
}

func (m *MockSessionRepository) UpdateStatus(_ context.Context, id string, status model.SessionStatus) error {
	session, err := m.mustGet(id)
	if err != nil {
		return err
	}
	session.Status = status
	return nil
}

func (m *MockSessionRepository) IncrementUnreadCount(_ context.Context, id string) error {
	session, err := m.mustGet(id)
	if err != nil {
		return err
	}
	session.UnreadMessageCount++
	return nil
}

func (m *MockSessionRepository) DecrementUnreadCount(_ context.Context, id string) error {
	session, err := m.mustGet(id)
	if err != nil {
		return err
	}
	// 下限保护是真实 SQL GREATEST 的职责，这里只为让 mock 状态可读，
	// 该语义不在本层验证（Service fake 不模拟 SQL 语义）。
	if session.UnreadMessageCount > 0 {
		session.UnreadMessageCount--
	}
	return nil
}

func (m *MockSessionRepository) SetUnreadCount(_ context.Context, id string, count int) error {
	session, err := m.mustGet(id)
	if err != nil {
		return err
	}
	session.UnreadMessageCount = count
	return nil
}

func (m *MockSessionRepository) WithTx(_ context.Context, fn func(repo repository.SessionRepository) error) error {
	return fn(m)
}

// 确保 Mock 实现正确的接口
var _ repository.SessionRepository = (*MockSessionRepository)(nil)

// requireAppErrKind 断言 err 是携带指定 Kind 的业务错误。
// 只断言"有错"守护不了错误契约：NotFound 被误改成 Internal（500）时测试依然通过。
func requireAppErrKind(t *testing.T, err error, kind apperr.Kind) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("error = %v, want *apperr.Error", err)
	}
	if ae.Kind != kind {
		t.Errorf("error kind = %s, want %s", ae.Kind, kind)
	}
}

func TestSessionService_CreateSession(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	session, err := svc.CreateSession(context.Background())
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// 验证会话属性
	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.Title != "新对话" {
		t.Errorf("Session Title = %s, want 新对话", session.Title)
	}
	if session.UnreadMessageCount != 0 {
		t.Errorf("Session UnreadMessageCount = %d, want 0", session.UnreadMessageCount)
	}
	if session.Status != model.SessionStatusPending {
		t.Errorf("Session Status = %s, want pending", session.Status)
	}
	if len(session.Events) != 0 {
		t.Errorf("Session Events length = %d, want 0", len(session.Events))
	}
}

func TestSessionService_CreateSession_RepositoryError(t *testing.T) {
	repo := NewMockSessionRepository()
	repo.createErr = errors.New("database error")
	svc := NewSessionService(repo, nil, "")

	_, err := svc.CreateSession(context.Background())
	if !errors.Is(err, repo.createErr) {
		t.Errorf("CreateSession() error = %v, want repository error propagated", err)
	}
}

func TestSessionService_GetSession(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 先创建会话
	created, _ := svc.CreateSession(context.Background())

	// 获取会话
	session, err := svc.GetSession(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if session.ID != created.ID {
		t.Errorf("Session ID = %s, want %s", session.ID, created.ID)
	}
}

func TestSessionService_GetSession_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	_, err := svc.GetSession(context.Background(), "nonexistent-id")
	requireAppErrKind(t, err, apperr.KindNotFound)
}

func TestSessionService_GetSession_RepositoryError(t *testing.T) {
	repo := NewMockSessionRepository()
	repo.getErr = errors.New("database error")
	svc := NewSessionService(repo, nil, "")

	_, err := svc.GetSession(context.Background(), "any-id")
	if !errors.Is(err, repo.getErr) {
		t.Errorf("GetSession() error = %v, want repository error propagated", err)
	}
}

// GetAllSessions 是纯透传，这里守护"service 不额外过滤或截断仓储结果"。
func TestSessionService_GetAllSessions(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 创建多个会话
	svc.CreateSession(context.Background())
	svc.CreateSession(context.Background())
	svc.CreateSession(context.Background())

	sessions, err := svc.GetAllSessions(context.Background())
	if err != nil {
		t.Fatalf("GetAllSessions() error = %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("GetAllSessions() returned %d sessions, want 3", len(sessions))
	}
}

func TestSessionService_ListSessions(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 分页切片是真实 SQL 的职责（集成测试覆盖），这里只验证 service 原样透传分页参数
	if _, _, err := svc.ListSessions(context.Background(), 2, 10); err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if repo.listLimit != 2 {
		t.Errorf("ListSessions() passed limit = %d, want 2", repo.listLimit)
	}
	if repo.listOffset != 10 {
		t.Errorf("ListSessions() passed offset = %d, want 10", repo.listOffset)
	}
}

func TestSessionService_ListSessions_DefaultLimit(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 关键断言：service 把"未指定 limit（0）"归一化为默认值后透传给 repo
	if _, _, err := svc.ListSessions(context.Background(), 0, 0); err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if repo.listLimit != DefaultSessionListLimit {
		t.Errorf("ListSessions() passed limit = %d, want default %d", repo.listLimit, DefaultSessionListLimit)
	}
}

func TestSessionService_DeleteSession(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 创建会话
	session, _ := svc.CreateSession(context.Background())

	// 删除会话
	err := svc.DeleteSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// 验证会话已被删除
	_, err = svc.GetSession(context.Background(), session.ID)
	if err == nil {
		t.Error("GetSession() should return error after deletion")
	}
}

func TestSessionService_DeleteSession_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 仓储以 RowsAffected==0 返回哨兵错误，response 层据此映射 404
	err := svc.DeleteSession(context.Background(), "nonexistent-id")
	if !errors.Is(err, apperr.ErrSessionNotFound) {
		t.Errorf("DeleteSession() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionService_DeleteSession_RepositoryError(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")
	session, _ := svc.CreateSession(context.Background())

	repo.deleteErr = errors.New("database error")
	err := svc.DeleteSession(context.Background(), session.ID)
	if !errors.Is(err, repo.deleteErr) {
		t.Errorf("DeleteSession() error = %v, want repository error propagated", err)
	}
}

func TestSessionService_ClearUnreadCount(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 创建会话
	session, _ := svc.CreateSession(context.Background())
	repo.sessions[session.ID].UnreadMessageCount = 5

	// 清除未读数
	err := svc.ClearUnreadCount(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("ClearUnreadCount() error = %v", err)
	}

	// 验证未读数已清除
	updated, _ := svc.GetSession(context.Background(), session.ID)
	if updated.UnreadMessageCount != 0 {
		t.Errorf("UnreadMessageCount = %d, want 0", updated.UnreadMessageCount)
	}
}

func TestSessionService_ClearUnreadCount_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	err := svc.ClearUnreadCount(context.Background(), "nonexistent-id")
	if !errors.Is(err, apperr.ErrSessionNotFound) {
		t.Errorf("ClearUnreadCount() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionService_RenameSession(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	session, _ := svc.CreateSession(context.Background())

	// 首尾空格必须被裁剪后再落库
	err := svc.RenameSession(context.Background(), session.ID, "  我的标题  ")
	if err != nil {
		t.Fatalf("RenameSession() error = %v", err)
	}
	if got := repo.sessions[session.ID].Title; got != "我的标题" {
		t.Errorf("Title = %q, want %q", got, "我的标题")
	}
}

func TestSessionService_RenameSession_EmptyTitle(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	session, _ := svc.CreateSession(context.Background())

	// 纯空白标题去空格后为空，必须被拒绝
	err := svc.RenameSession(context.Background(), session.ID, "   ")
	requireAppErrKind(t, err, apperr.KindInvalidArgument)
}

func TestSessionService_RenameSession_TitleAtLimit(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	session, _ := svc.CreateSession(context.Background())

	// 100 个 rune 是上边界，必须放行（限制按 rune 而非 byte 计数）
	title := strings.Repeat("字", 100)
	if err := svc.RenameSession(context.Background(), session.ID, title); err != nil {
		t.Fatalf("RenameSession() error = %v, want nil at 100 runes", err)
	}
	if got := repo.sessions[session.ID].Title; got != title {
		t.Errorf("Title = %q, want %q", got, title)
	}
}

func TestSessionService_RenameSession_TitleTooLong(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	session, _ := svc.CreateSession(context.Background())

	// 101 个中文字符为 101 个 rune，必须被拒绝（按 byte 计数会误判为更短）
	err := svc.RenameSession(context.Background(), session.ID, strings.Repeat("字", 101))
	requireAppErrKind(t, err, apperr.KindInvalidArgument)
}

func TestSessionService_RenameSession_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	err := svc.RenameSession(context.Background(), "nonexistent-id", "标题")
	if !errors.Is(err, apperr.ErrSessionNotFound) {
		t.Errorf("RenameSession() error = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionService_GetSessionFiles(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, NewMockFileRepository(), "")

	// 创建会话
	session, _ := svc.CreateSession(context.Background())

	files, err := svc.GetSessionFiles(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("GetSessionFiles() error = %v", err)
	}

	if len(files) != 0 {
		t.Errorf("GetSessionFiles() returned %d files, want 0", len(files))
	}
}

func TestSessionService_GetSessionFiles_NotFound(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, NewMockFileRepository(), "")

	_, err := svc.GetSessionFiles(context.Background(), "missing-session")
	requireAppErrKind(t, err, apperr.KindNotFound)
}

// 未注入 fileRepo 时是文档承诺的 FailedPrecondition 契约，不是 panic 也不是 500
func TestSessionService_GetSessionFiles_RepositoryNotInjected(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	session, _ := svc.CreateSession(context.Background())

	_, err := svc.GetSessionFiles(context.Background(), session.ID)
	requireAppErrKind(t, err, apperr.KindFailedPrecondition)
}

func TestSessionService_AppendEvent(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo, nil, "")

	// 创建会话
	session, _ := svc.CreateSession(context.Background())

	// 添加事件
	msgEvent := &model.MessageEvent{Type: model.EventTypeMessage, Role: model.RoleUser, Message: "Hello"}
	eventData, _ := sonic.Marshal(msgEvent)
	event := &model.Event{
		ID:        "event-1",
		Type:      model.EventTypeMessage,
		CreatedAt: time.Now(),
		Data:      eventData,
	}

	err := svc.AppendEvent(context.Background(), session.ID, event)
	if err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	// 验证事件已添加 (从 repository 获取最新数据)
	updated, err := svc.GetSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if len(updated.Events) != 1 {
		t.Errorf("Session Events length = %d, want 1", len(updated.Events))
	}
}
