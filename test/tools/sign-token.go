//go:build ignore

// JWT Token Signing Tool for Manual Testing
// Usage: go run sign-token.go <user_id> [email] [name]
// Example: go run sign-token.go alice alice@test.com "Alice Test"
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jwtSecret = "test-jwt-secret-for-blackbox-testing"

func main() {
	userID := "alice"
	email := "alice@test.com"
	name := "Alice Test"

	if len(os.Args) > 1 {
		userID = os.Args[1]
	}
	if len(os.Args) > 2 {
		email = os.Args[2]
	}
	if len(os.Args) > 3 {
		name = os.Args[3]
	}

	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"name":  name,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error signing token: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Email:   %s\n", email)
	fmt.Printf("Name:    %s\n", name)
	fmt.Printf("Expires: %s\n", time.Now().Add(24*time.Hour).Format(time.RFC3339))
	fmt.Printf("\nJWT Token:\n%s\n", tokenString)
	fmt.Printf("\nAuthorization header:\nAuthorization: Bearer %s\n", tokenString)
}