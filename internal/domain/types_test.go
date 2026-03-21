package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---- UserQuota.Remaining ----

func TestUserQuota_Remaining(t *testing.T) {
	cases := []struct {
		name     string
		quota    int64
		used     int64
		expected int64
	}{
		{"normal", 1000, 400, 600},
		{"exactly zero", 500, 500, 0},
		{"over-deducted", 100, 150, -50},
		{"zero quota zero used", 0, 0, 0},
		{"large values", 10_000_000_000, 3_000_000_000, 7_000_000_000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q := &UserQuota{QuotaTokens: c.quota, UsedTokens: c.used}
			if got := q.Remaining(); got != c.expected {
				t.Errorf("Remaining() = %d, want %d", got, c.expected)
			}
		})
	}
}

// ---- ChatLog ----

func TestChatLog_IDIsUUID(t *testing.T) {
	id := uuid.New()
	l := ChatLog{ID: id}
	if l.ID != id {
		t.Errorf("ChatLog.ID mismatch: got %v, want %v", l.ID, id)
	}
}

func TestChatLog_CredentialID_NilByDefault(t *testing.T) {
	var l ChatLog
	if l.CredentialID != nil {
		t.Errorf("CredentialID should be nil by default, got %v", l.CredentialID)
	}
}

func TestChatLog_CredentialID_Settable(t *testing.T) {
	id := 42
	l := ChatLog{CredentialID: &id}
	if *l.CredentialID != 42 {
		t.Errorf("CredentialID = %d, want 42", *l.CredentialID)
	}
}

func TestChatLog_RequestMessagesIsBytes(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "hello"}}
	raw, _ := json.Marshal(msgs)
	l := ChatLog{RequestMessages: raw}

	var parsed []Message
	if err := json.Unmarshal(l.RequestMessages, &parsed); err != nil {
		t.Fatalf("failed to unmarshal RequestMessages: %v", err)
	}
	if len(parsed) != 1 || parsed[0].Role != "user" || parsed[0].Content != "hello" {
		t.Errorf("unexpected parsed messages: %v", parsed)
	}
}

func TestChatLog_StatusValues(t *testing.T) {
	for _, status := range []string{"success", "quota_exceeded", "error"} {
		l := ChatLog{Status: status}
		if l.Status != status {
			t.Errorf("Status = %q, want %q", l.Status, status)
		}
	}
}

// ---- TokenUsage ----

func TestTokenUsage_Fields(t *testing.T) {
	u := TokenUsage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}
	if u.InputTokens != 10 || u.OutputTokens != 5 || u.TotalTokens != 15 {
		t.Errorf("TokenUsage fields incorrect: %+v", u)
	}
}

func TestTokenUsage_TotalIsIndependent(t *testing.T) {
	// TotalTokens is a stored field, not automatically computed from In+Out.
	u := TokenUsage{InputTokens: 10, OutputTokens: 5, TotalTokens: 99}
	if u.TotalTokens != 99 {
		t.Errorf("TotalTokens should be stored as-is, got %d", u.TotalTokens)
	}
}

// ---- Message / ChatRequest / ChatResponse ----

func TestMessage_RoleAndContent(t *testing.T) {
	m := Message{Role: "assistant", Content: "hello"}
	if m.Role != "assistant" || m.Content != "hello" {
		t.Errorf("Message fields incorrect: %+v", m)
	}
}

func TestChatRequest_StreamDefaultsFalse(t *testing.T) {
	r := ChatRequest{Model: "gpt-4o", Messages: []Message{{Role: "user", Content: "hi"}}}
	if r.Stream {
		t.Error("Stream should default to false")
	}
}

func TestChatRequest_JSONRoundTrip(t *testing.T) {
	orig := ChatRequest{
		Model:     "gpt-4o",
		Messages:  []Message{{Role: "user", Content: "test"}},
		SessionID: "sess-1",
		Stream:    true,
	}
	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ChatRequest
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Model != orig.Model || got.SessionID != orig.SessionID || !got.Stream {
		t.Errorf("ChatRequest round-trip mismatch: %+v", got)
	}
}

func TestChatResponse_JSONRoundTrip(t *testing.T) {
	orig := ChatResponse{
		Content: "answer",
		Usage:   TokenUsage{InputTokens: 5, OutputTokens: 3, TotalTokens: 8},
	}
	b, _ := json.Marshal(orig)
	var got ChatResponse
	_ = json.Unmarshal(b, &got)
	if got.Content != "answer" || got.Usage.InputTokens != 5 {
		t.Errorf("ChatResponse round-trip mismatch: %+v", got)
	}
}

// ---- Model ----

func TestModel_IsActiveField(t *testing.T) {
	m := Model{ID: "gpt-4o", Name: "GPT-4o", Provider: "openai", IsActive: true}
	if !m.IsActive {
		t.Error("IsActive should be true")
	}
}

// ---- ModelCredential ----

func TestModelCredential_Fields(t *testing.T) {
	now := time.Now()
	c := ModelCredential{
		ID:        1,
		ModelID:   "gpt-4o",
		APIKey:    "sk-test",
		Label:     "prod",
		IsActive:  true,
		CreatedAt: now,
	}
	if c.ID != 1 || c.APIKey != "sk-test" || !c.CreatedAt.Equal(now) {
		t.Errorf("ModelCredential fields incorrect: %+v", c)
	}
}

// ---- User ----

func TestUser_Fields(t *testing.T) {
	u := User{ID: "u1", Email: "u1@example.com", Name: "Alice"}
	if u.ID != "u1" || u.Email != "u1@example.com" || u.Name != "Alice" {
		t.Errorf("User fields incorrect: %+v", u)
	}
}
