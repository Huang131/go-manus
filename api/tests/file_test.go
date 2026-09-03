//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// makeMultipartFile 创建 multipart 请求
func makeMultipartFile(fields map[string]string, fileName string, fileContent []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加字段
	for key, value := range fields {
		writer.WriteField(key, value)
	}

	// 添加文件
	if fileName != "" && fileContent != nil {
		part, _ := writer.CreateFormFile("file", fileName)
		part.Write(fileContent)
	}

	writer.Close()
	return body, writer.FormDataContentType()
}

// TestFileAPI_Upload_Success 测试成功上传文件（真实 MinIO）
func TestFileAPI_Upload_Success(t *testing.T) {
	// 1. 先创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)
	assert.Equal(t, http.StatusOK, sessionW.Code)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 2. 上传文件
	fileContent := []byte("Hello, World! This is a test file.")
	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "test.txt", fileContent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/files", body)
	req.Header.Set("Content-Type", contentType)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "上传文件应该成功")

	var uploadResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &uploadResp)
	fileData := uploadResp["data"].(map[string]interface{})

	fileID := fileData["id"].(string)
	assert.NotEmpty(t, fileID)
	assert.Equal(t, "test.txt", fileData["filename"])
	assert.Equal(t, int64(len(fileContent)), int64(fileData["size"].(float64)))
	assert.Equal(t, sessionID, fileData["session_id"])

	defer CleanupFile(t, fileID)
}

// TestFileAPI_Upload_MissingSession 测试缺少 session_id
func TestFileAPI_Upload_MissingSession(t *testing.T) {
	// 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 上传文件时不提供 session_id
	body, contentType := makeMultipartFile(nil, "test.txt", []byte("content"))
	_ = sessionID // 避免未使用警告

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/files", body)
	req.Header.Set("Content-Type", contentType)
	testServer.ServeHTTP(w, req)

	// 应该返回错误
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NotEqual(t, 0, resp["code"])
}

// TestFileAPI_Upload_MissingFile 测试缺少文件
func TestFileAPI_Upload_MissingFile(t *testing.T) {
	// 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 只发送 session_id，不发送文件
	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "", nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/files", body)
	req.Header.Set("Content-Type", contentType)
	testServer.ServeHTTP(w, req)

	// 应该返回错误
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFileAPI_Upload_LargeFile 测试上传大文件（1MB）
func TestFileAPI_Upload_LargeFile(t *testing.T) {
	// 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 创建 1MB 文件
	largeContent := bytes.Repeat([]byte("A"), 1024*1024)
	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "large.bin", largeContent)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/files", body)
	req.Header.Set("Content-Type", contentType)
	testServer.ServeHTTP(w, req)

	// 大文件上传应该成功
	assert.Equal(t, http.StatusOK, w.Code, "大文件上传应该成功")

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	fileData := resp["data"].(map[string]interface{})
	fileID := fileData["id"].(string)
	defer CleanupFile(t, fileID)
}

// TestFileAPI_GetInfo 测试获取文件信息
func TestFileAPI_GetInfo(t *testing.T) {
	// 1. 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 2. 上传文件
	fileContent := []byte("Test file content")
	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "info_test.txt", fileContent)

	uploadW := httptest.NewRecorder()
	uploadReq, _ := http.NewRequest("POST", "/api/files", body)
	uploadReq.Header.Set("Content-Type", contentType)
	testServer.ServeHTTP(uploadW, uploadReq)

	var uploadResp map[string]interface{}
	json.Unmarshal(uploadW.Body.Bytes(), &uploadResp)
	fileID := uploadResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupFile(t, fileID)

	// 3. 获取文件信息
	infoW := httptest.NewRecorder()
	infoReq, _ := http.NewRequest("GET", "/api/files/"+fileID, nil)
	testServer.ServeHTTP(infoW, infoReq)

	assert.Equal(t, http.StatusOK, infoW.Code)

	var infoResp map[string]interface{}
	json.Unmarshal(infoW.Body.Bytes(), &infoResp)
	infoData := infoResp["data"].(map[string]interface{})

	assert.Equal(t, fileID, infoData["id"])
	assert.Equal(t, "info_test.txt", infoData["filename"])
}

// TestFileAPI_Download 测试下载文件
func TestFileAPI_Download(t *testing.T) {
	// 1. 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 2. 上传文件
	fileContent := []byte("Download test content")
	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "download_test.txt", fileContent)

	uploadW := httptest.NewRecorder()
	uploadReq, _ := http.NewRequest("POST", "/api/files", body)
	uploadReq.Header.Set("Content-Type", contentType)
	testServer.ServeHTTP(uploadW, uploadReq)

	var uploadResp map[string]interface{}
	json.Unmarshal(uploadW.Body.Bytes(), &uploadResp)
	fileID := uploadResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupFile(t, fileID)

	// 3. 下载文件
	downloadW := httptest.NewRecorder()
	downloadReq, _ := http.NewRequest("GET", "/api/files/"+fileID+"/download", nil)
	testServer.ServeHTTP(downloadW, downloadReq)

	assert.Equal(t, http.StatusOK, downloadW.Code)

	// 验证 Content-Disposition header
	contentDisposition := downloadW.Header().Get("Content-Disposition")
	assert.Contains(t, contentDisposition, "attachment")
	assert.Contains(t, contentDisposition, "filename=download_test.txt")

	// 验证文件内容
	downloadedContent, _ := io.ReadAll(downloadW.Body)
	assert.Equal(t, fileContent, downloadedContent)
}

