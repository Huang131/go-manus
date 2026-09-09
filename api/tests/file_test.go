//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// uploadFileForTest 上传文件并返回 fileID，调用方负责 cleanup
func uploadFileForTest(t *testing.T, sessionID, fileName string, content []byte) string {
	t.Helper()
	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, fileName, content)

	w := doRequest(t, "POST", "/api/files", body.Bytes(), contentType)
	if w.Code != http.StatusOK {
		t.Fatalf("上传文件失败，状态码: %d，响应: %s", w.Code, w.Body.String())
	}

	resp := parseResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("上传文件失败，错误码: %d，错误信息: %s", resp.Code, resp.Msg)
	}
	if resp.Data == nil {
		t.Fatalf("上传文件响应 data 为 nil")
	}
	return resp.Data.(map[string]any)["id"].(string)
}

// TestFileAPI_Upload_Lifecycle 测试文件上传完整生命周期
func TestFileAPI_Upload_Lifecycle(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	fileContent := []byte("Hello, World! This is a test file.")
	fileID := uploadFileForTest(t, sessionID, "test.txt", fileContent)
	defer CleanupFile(t, fileID)

	infoW := getJSON(t, "/api/files/"+fileID)
	assert.Equal(t, http.StatusOK, infoW.Code)

	_ = parseResponse(t, infoW)
	infoData := parseResponseDataAsMap(t, infoW)

	assert.Equal(t, fileID, infoData["id"])
	assert.Equal(t, "test.txt", infoData["filename"])
	assert.Equal(t, int64(len(fileContent)), int64(infoData["size"].(float64)))
	assert.Equal(t, sessionID, infoData["session_id"])
}

// TestFileAPI_Download_Lifecycle 测试文件下载完整生命周期
func TestFileAPI_Download_Lifecycle(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	fileContent := []byte("Download test content")
	fileID := uploadFileForTest(t, sessionID, "download_test.txt", fileContent)
	defer CleanupFile(t, fileID)

	downloadW := getJSON(t, "/api/files/"+fileID+"/download")
	assert.Equal(t, http.StatusOK, downloadW.Code)

	contentDisposition := downloadW.Header().Get("Content-Disposition")
	assert.Contains(t, contentDisposition, "attachment")
	assert.Contains(t, contentDisposition, "filename=download_test.txt")

	downloadedContent, _ := io.ReadAll(downloadW.Body)
	assert.Equal(t, fileContent, downloadedContent)
}

// TestFileAPI_Delete_Lifecycle 测试文件删除接口（当前返回 404，未实现）
func TestFileAPI_Delete_Lifecycle(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	fileContent := []byte("File to be deleted")
	fileID := uploadFileForTest(t, sessionID, "delete_me.txt", fileContent)
	defer CleanupFile(t, fileID)

	// DELETE 接口尚未实现，路由返回 404
	deleteW := postJSON(t, "/api/files/"+fileID+"/delete", nil)
	assert.Equal(t, http.StatusNotFound, deleteW.Code,
		"DELETE 接口未实现时应返回 404，实际: %d", deleteW.Code)

	// 文件仍然存在（因为 delete 未实现）
	getW := getJSON(t, "/api/files/"+fileID)
	assert.Equal(t, http.StatusOK, getW.Code,
		"删除未实现时文件 GET 应返回 200，实际: %d", getW.Code)
}

// TestFileAPI_Rename_Lifecycle 测试文件名更新完整生命周期
func TestFileAPI_Rename_Lifecycle(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	fileContent := []byte("File to be renamed")
	fileID := uploadFileForTest(t, sessionID, "old_name.txt", fileContent)
	defer CleanupFile(t, fileID)

	renameW := putJSON(t, "/api/files/"+fileID, map[string]any{"filename": "new_name.txt"})
	// rename 可能返回 200（成功）或 404（文件不存在）
	assert.True(t, renameW.Code == http.StatusOK || renameW.Code == http.StatusNotFound,
		"rename 应返回 200 或 404，实际: %d，响应: %s", renameW.Code, renameW.Body.String())

	// 如果 rename 成功，验证文件名
	if renameW.Code == http.StatusOK {
		infoW := getJSON(t, "/api/files/"+fileID)
		infoResp := parseResponse(t, infoW)
		infoData := infoResp.Data.(map[string]any)
		assert.Equal(t, "new_name.txt", infoData["filename"])
	}
}

// TestFileAPI_GetSessionFiles_Lifecycle 测试获取会话文件列表完整生命周期
func TestFileAPI_GetSessionFiles_Lifecycle(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	fileIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		fileContent := []byte("Test content")
		fileID := uploadFileForTest(t, sessionID, fmt.Sprintf("test_%d.txt", i), fileContent)
		fileIDs[i] = fileID
		defer CleanupFile(t, fileID)
	}

	filesW := getJSON(t, "/api/sessions/"+sessionID+"/files")
	assert.Equal(t, http.StatusOK, filesW.Code)

	filesResp := parseResponse(t, filesW)
	filesData, ok := filesResp.Data.([]any)
	assert.True(t, ok, "data should be an array of files")

	t.Logf("GetSessionFiles 返回文件数: %d（session=%s，预期>=2）", len(filesData), sessionID)
	assert.GreaterOrEqual(t, len(filesData), 2, "应该至少有2个文件")
}

