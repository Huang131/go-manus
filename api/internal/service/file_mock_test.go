package service

import (
	"context"
	"io"
	"strings"

	"github.com/Huang131/go-manus/api/internal/model"
	"github.com/Huang131/go-manus/api/internal/repository"
)

// MockFileRepository 可配置的 FileRepository mock，供 service 包所有测试共享。
// 统一了原先分散在 file_service_test / file_cleanup_service_test /
// session_service_test 中的 4 份独立实现，避免接口演化时多处契约漂移。
type MockFileRepository struct {
	// 返回数据
	file               *model.File              // GetByID / GetBySessionAndID 的返回（会话匹配时）
	existingByName     *model.File              // GetBySessionAndFilename 的返回
	filesBySession     map[string][]*model.File // ListBySessionID 的返回
	expiredFiles       []*model.File            // GetExpiredFiles 返回的文件列表
	expiredFilesRepeat bool                     // 为真时每次调用都返回，用于验证清理循环的终止条件

	// 可注入错误
	getErr         error
	createErr      error
	deleteErr      error
	deleteByIDsErr error

	// 记录的行为
	createdKey   string
	deletedID    string
	deletedIDs   []string
	expiredCalls int
}

func NewMockFileRepository() *MockFileRepository {
	return &MockFileRepository{filesBySession: map[string][]*model.File{}}
}

func (m *MockFileRepository) Create(_ context.Context, f *model.File) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdKey = f.Key
	return nil
}

func (m *MockFileRepository) GetByID(_ context.Context, id string) (*model.File, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.file != nil && m.file.ID == id {
		return m.file, nil
	}
	return nil, nil
}

func (m *MockFileRepository) GetBySessionAndID(_ context.Context, sessionID, id string) (*model.File, error) {
	if m.file != nil && m.file.ID == id && m.file.SessionID == sessionID {
		return m.file, nil
	}
	return nil, nil
}

func (m *MockFileRepository) GetBySessionAndFilepath(context.Context, string, string) (*model.File, error) {
	return nil, nil
}

func (m *MockFileRepository) GetBySessionAndFilename(_ context.Context, _, _ string) (*model.File, error) {
	return m.existingByName, nil
}

func (m *MockFileRepository) GetBySessionAndHash(context.Context, string, string) (*model.File, error) {
	return nil, nil
}

func (m *MockFileRepository) Update(context.Context, *model.File) error { return nil }

func (m *MockFileRepository) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deletedID = id
	return nil
}

func (m *MockFileRepository) DeleteBySessionID(_ context.Context, sessionID string) error {
	delete(m.filesBySession, sessionID)
	return nil
}

func (m *MockFileRepository) ListBySessionID(_ context.Context, sessionID string) ([]*model.File, error) {
	return m.filesBySession[sessionID], nil
}

// GetExpiredFiles 默认仅首次调用返回 expiredFiles，防止清理循环在测试中无限查询；
// 设置 expiredFilesRepeat 后每次调用都返回，用于验证清理循环的终止条件。
func (m *MockFileRepository) GetExpiredFiles(context.Context, string, int64) ([]*model.File, error) {
	m.expiredCalls++
	if !m.expiredFilesRepeat && m.expiredCalls > 1 {
		return nil, nil
	}
	return m.expiredFiles, nil
}

func (m *MockFileRepository) DeleteByIDs(_ context.Context, ids []string) (int64, error) {
	if m.deleteByIDsErr != nil {
		return 0, m.deleteByIDsErr
	}
	m.deletedIDs = append([]string(nil), ids...)
	return int64(len(ids)), nil
}

func (m *MockFileRepository) GetFilesBySessionIDs(context.Context, []string) ([]*model.File, error) {
	return nil, nil
}

func (m *MockFileRepository) WithTx(_ context.Context, fn func(repository.FileRepository) error) error {
	return fn(m)
}

var _ repository.FileRepository = (*MockFileRepository)(nil)

// MockFileStorage 可配置的 FileStorage mock，供 service 包所有测试共享。
// 统一了原先分散在 file_service_test / file_cleanup_service_test 中的 2 份实现。
type MockFileStorage struct {
	// 可注入错误
	uploadErr   error
	downloadErr error
	deleteErr   error

	// 返回内容
	downloadContent string // Download 返回的内容，默认 "file-content"

	// 记录的行为
	uploadedKey string
	downloaded  string
	deletedKeys []string
}

func (s *MockFileStorage) Upload(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	if s.uploadErr != nil {
		return s.uploadErr
	}
	s.uploadedKey = key
	return nil
}

func (s *MockFileStorage) Download(_ context.Context, key string) (io.ReadCloser, error) {
	if s.downloadErr != nil {
		return nil, s.downloadErr
	}
	s.downloaded = key
	content := s.downloadContent
	if content == "" {
		content = "file-content"
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (s *MockFileStorage) Delete(_ context.Context, key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedKeys = append(s.deletedKeys, key)
	return nil
}

func (s *MockFileStorage) GetURL(_ context.Context, key string) (string, error) {
	return "http://storage.local/" + key, nil
}

var _ FileStorage = (*MockFileStorage)(nil)
