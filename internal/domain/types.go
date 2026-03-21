package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        string    `db:"id"`
	Email     string    `db:"email"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

type Model struct {
	ID       string `db:"id"`
	Name     string `db:"name"`
	Provider string `db:"provider"`
	IsActive bool   `db:"is_active"`
}

type ModelCredential struct {
	ID        int       `db:"id"`
	ModelID   string    `db:"model_id"`
	APIKey    string    `db:"api_key"`
	Label     string    `db:"label"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
}

type UserQuota struct {
	ID          int       `db:"id"`
	UserID      string    `db:"user_id"`
	ModelID     string    `db:"model_id"`
	QuotaTokens int64     `db:"quota_tokens"`
	UsedTokens  int64     `db:"used_tokens"`
	ResetPeriod string    `db:"reset_period"`
	ResetDate   time.Time `db:"reset_date"`
}

func (q *UserQuota) Remaining() int64 {
	return q.QuotaTokens - q.UsedTokens
}

type ChatLog struct {
	ID              uuid.UUID `db:"id"`
	UserID          string    `db:"user_id"`
	SessionID       uuid.UUID `db:"session_id"`
	ModelID         string    `db:"model_id"`
	RequestAt       time.Time `db:"request_at"`
	ResponseAt      time.Time `db:"response_at"`
	RequestMessages []byte    `db:"request_messages"` // JSONB
	ResponseContent string    `db:"response_content"`
	InputTokens     int       `db:"input_tokens"`
	OutputTokens    int       `db:"output_tokens"`
	Status          string    `db:"status"` // success | quota_exceeded | error
	ErrorMessage    string    `db:"error_message"`
	CredentialID    *int      `db:"credential_id"` // backend account used for this request
}

// Chat request/response types shared across handlers and proxy

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model     string    `json:"model" binding:"required"`
	Messages  []Message `json:"messages" binding:"required"`
	SessionID string    `json:"session_id"`
	Stream    bool      `json:"stream"`
}

type ChatResponse struct {
	Content string     `json:"content"`
	Usage   TokenUsage `json:"usage"`
}

type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}