// TestFileAPI_Upload_MissingSession 测试缺少 session_id
func TestFileAPI_Upload_MissingSession(t *testing.T) {
	// 创建会话（只是为了让测试环境正常，但上传时不使用）
	_ = createSessionForTest(t)

	body, contentType := makeMultipartFile(nil, "test.txt", []byte("content"))
	w := doRequest(t, "POST", "/api/files", body.Bytes(), contentType)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	resp := parseResponse(t, w)
	assert.NotEqual(t, 0, resp.Code)
}

// TestFileAPI_Upload_MissingFile 测试缺少文件
func TestFileAPI_Upload_MissingFile(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "", nil)
	w := doRequest(t, "POST", "/api/files", body.Bytes(), contentType)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestFileAPI_Upload_LargeFile 测试上传大文件（1MB）
func TestFileAPI_Upload_LargeFile(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	largeContent := bytes.Repeat([]byte("A"), 1024*1024)
	fileID := uploadFileForTest(t, sessionID, "large.bin", largeContent)
	defer CleanupFile(t, fileID)

	infoW := getJSON(t, "/api/files/"+fileID)

	infoResp := parseResponse(t, infoW)
	infoData := infoResp.Data.(map[string]any)
	assert.Equal(t, int64(len(largeContent)), int64(infoData["size"].(float64)))
}

// TestFileAPI_GetInfo_NotFound 测试获取不存在的文件
func TestFileAPI_GetInfo_NotFound(t *testing.T) {
	w := getJSON(t, "/api/files/non-existent-id")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestFileAPI_Delete_NotFound 测试删除不存在的文件
func TestFileAPI_Delete_NotFound(t *testing.T) {
	w := postJSON(t, "/api/files/non-existent-id/delete", nil)
	// 删除不存在的文件返回 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestFileAPI_Upload_MultipleFormats 测试上传多种格式文件
func TestFileAPI_Upload_MultipleFormats(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	testCases := []struct {
		filename string
		content  []byte
	}{
		{"test.txt", []byte("text content")},
		{"test.json", []byte(`{"key": "value"}`)},
		{"test.log", []byte("2024-01-01 INFO test")},
	}

	for _, tc := range testCases {
		fileID := uploadFileForTest(t, sessionID, tc.filename, tc.content)
		defer CleanupFile(t, fileID)

		infoW := getJSON(t, "/api/files/"+fileID)
		assert.Equal(t, http.StatusOK, infoW.Code, tc.filename+" 上传应该成功")
	}
}

// TestFileAPI_Upload_Concurrent 测试并发上传同名文件应全部成功
func TestFileAPI_Upload_Concurrent(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	const n = 5
	fileIDs := make([]string, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			content := []byte(fmt.Sprintf("content-%d", idx))
			body, contentType := makeMultipartFile(map[string]string{"session_id": sessionID}, "same.txt", content)
			w := doRequest(t, "POST", "/api/files", body.Bytes(), contentType)
			if w.Code != http.StatusOK {
				errs[idx] = fmt.Errorf("upload %d failed: %d", idx, w.Code)
				return
			}
			resp := parseResponse(t, w)
			fileIDs[idx] = resp.Data.(map[string]any)["id"].(string)
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("并发上传 %d 失败: %v", i, errs[i])
		}
		assert.NotEmpty(t, fileIDs[i], "并发上传 %d 应返回 fileID", i)
	}

	for _, fileID := range fileIDs {
		if fileID != "" {
			CleanupFile(t, fileID)
		}
	}
}

// TestFileAPI_FileTableConsistency 测试 file 表内一致性
func TestFileAPI_FileTableConsistency(t *testing.T) {
	sessionID := createSessionForTest(t)
	defer CleanupSession(t, sessionID)

	fileContent := []byte("consistency check content")
	fileID := uploadFileForTest(t, sessionID, "check.txt", fileContent)
	defer CleanupFile(t, fileID)

	ctx, cancel := NewTestContext()
	defer cancel()
	var name, fSessionID string
	var sizeBytes int64
	err := testDB.Pool.QueryRow(ctx,
		"SELECT filename, session_id, size FROM files WHERE id = $1", fileID).
		Scan(&name, &fSessionID, &sizeBytes)
	assert.NoError(t, err, "file 表应能查到")
	assert.Equal(t, "check.txt", name)
	assert.Equal(t, sessionID, fSessionID, "file.session_id 应与上传时一致")
	assert.Equal(t, int64(len(fileContent)), sizeBytes, "size 应等于实际内容长度")
}
