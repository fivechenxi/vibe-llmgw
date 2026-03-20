package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourorg/llmgw/internal/domain"
)

func SignToken(user *domain.User, secret string, expireHours int) (string, error) {
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"exp":   time.Now().Add(time.Duration(expireHours) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}