// TestFileAPI_GetInfo_NotFound 测试获取不存在的文件
func TestFileAPI_GetInfo_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/files/non-existent-id", nil)
	testServer.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFileAPI_GetSessionFiles 测试获取会话的文件列表
// 业务预期：上传到某 session 的文件，应该出现在该 session 的文件列表中。
// 已修复（Issue #1）：文件统一写入 files 表，GetSessionFiles 直接读 files 表。
func TestFileAPI_GetSessionFiles(t *testing.T) {
	// 1. 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	// 2. 上传两个文件
	fileIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		fileContent := []byte("Test content")
		body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "test.txt", fileContent)

		uploadW := httptest.NewRecorder()
		uploadReq, _ := http.NewRequest("POST", "/api/files", body)
		uploadReq.Header.Set("Content-Type", contentType)
		testServer.ServeHTTP(uploadW, uploadReq)
		assert.Equal(t, http.StatusOK, uploadW.Code)

		var uploadResp map[string]interface{}
		json.Unmarshal(uploadW.Body.Bytes(), &uploadResp)
		fileIDs[i] = uploadResp["data"].(map[string]interface{})["id"].(string)
		defer CleanupFile(t, fileIDs[i])
	}

	// 3. 获取会话的文件列表
	filesW := httptest.NewRecorder()
	filesReq, _ := http.NewRequest("GET", "/api/sessions/"+sessionID+"/files", nil)
	testServer.ServeHTTP(filesW, filesReq)

	assert.Equal(t, http.StatusOK, filesW.Code)

	var filesResp map[string]interface{}
	json.Unmarshal(filesW.Body.Bytes(), &filesResp)
	filesData, ok := filesResp["data"].([]interface{})
	assert.True(t, ok, "data should be an array of files")

	// 业务预期：上传的 2 个文件应该出现在会话文件列表里
	t.Logf("GetSessionFiles 返回文件数: %d（session=%s，预期>=2）", len(filesData), sessionID)
	assert.GreaterOrEqual(t, len(filesData), 2, "应该至少有2个文件（业务预期）")
}

// TestFileAPI_Upload_MultipleFormats 测试上传多种格式文件
func TestFileAPI_Upload_MultipleFormats(t *testing.T) {
	// 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)

	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	testCases := []struct {
		filename string
		content  []byte
		mimeType string
	}{
		{"test.txt", []byte("text content"), "text/plain"},
		{"test.json", []byte(`{"key": "value"}`), "application/json"},
		{"test.log", []byte("2024-01-01 INFO test"), "text/plain"},
	}

	for _, tc := range testCases {
		body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, tc.filename, tc.content)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/files", body)
		req.Header.Set("Content-Type", contentType)
		testServer.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, tc.filename+" 上传应该成功")

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		fileID := resp["data"].(map[string]interface{})["id"].(string)
		defer CleanupFile(t, fileID)
	}
}

// TestFileAPI_Upload_Concurrent 覆盖 Issue #6：并发上传同名文件应全部成功（S3 key 由 UUID 兜底）。
func TestFileAPI_Upload_Concurrent(t *testing.T) {
	// 1. 创建会话
	sessionW := httptest.NewRecorder()
	sessionReq, _ := http.NewRequest("POST", "/api/sessions", nil)
	testServer.ServeHTTP(sessionW, sessionReq)
	var sessionResp map[string]interface{}
	json.Unmarshal(sessionW.Body.Bytes(), &sessionResp)
	sessionID := sessionResp["data"].(map[string]interface{})["id"].(string)
	defer CleanupSession(t, sessionID)

	const n = 5
	fileIDs := make([]string, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			content := []byte(fmt.Sprintf("content-%d", i))
			body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "same.txt", content)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/files", body)
			req.Header.Set("Content-Type", contentType)
			testServer.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				errs[i] = fmt.Errorf("upload %d failed: %d", i, w.Code)
				return
			}
			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			fileIDs[i] = resp["data"].(map[string]interface{})["id"].(string)
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("并发上传 %d 失败: %v", i, errs[i])
		}
		if fileIDs[i] != "" {
			defer CleanupFile(t, fileIDs[i])
		}
	}
}

// Helper function to suppress unused variable warning
var _ = context.Background
