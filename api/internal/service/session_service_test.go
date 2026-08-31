package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mooc-manus/go-manus/api/internal/model"
	"github.com/mooc-manus/go-manus/api/internal/repository"
)

// MockSessionRepository 用于测试的 Repository Mock
type MockSessionRepository struct {
	sessions  map[string]*model.Session
	events    map[string][]model.Event
	createErr error
	getErr    error
	deleteErr error
	updateErr error
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
		return nil, errors.New("会话不存在")
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
	// 真实实现：使用 SQL LIMIT/OFFSET
	allSessions := make([]*model.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		allSessions = append(allSessions, s)
	}

	// 计算分页
	total := len(allSessions)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	return allSessions[start:end], total, nil
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

func (m *MockSessionRepository) AddFile(ctx context.Context, id string, file *model.File) error {
	return nil
}

func (m *MockSessionRepository) RemoveFile(ctx context.Context, id string, fileID string) error {
	return nil
}

func (m *MockSessionRepository) GetFileByPath(ctx context.Context, id string, filepath string) (*model.File, error) {
	return nil, nil
}

func (m *MockSessionRepository) GetMemory(ctx context.Context, id string, agentName string) (*model.Memory, error) {
	return nil, nil
}

func (m *MockSessionRepository) SaveMemory(ctx context.Context, id string, agentName string, memory *model.Memory) error {
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
	if len(session.Files) != 0 {
		t.Errorf("Session Files length = %d, want 0", len(session.Files))
	}
	if session.Memories == nil {
		t.Error("Session Memories should not be nil")
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

	// 测试分页
	sessions, total, err := svc.ListSessions(context.Background(), 2, 0)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("ListSessions() returned %d sessions, want 2", len(sessions))
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
	sessions, total, err := svc.ListSessions(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	// 默认 limit 为 20
	if len(sessions) != 20 {
		t.Errorf("ListSessions() returned %d sessions with default limit, want 20", len(sessions))
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
	svc := NewSessionService(repo)

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

func TestSessionService_AppendEvent(t *testing.T) {
	repo := NewMockSessionRepository()
	svc := NewSessionService(repo)

	// 创建会话
	session, _ := svc.CreateSession(context.Background())

	// 添加事件
	msgEvent := &model.MessageEvent{Type: model.EventTypeMessage, Role: "user", Message: "Hello"}
	eventData, _ := json.Marshal(msgEvent)
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
