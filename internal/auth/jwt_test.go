package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourorg/llmgw/internal/domain"
)

func testUser() *domain.User {
	return &domain.User{ID: "u1", Email: "alice@example.com", Name: "Alice"}
}

// TestSignToken_ValidToken signs a token and verifies all claims.
func TestSignToken_ValidToken(t *testing.T) {
	user := testUser()
	tokenStr, err := SignToken(user, "secret", 1)
	if err != nil {
		t.Fatalf("SignToken error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse error: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("token should be valid")
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims should be MapClaims")
	}
	if claims["sub"] != user.ID {
		t.Errorf("sub = %v, want %q", claims["sub"], user.ID)
	}
	if claims["email"] != user.Email {
		t.Errorf("email = %v, want %q", claims["email"], user.Email)
	}
	if claims["name"] != user.Name {
		t.Errorf("name = %v, want %q", claims["name"], user.Name)
	}
	exp, ok := claims["exp"].(float64)
	if !ok || exp <= float64(time.Now().Unix()) {
		t.Errorf("exp claim should be in the future, got %v", claims["exp"])
	}
}

// TestSignToken_ClaimsMatchAllFields verifies that all three user fields are
// embedded exactly, including names/emails with special characters.
func TestSignToken_ClaimsMatchAllFields(t *testing.T) {
	user := &domain.User{ID: "usr-99", Email: "bob+tag@corp.io", Name: "Bob O'Brien"}
	tokenStr, _ := SignToken(user, "s", 24)

	parsed, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte("s"), nil
	})
	claims := parsed.Claims.(jwt.MapClaims)

	if claims["sub"] != "usr-99" {
		t.Errorf("sub = %v", claims["sub"])
	}
	if claims["email"] != "bob+tag@corp.io" {
		t.Errorf("email = %v", claims["email"])
	}
	if claims["name"] != "Bob O'Brien" {
		t.Errorf("name = %v", claims["name"])
	}
}

// TestSignToken_WrongSecretFails verifies that a token signed with one secret
// cannot be verified with a different secret.
func TestSignToken_WrongSecretFails(t *testing.T) {
	tokenStr, _ := SignToken(testUser(), "correct-secret", 1)

	_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})
	if err == nil {
		t.Error("expected error when verifying with wrong secret, got nil")
	}
}

// TestSignToken_ExpiredToken verifies that a 0-hour token is immediately expired.
func TestSignToken_ExpiredToken(t *testing.T) {
	// expireHours=0 → exp = time.Now() (already past by the time Parse runs)
	tokenStr, err := SignToken(testUser(), "secret", 0)
	if err != nil {
		t.Fatalf("SignToken error: %v", err)
	}

	_, err = jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

// TestSignToken_SigningMethodIsHS256 verifies the token uses HMAC-SHA256.
func TestSignToken_SigningMethodIsHS256(t *testing.T) {
	tokenStr, _ := SignToken(testUser(), "secret", 1)
	parsed, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if parsed.Method != jwt.SigningMethodHS256 {
		t.Errorf("signing method = %v, want HS256", parsed.Method)
	}
}

// TestSignToken_DifferentUsersProduceDifferentTokens ensures distinct output
// for distinct inputs (trivial sanity check against a static token).
func TestSignToken_DifferentUsersProduceDifferentTokens(t *testing.T) {
	t1, _ := SignToken(&domain.User{ID: "u1", Email: "a@b.com", Name: "A"}, "s", 1)
	t2, _ := SignToken(&domain.User{ID: "u2", Email: "c@d.com", Name: "C"}, "s", 1)
	if t1 == t2 {
		t.Error("different users should produce different tokens")
	}
}
