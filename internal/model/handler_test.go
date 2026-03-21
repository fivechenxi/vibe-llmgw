package model

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/domain"
	"github.com/yourorg/llmgw/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- stub quota lister ----

type stubQuotaLister struct {
	quotas []domain.UserQuota
	err    error
}

func (s *stubQuotaLister) ListByUser(_ context.Context, _ string) ([]domain.UserQuota, error) {
	return s.quotas, s.err
}

var _ quotaLister = (*stubQuotaLister)(nil)

// newHandlerWithStub bypasses NewHandler's concrete *quota.Repository parameter.
func newHandlerWithStub(ql quotaLister) *Handler {
	return &Handler{repo: nil, quotaRepo: ql}
}

func ginCtx(t *testing.T, userID string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(middleware.UserIDKey, userID)
	return c, w
}

// ---- ListModels tests ----

func TestModelHandler_ListModels_ReturnsModelsWithRemainingQuota(t *testing.T) {
	stub := &stubQuotaLister{quotas: []domain.UserQuota{
		{ModelID: "gpt-4o", QuotaTokens: 1000, UsedTokens: 400},       // remaining=600 → included
		{ModelID: "claude-haiku-4-5", QuotaTokens: 200, UsedTokens: 200}, // remaining=0 → excluded
		{ModelID: "deepseek-chat", QuotaTokens: 500, UsedTokens: 100},  // remaining=400 → included
	}}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListModels(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body struct {
		Models []struct {
			ModelID   string `json:"model_id"`
			Remaining int64  `json:"remaining_tokens"`
		} `json:"models"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Models) != 2 {
		t.Errorf("models count = %d, want 2 (zero-remaining excluded)", len(body.Models))
	}
	for _, m := range body.Models {
		if m.ModelID == "claude-haiku-4-5" {
			t.Error("claude-haiku-4-5 has 0 remaining and should be excluded")
		}
		if m.Remaining <= 0 {
			t.Errorf("model %q has remaining=%d, should be > 0", m.ModelID, m.Remaining)
		}
	}
}

func TestModelHandler_ListModels_AllExhausted_ReturnsEmpty(t *testing.T) {
	stub := &stubQuotaLister{quotas: []domain.UserQuota{
		{ModelID: "gpt-4o", QuotaTokens: 100, UsedTokens: 100},
	}}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListModels(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body struct {
		Models []interface{} `json:"models"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if len(body.Models) != 0 {
		t.Errorf("expected empty models list, got %d", len(body.Models))
	}
}

func TestModelHandler_ListModels_RemainingCorrect(t *testing.T) {
	stub := &stubQuotaLister{quotas: []domain.UserQuota{
		{ModelID: "gpt-4o", QuotaTokens: 1000, UsedTokens: 300},
	}}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListModels(c)

	var body struct {
		Models []struct {
			ModelID   string `json:"model_id"`
			Remaining int64  `json:"remaining_tokens"`
		} `json:"models"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if len(body.Models) != 1 || body.Models[0].Remaining != 700 {
		t.Errorf("expected remaining=700, got %+v", body.Models)
	}
}

func TestModelHandler_ListModels_RepoError(t *testing.T) {
	stub := &stubQuotaLister{err: errors.New("db down")}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListModels(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestModelHandler_ListModels_NegativeRemaining_Excluded(t *testing.T) {
	// Over-deducted quota: remaining < 0 should be excluded same as = 0.
	stub := &stubQuotaLister{quotas: []domain.UserQuota{
		{ModelID: "gpt-4o", QuotaTokens: 100, UsedTokens: 200}, // remaining=-100
	}}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListModels(c)

	var body struct {
		Models []interface{} `json:"models"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	if len(body.Models) != 0 {
		t.Errorf("over-deducted model should be excluded, got %d models", len(body.Models))
	}
}

// ---- ListQuota tests ----

func TestModelHandler_ListQuota_ReturnsAll(t *testing.T) {
	quotas := []domain.UserQuota{
		{ModelID: "gpt-4o", QuotaTokens: 1000, UsedTokens: 400},
		{ModelID: "claude-haiku-4-5", QuotaTokens: 200, UsedTokens: 200},
	}
	stub := &stubQuotaLister{quotas: quotas}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListQuota(c)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var body struct {
		Quotas []domain.UserQuota `json:"quotas"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// ListQuota returns ALL quotas regardless of remaining balance.
	if len(body.Quotas) != 2 {
		t.Errorf("quotas count = %d, want 2", len(body.Quotas))
	}
}

func TestModelHandler_ListQuota_IncludesExhausted(t *testing.T) {
	stub := &stubQuotaLister{quotas: []domain.UserQuota{
		{ModelID: "gpt-4o", QuotaTokens: 100, UsedTokens: 100}, // remaining=0
	}}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListQuota(c)

	var body struct {
		Quotas []domain.UserQuota `json:"quotas"`
	}
	_ = json.NewDecoder(w.Body).Decode(&body)
	// Unlike ListModels, ListQuota includes exhausted quotas.
	if len(body.Quotas) != 1 {
		t.Errorf("ListQuota should include exhausted quotas; got %d", len(body.Quotas))
	}
}

func TestModelHandler_ListQuota_RepoError(t *testing.T) {
	stub := &stubQuotaLister{err: errors.New("db error")}
	h := newHandlerWithStub(stub)
	c, w := ginCtx(t, "alice")

	h.ListQuota(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
