package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/domain"
	"github.com/yourorg/llmgw/internal/middleware"
	"github.com/yourorg/llmgw/internal/proxy/providers"
	"github.com/yourorg/llmgw/internal/quota"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- test doubles ----

type stubQuota struct {
	checkErr error
	deducted int
}

func (s *stubQuota) Check(_ context.Context, _, _ string) error { return s.checkErr }
func (s *stubQuota) Deduct(_ context.Context, _, _ string, tokens int) error {
	s.deducted += tokens
	return nil
}

type stubChatSaver struct {
	saved []*domain.ChatLog
}

func (s *stubChatSaver) Save(_ context.Context, l *domain.ChatLog) error {
	s.saved = append(s.saved, l)
	return nil
}

// stubRouter wraps a single provider for all models.
type stubRouter struct {
	provider Provider
	missing  bool
}

func (s *stubRouter) Get(_ string) (Provider, error) {
	if s.missing {
		return nil, errors.New("no provider")
	}
	return s.provider, nil
}

// newTestEngine returns a gin engine with the handler wired up and userID pre-set.
func newTestEngine(h *Handler) *gin.Engine {
	r := gin.New()
	r.POST("/api/chat", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, "test-user")
		h.Chat(c)
	})
	return r
}

func postChat(engine *gin.Engine, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// ---- tests ----

func TestHandlerChat_Complete(t *testing.T) {
	mock := &providers.MockProvider{Response: "hello from mock"}
	q := &stubQuota{}
	saver := &stubChatSaver{}

	h := newHandlerWithRouter(q, saver, &stubRouter{provider: mock})
	w := postChat(newTestEngine(h), map[string]interface{}{
		"model":    "mock",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
		"stream":   false,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.ChatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Content != "hello from mock" {
		t.Errorf("unexpected content: %q", resp.Content)
	}

	// Deduct is async — give the goroutine a moment
	time.Sleep(50 * time.Millisecond)
	if q.deducted == 0 {
		t.Error("expected quota to be deducted")
	}
}

func TestHandlerChat_BadRequest_MissingModel(t *testing.T) {
	h := newHandlerWithRouter(&stubQuota{}, &stubChatSaver{}, &stubRouter{})
	w := postChat(newTestEngine(h), map[string]interface{}{
		// "model" intentionally omitted
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerChat_QuotaExceeded(t *testing.T) {
	q := &stubQuota{checkErr: quota.ErrQuotaExceeded}
	h := newHandlerWithRouter(q, &stubChatSaver{}, &stubRouter{provider: providers.NewMockProvider()})
	w := postChat(newTestEngine(h), map[string]interface{}{
		"model":    "mock",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestHandlerChat_QuotaServiceError(t *testing.T) {
	q := &stubQuota{checkErr: errors.New("db down")}
	h := newHandlerWithRouter(q, &stubChatSaver{}, &stubRouter{provider: providers.NewMockProvider()})
	w := postChat(newTestEngine(h), map[string]interface{}{
		"model":    "mock",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestHandlerChat_UnsupportedModel(t *testing.T) {
	h := newHandlerWithRouter(&stubQuota{}, &stubChatSaver{}, &stubRouter{missing: true})
	w := postChat(newTestEngine(h), map[string]interface{}{
		"model":    "no-such-model",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerChat_Stream(t *testing.T) {
	mock := &providers.MockProvider{Response: "streamed response"}
	h := newHandlerWithRouter(&stubQuota{}, &stubChatSaver{}, &stubRouter{provider: mock})

	body, _ := json.Marshal(map[string]interface{}{
		"model":    "mock",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
		"stream":   true,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newTestEngine(h).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("expected SSE content-type, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "streamed") {
		t.Errorf("expected streamed content in body, got: %s", w.Body.String())
	}
}
