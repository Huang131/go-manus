package response

import (
	"github.com/bytedance/sonic"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestResponse_Structure(t *testing.T) {
	resp := Response{
		Code: 200,
		Msg:  "success",
		Data: map[string]string{"key": "value"},
	}

	data, err := sonic.Marshal(resp)
	if err != nil {
		t.Errorf("Response marshal error: %v", err)
	}

	var parsed Response
	if err := sonic.Unmarshal(data, &parsed); err != nil {
		t.Errorf("Response unmarshal error: %v", err)
	}

	if parsed.Code != 200 {
		t.Errorf("Code = %d, want 200", parsed.Code)
	}
	if parsed.Msg != "success" {
		t.Errorf("Msg = %s, want success", parsed.Msg)
	}
}

func TestTotalResponse_Structure(t *testing.T) {
	resp := TotalResponse{
		Code:  0,
		Msg:   "success",
		Data:  []string{"a", "b"},
		Total: 2,
	}

	data, err := sonic.Marshal(resp)
	if err != nil {
		t.Errorf("TotalResponse marshal error: %v", err)
	}

	var parsed TotalResponse
	if err := sonic.Unmarshal(data, &parsed); err != nil {
		t.Errorf("TotalResponse unmarshal error: %v", err)
	}

	if parsed.Total != 2 {
		t.Errorf("Total = %d, want 2", parsed.Total)
	}
}

func TestSuccess_HttpResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"key": "value"}
	Success(c, data)

	if w.Code != http.StatusOK {
		t.Errorf("Success() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	if err := sonic.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Success() response is not valid JSON: %v", err)
	}

	if resp.Code != 0 {
		t.Errorf("Success() code = %d, want 0", resp.Code)
	}
}

func TestError_HttpResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, "bad request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Error() status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp Response
	if err := sonic.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Error() response is not valid JSON: %v", err)
	}

	if resp.Code != 400 {
		t.Errorf("Error() code = %d, want 400", resp.Code)
	}
	if resp.Msg != "bad request" {
		t.Errorf("Error() msg = %s, want bad request", resp.Msg)
	}
}

func TestFromError_MappedBusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FromError(c, apperr.NotFound("模型不存在"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("FromError() status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp Response
	if err := sonic.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("FromError() response invalid JSON: %v", err)
	}
	if resp.Code != http.StatusNotFound {
		t.Fatalf("FromError() code = %d, want %d", resp.Code, http.StatusNotFound)
	}
	if resp.Msg != "模型不存在" {
		t.Fatalf("FromError() msg = %q, want %q", resp.Msg, "模型不存在")
	}
}

func TestFromError_GenericError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FromError(c, http.ErrServerClosed)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("FromError() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
