package service

import (
	"context"
	"errors"
	"github.com/bytedance/sonic"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// MockSessionRepository 用于测试的 Repository Mock
type MockSessionRepository struct {
	sessions   map[string]*model.Session
	events     map[string][]model.Event
	createErr  error
	getErr     error
	deleteErr  error
	updateErr  error
	listLimit  int
	listOffset int
}

func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{
		sessions: make(map[string]*model.Session),
		events:   make(map[string][]model.Event),
	}
}

func (m *MockSessionRepository) Create(ctx context.Context, session *model.Session) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.sessions[session.ID] = session
	m.events[session.ID] = []model.Event{}
	return nil
}

func (m *MockSessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
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

func (m *MockSessionRepository) GetAll(ctx context.Context) ([]*model.Session, error) {
	result := make([]*model.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, nil
}

func (m *MockSessionRepository) List(ctx context.Context, limit, offset int) ([]*model.Session, int, error) {
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

func (m *MockSessionRepository) Update(ctx context.Context, session *model.Session) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *MockSessionRepository) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.sessions, id)
	delete(m.events, id)
	return nil
}

func (m *MockSessionRepository) AppendEvent(ctx context.Context, id string, event *model.Event) error {
	// 真实实现：更新 session.Events JSONB 字段
	session, ok := m.sessions[id]
	if !ok {
		return errors.New("会话不存在")
	}
	session.Events = append(session.Events, *event)
	m.events[id] = session.Events
	return nil
}

func (m *MockSessionRepository) UpdateTitle(ctx context.Context, id string, title string) error {
	if session, ok := m.sessions[id]; ok {
		session.Title = title
	}
	return nil
}

func (m *MockSessionRepository) UpdateLatestMessage(ctx context.Context, id string, message string) error {
	if session, ok := m.sessions[id]; ok {
		session.LatestMessage = message
	}
	return nil
}

func (m *MockSessionRepository) UpdateStatus(ctx context.Context, id string, status model.SessionStatus) error {
	if session, ok := m.sessions[id]; ok {
		session.Status = status
	}
	return nil
}

func (m *MockSessionRepository) IncrementUnreadCount(ctx context.Context, id string) error {
	if session, ok := m.sessions[id]; ok {
		session.UnreadMessageCount++
	}
	return nil
}

func (m *MockSessionRepository) DecrementUnreadCount(ctx context.Context, id string) error {
	if session, ok := m.sessions[id]; ok && session.UnreadMessageCount > 0 {
		session.UnreadMessageCount--
	}
	return nil
}

func (m *MockSessionRepository) SetUnreadCount(ctx context.Context, id string, count int) error {
	if session, ok := m.sessions[id]; ok {
		session.UnreadMessageCount = count
	}
	return nil
}

func (m *MockSessionRepository) WithTx(ctx context.Context, fn func(repo repository.SessionRepository) error) error {
	return fn(m)
}

// 确保 Mock 实现正确的接口
var _ repository.SessionRepository = (*MockSessionRepository)(nil)

func TestSessionService_CreateSession(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

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

func TestSessionService_GetSession(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

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
	svc := NewSessionService(repo)

	_, err := svc.GetSession(context.Background(), "nonexistent-id")
	if err == nil {
		t.Error("GetSession() should return error for nonexistent session")
	}
}

func TestSessionService_GetAllSessions(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

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
	svc := NewSessionService(repo)

	// 创建多个会话
	for i := 0; i < 5; i++ {
		svc.CreateSession(context.Background())
	}

	// 测试分页参数透传
	_, total, err := svc.ListSessions(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	// 分页切片是真实 SQL 的职责（集成测试覆盖），这里只验证 service 透传参数
	if repo.listLimit != 2 {
		t.Errorf("ListSessions() passed limit = %d, want 2", repo.listLimit)
	}
	if repo.listOffset != 0 {
		t.Errorf("ListSessions() passed offset = %d, want 0", repo.listOffset)
	}
	if total != 5 {
		t.Errorf("ListSessions() total = %d, want 5", total)
	}
}

func TestSessionService_ListSessions_DefaultLimit(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

	// 创建多个会话
	for i := 0; i < 25; i++ {
		svc.CreateSession(context.Background())
	}

	// 测试默认 limit
	_, total, err := svc.ListSessions(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	// 关键断言：service 把"未指定 limit（0）"归一化为默认 20 后透传给 repo
	if repo.listLimit != 20 {
		t.Errorf("ListSessions() passed limit = %d, want default 20", repo.listLimit)
	}
	if total != 25 {
		t.Errorf("ListSessions() total = %d, want 25", total)
	}
}

func TestSessionService_DeleteSession(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

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
	svc := NewSessionService(repo)

	err := svc.DeleteSession(context.Background(), "nonexistent-id")
	if err == nil {
		t.Error("DeleteSession() should return error for nonexistent session")
	}
}

func TestSessionService_ClearUnreadCount(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

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

func TestSessionService_GetSessionFiles(t *testing.T) {
	repo := NewMockSessionRepository()
	fileRepo := newMockFileRepo()
	svc := NewSessionServiceWithSandbox(repo, fileRepo, "")

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
	svc := NewSessionServiceWithSandbox(repo, newMockFileRepo(), "")

	_, err := svc.GetSessionFiles(context.Background(), "missing-session")
	if err == nil {
		t.Fatal("GetSessionFiles() error = nil, want not found")
	}
}

// newMockFileRepo 为 GetSessionFiles 测试提供最小化的 FileRepository mock
func newMockFileRepo() *MockFileRepository {
	return NewMockFileRepository()
}

func TestSessionService_AppendEvent(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

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

func TestSessionService_CreateSession_RepositoryError(t *testing.T) {
	repo := NewMockSessionRepository()
	repo.createErr = errors.New("database error")
	svc := NewSessionService(repo)

	_, err := svc.CreateSession(context.Background())
	if err == nil {
		t.Error("CreateSession() should return error when repository fails")
	}
}

// GetVNCURL 测试已由 vnc_test.go 覆盖（TestSessionService_GetVNCURL_Success, HTTPS, NotFound, EmptyAddress）
// 本文件不再重复
