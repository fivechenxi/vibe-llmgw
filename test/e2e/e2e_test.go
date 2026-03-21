// Package e2e provides end-to-end tests against a running LLM Gateway server.
//
// These tests require a running test environment (PostgreSQL + LLM Gateway).
// Use test/docker/setup-test-env.sh to start the environment.
//
// Environment variables:
//   - TEST_API_BASE: Base URL of the LLM Gateway (default: http://localhost:8080)
//   - TEST_JWT_SECRET: JWT secret for signing test tokens
//   - TEST_DATABASE_URL: PostgreSQL connection string for cleanup
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	apiBase       string
	jwtSecret     string
	dbURL         string
	httpClient    *http.Client
	noRedirect    *http.Client
)

func init() {
	apiBase = getEnv("TEST_API_BASE", "http://localhost:8080")
	jwtSecret = getEnv("TEST_JWT_SECRET", "test-jwt-secret-for-blackbox-testing")
	dbURL = os.Getenv("TEST_DATABASE_URL")
	httpClient = &http.Client{Timeout: 30 * time.Second}
	// Client that doesn't follow redirects for testing auth endpoints
	noRedirect = &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// ==============================================================================
// Test Helpers
// ==============================================================================

func signToken(t *testing.T, userID, email, name string, exp time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"name":  name,
		"exp":   time.Now().Add(exp).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("signToken: %v", err)
	}
	return s
}

func request(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	t.Helper()
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, apiBase+path, reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func get(t *testing.T, path, token string) *http.Response {
	return request(t, http.MethodGet, path, nil, token)
}

func post(t *testing.T, path string, body interface{}, token string) *http.Response {
	return request(t, http.MethodPost, path, body, token)
}

func parseJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	return result
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// skipIfNoEnv skips the test if the required environment is not available
func skipIfNoEnv(t *testing.T) {
	t.Helper()
	resp, err := http.Get(apiBase + "/auth/login")
	if err != nil {
		t.Skipf("E2E test environment not available: %v", err)
	}
	resp.Body.Close()
}

// ==============================================================================
// E2E Tests
// ==============================================================================

func TestE2E_HealthCheck(t *testing.T) {
	skipIfNoEnv(t)

	// Use noRedirect client to not follow redirects
	req, _ := http.NewRequest(http.MethodGet, apiBase+"/auth/login", nil)
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Auth login should redirect (302) to SSO
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
}

func TestE2E_Auth_Login(t *testing.T) {
	skipIfNoEnv(t)

	req, _ := http.NewRequest(http.MethodGet, apiBase+"/auth/login", nil)
	resp, err := noRedirect.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		t.Error("expected Location header")
	}
	t.Logf("Login redirects to: %s", location)
}

