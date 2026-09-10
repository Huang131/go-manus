package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
)

func setupTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestRecovery_PanicReturns500(t *testing.T) {
	engine := setupTestEngine()
	engine.Use(Recovery())
	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Recovery() status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRecovery_NoPanicPassesThrough(t *testing.T) {
	engine := setupTestEngine()
	engine.Use(Recovery())
	engine.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Recovery() passthrough status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLogger_PassesThrough(t *testing.T) {
	engine := setupTestEngine()
	engine.Use(Logger())
	engine.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Logger() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCORS_PreflightOptions(t *testing.T) {
	engine := setupTestEngine()
	engine.Use(CORS())
	engine.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodOptions, "/data", nil)
	// Origin 必须与请求 Host 不同，否则 cors 视为同源请求直接放行
	req.Header.Set("Origin", "http://frontend.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("CORS() preflight status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("CORS() Access-Control-Allow-Origin = %q, want *", origin)
	}
}

func TestCORS_AllowedHeader(t *testing.T) {
	engine := setupTestEngine()
	engine.Use(CORS())
	engine.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodOptions, "/data", nil)
	req.Header.Set("Origin", "http://frontend.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Last-Event-ID")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("CORS() Last-Event-ID preflight status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestCORS_SameOriginNotCorsRequest(t *testing.T) {
	// Origin 与 Host 相同 → cors 视为同源请求，不做任何处理，正常路由
	engine := setupTestEngine()
	engine.Use(CORS())
	engine.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Origin", "http://example.com") // httptest.NewRequest 默认 Host 即 example.com
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CORS() same-origin status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequestID_PropagatesIncomingHeader(t *testing.T) {
	engine := setupTestEngine()
	engine.Use(RequestID())

	var gotRequestID string
	engine.GET("/echo", func(c *gin.Context) {
		gotRequestID = c.GetString("request_id")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	req.Header.Set("X-Request-ID", "client-supplied-id")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if gotRequestID != "client-supplied-id" {
		t.Errorf("RequestID() context value = %q, want client-supplied-id", gotRequestID)
	}
	if hdr := w.Header().Get("X-Request-ID"); hdr != "client-supplied-id" {
		t.Errorf("RequestID() response header = %q, want client-supplied-id", hdr)
	}
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	engine := setupTestEngine()
	engine.Use(RequestID())

	var gotRequestID string
	engine.GET("/echo", func(c *gin.Context) {
		gotRequestID = c.GetString("request_id")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/echo", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if gotRequestID == "" {
		t.Error("RequestID() context value is empty, want generated id")
	}
	if hdr := w.Header().Get("X-Request-ID"); hdr == "" {
		t.Error("RequestID() response header is empty, want generated id")
	}
	if hdr := w.Header().Get("X-Request-ID"); hdr != gotRequestID {
		t.Errorf("RequestID() response header = %q, want %q (same as context)", hdr, gotRequestID)
	}
}

func TestGenerateRequestID_NonEmpty(t *testing.T) {
	id := generateRequestID()
	if id == "" {
		t.Error("generateRequestID() returned empty string")
	}
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("generateRequestID() = %q, want UUID: %v", id, err)
	}
}
