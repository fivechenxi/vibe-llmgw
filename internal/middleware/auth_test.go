package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const testSecret = "test-jwt-secret"

// signToken creates a signed HS256 JWT for testing.
func signToken(t *testing.T, sub string, secret string, exp time.Duration) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":   sub,
		"email": sub + "@test.com",
		"exp":   time.Now().Add(exp).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("signToken: %v", err)
	}
	return s
}

// newTestRouter wraps JWTAuth middleware + a simple echo handler that returns
// the userID stored in the context.
func newTestRouter(secret string) *gin.Engine {
	r := gin.New()
	r.GET("/test", JWTAuth(secret), func(c *gin.Context) {
		userID := c.GetString(UserIDKey)
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestJWTAuth_ValidToken_Passes(t *testing.T) {
	r := newTestRouter(testSecret)
	token := signToken(t, "alice", testSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestJWTAuth_ValidToken_SetsUserIDKey(t *testing.T) {
	r := gin.New()
	var capturedUserID interface{}
	r.GET("/test", JWTAuth(testSecret), func(c *gin.Context) {
		capturedUserID = c.MustGet(UserIDKey)
		c.Status(http.StatusOK)
	})

	token := signToken(t, "alice", testSecret, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(httptest.NewRecorder(), req)

	if capturedUserID != "alice" {
		t.Errorf("UserIDKey = %v, want alice", capturedUserID)
	}
}

func TestJWTAuth_MissingAuthorizationHeader(t *testing.T) {
	r := newTestRouter(testSecret)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuth_WrongPrefix(t *testing.T) {
	r := newTestRouter(testSecret)
	token := signToken(t, "alice", testSecret, time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Token "+token) // wrong prefix
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuth_InvalidTokenString(t *testing.T) {
	r := newTestRouter(testSecret)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer this.is.not.a.jwt")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	r := newTestRouter(testSecret)
	token := signToken(t, "alice", testSecret, -time.Second) // already expired

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for expired token", w.Code)
	}
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	r := newTestRouter(testSecret)
	token := signToken(t, "alice", "different-secret", time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for wrong-secret token", w.Code)
	}
}

func TestJWTAuth_AbortsPipeline(t *testing.T) {
	// Verify that JWTAuth calls Abort — the downstream handler must NOT run
	// when the token is invalid.
	downstreamCalled := false
	r := gin.New()
	r.GET("/test", JWTAuth(testSecret), func(c *gin.Context) {
		downstreamCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header
	r.ServeHTTP(httptest.NewRecorder(), req)

	if downstreamCalled {
		t.Error("downstream handler should NOT be called when auth fails")
	}
}

func TestJWTAuth_UserIDKey_IsConstant(t *testing.T) {
	// Ensure UserIDKey is the exported string used by other packages.
	if UserIDKey != "userID" {
		t.Errorf("UserIDKey = %q, want 'userID'", UserIDKey)
	}
}