func TestE2E_Auth_Logout(t *testing.T) {
	skipIfNoEnv(t)

	resp := post(t, "/auth/logout", nil, "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestE2E_JWT_MissingToken(t *testing.T) {
	skipIfNoEnv(t)

	resp := get(t, "/api/models", "")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestE2E_JWT_ExpiredToken(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "alice", "alice@test.com", "Alice", -time.Hour)
	resp := get(t, "/api/models", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestE2E_Models_ListModels(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	resp := get(t, "/api/models", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	result := parseJSON(t, resp)
	models := result["models"].([]interface{})
	if len(models) == 0 {
		t.Error("expected at least one model with quota")
	}

	// Verify remaining_tokens > 0 for all returned models
	for _, m := range models {
		model := m.(map[string]interface{})
		remaining := int64(model["remaining_tokens"].(float64))
		if remaining <= 0 {
			t.Errorf("model %s should have positive remaining_tokens", model["model_id"])
		}
	}
	t.Logf("Found %d models with remaining quota", len(models))
}

func TestE2E_Models_ListModels_EmptyForNoQuota(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "charlie", "charlie@test.com", "Charlie", time.Hour)
	resp := get(t, "/api/models", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp)
	models := result["models"].([]interface{})
	if len(models) != 0 {
		t.Errorf("expected empty models for user with no quota, got %d", len(models))
	}
}

func TestE2E_Quota_List(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	resp := get(t, "/api/quota", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp)
	quotas, ok := result["quotas"].([]interface{})
	if !ok {
		t.Fatalf("expected quotas array, got %T", result["quotas"])
	}
	if len(quotas) == 0 {
		t.Error("expected at least one quota record")
	}
	t.Logf("Found %d quota records", len(quotas))
}

func TestE2E_Chat_Complete(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	resp := post(t, "/api/chat", map[string]interface{}{
		"model":    "mock",
		"messages": []map[string]string{{"role": "user", "content": "Hello from E2E test"}},
		"stream":   false,
	}, token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	result := parseJSON(t, resp)
	content := result["content"].(string)
	if content == "" {
		t.Error("expected non-empty content")
	}
	usage := result["usage"].(map[string]interface{})
	if usage["total_tokens"].(float64) == 0 {
		t.Error("expected non-zero token usage")
	}
	t.Logf("Chat response: %s (tokens: %.0f)", content[:min(50, len(content))], usage["total_tokens"].(float64))
}

func TestE2E_Chat_QuotaExceeded(t *testing.T) {
	skipIfNoEnv(t)

	// Bob has exhausted quota
	token := signToken(t, "bob", "bob@test.com", "Bob", time.Hour)
	resp := post(t, "/api/chat", map[string]interface{}{
		"model":    "mock",
		"messages": []map[string]string{{"role": "user", "content": "test"}},
	}, token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestE2E_Chat_Stream(t *testing.T) {
	skipIfNoEnv(t)

	token := signTestToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	req, _ := http.NewRequest(http.MethodPost, apiBase+"/api/chat", bytes.NewReader([]byte(`{
		"model": "mock",
		"messages": [{"role": "user", "content": "Stream test"}],
		"stream": true
	}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %s", contentType)
	}

	// Read SSE stream
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "data:") {
		t.Errorf("expected SSE data format, got: %s", string(body)[:min(200, len(body))])
	}
	t.Logf("SSE response received, length: %d bytes", len(body))
}

func signTestToken(t *testing.T, userID, email, name string, exp time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"name":  name,
		"exp":   time.Now().Add(exp).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		t.Fatalf("signToken: %v", err)
	}
	return s
}

func TestE2E_Sessions_List(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	resp := get(t, "/api/sessions", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	result := parseJSON(t, resp)
	sessions := result["sessions"].([]interface{})
	t.Logf("Found %d sessions", len(sessions))
}

func TestE2E_Sessions_Get(t *testing.T) {
	skipIfNoEnv(t)

	// Use a known session ID from seed data
	sessionID := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	token := signToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	resp := get(t, "/api/sessions/"+sessionID, token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, readBody(t, resp))
		return
	}

	result := parseJSON(t, resp)
	messages, ok := result["messages"].([]interface{})
	if !ok {
		t.Fatalf("expected messages array, got %T", result["messages"])
	}
	if len(messages) == 0 {
		t.Error("expected messages in session")
	}
	t.Logf("Session %s has %d messages", sessionID, len(messages))
}

func TestE2E_Sessions_InvalidUUID(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	resp := get(t, "/api/sessions/invalid-uuid", token)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestE2E_Chat_SessionSticky(t *testing.T) {
	skipIfNoEnv(t)

	token := signToken(t, "alice", "alice@test.com", "Alice", time.Hour)
	sessionID := uuid.New().String()

	// Make two requests with the same session_id
	for i := 0; i < 2; i++ {
		resp := post(t, "/api/chat", map[string]interface{}{
			"model":      "mock",
			"messages":   []map[string]string{{"role": "user", "content": fmt.Sprintf("Request %d", i+1)}},
			"session_id": sessionID,
		}, token)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, resp.StatusCode)
		}
	}

	// Verify session appears in session list
	time.Sleep(100 * time.Millisecond) // Wait for async save
	resp := get(t, "/api/sessions", token)
	result := parseJSON(t, resp)
	resp.Body.Close()

	sessions := result["sessions"].([]interface{})
	found := false
	for _, s := range sessions {
		if s.(string) == sessionID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("session %s not found in session list", sessionID)
	}
}

// Database verification test (optional, requires TEST_DATABASE_URL)
func TestE2E_Database_HasTestData(t *testing.T) {
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	// Check users
	var userCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		t.Error("no test users found in database")
	}
	t.Logf("Database has %d users", userCount)

	// Check quotas
	var quotaCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_quotas").Scan(&quotaCount)
	if quotaCount == 0 {
		t.Error("no test quotas found in database")
	}
	t.Logf("Database has %d quotas", quotaCount)

	// Check credentials
	var credCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM model_credentials WHERE is_active = true").Scan(&credCount)
	if credCount == 0 {
		t.Error("no active credentials found in database")
	}
	t.Logf("Database has %d active credentials", credCount)
}