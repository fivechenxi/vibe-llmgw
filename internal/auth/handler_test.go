package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/llmgw/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestHandler() *Handler {
	return NewHandler(&config.Config{}, nil) // db not used by current stubs
}

func TestAuthHandler_Login_Redirects(t *testing.T) {
	h := newTestHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/login", nil)

	h.Login(c)

	if w.Code != http.StatusFound {
		t.Errorf("Login status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc == "" {
		t.Error("Login should set Location header for redirect")
	}
}

func TestAuthHandler_Callback_ReturnsTODO(t *testing.T) {
	h := newTestHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/callback?code=xyz", nil)

	h.Callback(c)

	if w.Code != http.StatusOK {
		t.Errorf("Callback status = %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := body["token"]; !ok {
		t.Error("Callback response should contain 'token' key")
	}
}

func TestAuthHandler_Logout_OK(t *testing.T) {
	h := newTestHandler()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/logout", nil)

	h.Logout(c)

	if w.Code != http.StatusOK {
		t.Errorf("Logout status = %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["message"] != "logged out" {
		t.Errorf("Logout message = %q, want 'logged out'", body["message"])
	}
}

// TestAuthHandler_SSOProvider_Interface verifies the SSOProvider interface
// compiles as expected — any struct implementing AuthURL+ExchangeUser satisfies it.
func TestAuthHandler_SSOProvider_Interface(t *testing.T) {
	var _ SSOProvider = (*mockSSO)(nil)
}

type mockSSO struct{}

func (m *mockSSO) AuthURL(state string) string { return "https://sso.example.com?state=" + state }
func (m *mockSSO) ExchangeUser(code string) (*UserInfo, error) {
	return &UserInfo{ID: "u1", Email: "u@e.com", Name: "U"}, nil
}
